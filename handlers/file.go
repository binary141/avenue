package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"avenue/backend/persist"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
	"github.com/spf13/afero"
	"gorm.io/gorm"
)

type UploadReq struct {
	Name      string `json:"name" binding:"required"`
	Extension string `json:"extension"  binding:"required"`
	Data      string `json:"data" binding:"required"`
	Parent    string `json:"parent"`
}
type Response struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

func ensureDir(fs afero.Fs, path string) error {
	exists, err := afero.DirExists(fs, path)
	if err != nil {
		return err
	}
	if !exists {
		err := fs.Mkdir(path, os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) Upload(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	user, err := s.persist.GetUserByIDStr(userID)
	if err != nil {
		log.Printf("error getting user: %s", err.Error())
		c.JSON(http.StatusInternalServerError, Response{
			Error: err.Error(),
		})
		return
	}

	maxFileSize := shared.GetEnvInt64("MAX_FILE_BYTE_SIZE", shared.DEFAULTMAXFILESIZE)
	var total int64

	// 0 is unlimited
	if user.Quota != 0 {
		totalUsed, err := s.persist.GetUserUsage(userID)
		if err != nil {
			log.Printf("error getting user quota: %s", err.Error())
			c.JSON(http.StatusInternalServerError, Response{
				Error: err.Error(),
			})
			return
		}

		if totalUsed >= user.Quota {
			c.JSON(http.StatusUnprocessableEntity, Response{
				Error: "Max quota reached. Please delete files to be able to upload files",
			})
			return
		}

		remainingQuota := user.Quota - totalUsed

		maxFileSize = int64(math.Min(float64(remainingQuota), float64(maxFileSize)))
	}

	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		maxFileSize,
	)

	err = ensureDir(s.fs, fmt.Sprintf("/%s", userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not ensure dir exists",
			Error:   err.Error(),
		})
		return
	}

	mr, err := c.Request.MultipartReader()
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Message: "invalid multipart request",
			Error:   err.Error(),
		})
		return
	}

	var parent string
	var filename string
	var extension string
	var fileID string
	var fileSeen bool

	contentType := "application/octet-stream" // will be overwritten with the actual content type once we start streaming the file data

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Message: "multipart read error",
				Error:   err.Error(),
			})
			return
		}

		switch part.FormName() {
		case "parent":
			buf, err := io.ReadAll(io.LimitReader(part, 1024))
			if err != nil {
				c.JSON(http.StatusInternalServerError, Response{
					Message: "Unable to read multi part bytes for parent",
					Error:   err.Error(),
				})
				return
			}
			parent = string(buf)
		case "file":
			// only allow one file upload for now
			if fileSeen {
				c.JSON(http.StatusBadRequest, Response{Message: "only one file allowed"})
				return
			}
			fileSeen = true
			filename = filepath.Base(part.FileName())
			extension = strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))

			// Detect content type (read first 512 bytes) only if this is the first part
			buf := make([]byte, 512)
			n, err := io.ReadAtLeast(part, buf, 1)
			if err != nil && err != io.ErrUnexpectedEOF {
				c.JSON(http.StatusInternalServerError, Response{
					Message: "Unable to read multi part bytes",
					Error:   err.Error(),
				})
				return
			}
			contentType = http.DetectContentType(buf[:n])

			// Create file record in database
			fileID, err = s.persist.CreateFile(&persist.File{
				Name:      filename,
				Extension: extension,
				MimeType:  contentType,
				Parent:    parent,
				CreatedBy: userID,
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, Response{
					Message: "could not create file record",
					Error:   err.Error(),
				})
				return
			}

			// Create destination file
			dstPath := fmt.Sprintf("/%s/%s", userID, fileID)
			dst, err := s.fs.Create(dstPath)
			if err != nil {
				deleteErr := s.persist.DeleteFile(fileID, userID)
				if deleteErr != nil {
					log.Println(deleteErr)
					c.JSON(http.StatusInternalServerError, Response{
						Message: "could not delete file in db",
						Error:   deleteErr.Error(),
					})
					return
				}

				c.JSON(http.StatusInternalServerError, Response{
					Message: "could not create file",
					Error:   err.Error(),
				})
				return
			}

			r := bytes.NewReader(buf)
			written, err := io.Copy(dst, r)
			if err != nil {
				deleteErr := s.persist.DeleteFile(fileID, userID)
				if deleteErr != nil {
					log.Println(deleteErr)
					c.JSON(http.StatusInternalServerError, Response{
						Message: "could not delete file in db",
						Error:   deleteErr.Error(),
					})
					return
				}

				c.JSON(http.StatusInternalServerError, Response{
					Message: "Unable to read multi part bytes",
					Error:   err.Error(),
				})
				return
			}
			total += written

			written, err = io.Copy(dst, part)
			if err != nil {
				deleteErr := s.persist.DeleteFile(fileID, userID)
				if deleteErr != nil {
					log.Println(deleteErr)
					c.JSON(http.StatusInternalServerError, Response{
						Message: "could not delete file in db",
						Error:   deleteErr.Error(),
					})
					return
				}

				var maxErr *http.MaxBytesError
				if errors.As(err, &maxErr) {
					c.JSON(http.StatusRequestEntityTooLarge, Response{
						Error: "File too large",
					})
					return
				}

				if errors.Is(err, http.ErrBodyReadAfterClose) {
					c.JSON(http.StatusRequestEntityTooLarge, Response{
						Error: "File too large",
					})
					return
				}

				c.JSON(http.StatusInternalServerError, Response{
					Message: "Unable to read multi part bytes",
					Error:   err.Error(),
				})
				return
			}

			err = dst.Close()
			if err != nil {
				log.Println(err.Error())
			}
			total += written
		}

		err = part.Close()
		if err != nil {
			log.Println(err)
		}
	}

	if fileID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Message: "no file provided",
		})
		return
	}

	// Update file size in database
	err = s.persist.UpdateFile(persist.File{
		ID:        fileID,
		FileSize:  total,
		Extension: extension,
		Name:      filename,
		MimeType:  contentType,
		Parent:    parent,
	}, []string{"file_size", "extension", "name", "parent", "mime_type"})
	if err != nil {
		// what do we want to do if we cannot update the filesize?
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not update file size",
			Error:   err.Error(),
		})
		return
	}

	err = s.persist.UpdateUsage(userID, total)
	if err != nil {
		// todo should we rollback? Or just have a cron that'll reconcile?
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not update user quota usage",
			Error:   err.Error(),
		})
		return
	}

	c.Status(http.StatusCreated)
}

func (s *Server) ListFiles(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	files, err := s.persist.ListFiles(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not list files",
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, files)
}

func (s *Server) UpdateFileName(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	newName := c.Param("fileName")
	if newName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Message: "filename can't be empty",
		})
		return
	}

	file, err := s.persist.GetFileByID(c.Param("fileID"), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, Response{
				Message: "file not found in db",
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not get file",
			Error:   err.Error(),
		})
		return
	}

	file.Name = newName

	err = s.persist.UpdateFile(*file, []string{"name"})
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not update file",
			Error:   err.Error(),
		})
		return
	}

	c.Status(http.StatusOK)
}

func (s *Server) GetFile(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	file, err := s.persist.GetFileByID(c.Param("fileID"), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, Response{
				Message: "file not found in db",
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not get file",
			Error:   err.Error(),
		})
		return
	}

	path := fmt.Sprintf("/%s/%s", userID, file.ID)
	fileData, err := s.fs.Open(path)
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			c.JSON(http.StatusNotFound, Response{
				Message: "could not find file in fs",
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not open file",
			Error:   err.Error(),
		})
		return
	}
	defer func() {
		_ = fileData.Close()
	}()

	// ----- Streaming Download Headers -----
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.Name))

	c.Header("Cache-Control", "no-cache")
	c.Header("Access-Control-Expose-Headers", "Content-Disposition")
	c.Header("Content-Length", fmt.Sprintf("%d", file.FileSize))

	c.Writer.Flush()

	// ----- Stream file to client -----
	if _, err := io.Copy(c.Writer, fileData); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
}

func (s *Server) DeleteFile(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}
	f, err := s.persist.GetFileByID(c.Param("fileID"), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "error getting file",
			Error:   err.Error(),
		})
		return
	}

	if err = s.fs.Remove(fmt.Sprintf("/%s/%s", userID, f.ID)); err != nil {
		// only error if the file was found
		// if the file wasn't found, we still want to delete from the system
		if !errors.Is(err, afero.ErrFileNotFound) {
			c.JSON(http.StatusInternalServerError, Response{
				Message: "error deleting file from file system",
				Error:   err.Error(),
			})
			return
		}
	}

	if err = s.persist.DeleteFile(c.Param("fileID"), userID); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "error deleting file from db",
			Error:   err.Error(),
		})
		return
	}

	err = s.persist.UpdateUsage(userID, -f.FileSize)
	if err != nil {
		// todo should we rollback? Or just have a cron that'll reconcile?
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not update user quota usage",
			Error:   err.Error(),
		})
		return
	}

	c.Status(http.StatusOK)
}
