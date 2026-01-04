package handlers

import (
	"errors"
	"fmt"
	"io"
	"log"
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

func (s *Server) Upload(c *gin.Context) {
	// TODO: stream file uploads
	// TODO: file size
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	// Ensure user directory exists
	exists, err := afero.DirExists(s.fs, fmt.Sprintf("/%s", userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "error checking user directory",
			Error:   err.Error(),
		})
		return
	}
	if !exists {
		err := s.fs.Mkdir(fmt.Sprintf("/%s", userID), os.ModePerm)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Message: "error could not make dir",
				Error:   err.Error(),
			})
			return
		}
	}

	// Get uploaded file from multipart form
	f, err := c.FormFile("file")
	if err != nil {
		log.Printf("error gettting file from form: %v", err)
		c.JSON(http.StatusTeapot, Response{
			Message: "could not get file from form",
			Error:   err.Error(),
		})
		return
	}

	// Get parent folder ID from form (optional)
	parent := c.PostForm("parent")

	// Extract filename and extension
	filename := f.Filename
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))

	// Open uploaded file
	src, err := f.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Message: "could not open uploaded file",
			Error:   err.Error(),
		})
		return
	}
	defer src.Close()

	extBuffer := make([]byte, 512)
	_, err = src.Read(extBuffer)
	if err != nil && err != io.EOF {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not read file",
			Error:   err.Error(),
		})
		return
	}

	_, err = src.Seek(0, io.SeekStart)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not seek to start of file",
			Error:   err.Error(),
		})
		return
	}

	contentType := http.DetectContentType(extBuffer)

	// Create file record in database
	fileID, err := s.persist.CreateFile(&persist.File{
		Name:      filename,
		Extension: ext,
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
			c.JSON(http.StatusInternalServerError, Response{
				Message: "could not delete file in db",
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not create file",
			Error:   err.Error(),
		})
		return
	}
	defer func() {
		err := dst.Close()
		if err != nil {
			log.Println(err.Error())
		}
	}()

	// Copy file data
	size, err := io.Copy(dst, src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not write to file",
			Error:   err.Error(),
		})
		return
	}

	// Update file size in database
	err = s.persist.UpdateFile(persist.File{
		ID:       fileID,
		FileSize: int(size),
	}, []string{"file_size"})
	if err != nil {
		// what do we want to do if we cannot update the filesize?
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not update file size",
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

	c.Status(200)
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

	c.Status(http.StatusOK)
}
