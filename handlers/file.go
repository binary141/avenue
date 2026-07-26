package handlers

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"avenue/backend/db"
	"avenue/backend/logger"
	"avenue/backend/sdk"
	"avenue/backend/shared"
	"avenue/backend/sweeper"

	"github.com/gin-gonic/gin"
	"github.com/spf13/afero"
)

func (s *Server) Upload(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	user, err := db.GetUserByIDStr(userID)
	if err != nil {
		logger.Errorf("error getting user: %s", err.Error())
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Error: err.Error(),
		})
		return
	}

	userIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{Error: err.Error()})
		return
	}

	maxFileSize := shared.GetEnvInt64("MAX_FILE_BYTE_SIZE", shared.DEFAULTMAXFILESIZE)
	var total int64

	// 0 is unlimited
	if user.Quota != 0 {
		totalUsed, err := db.GetUserUsage(userIDInt)
		if err != nil {
			logger.Errorf("error getting user quota: %s", err.Error())
			c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
				Error: err.Error(),
			})
			return
		}

		if totalUsed >= user.Quota {
			c.JSON(http.StatusUnprocessableEntity, sdk.MessageResponse{
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

	mr, err := c.Request.MultipartReader()
	if err != nil {
		c.JSON(http.StatusBadRequest, sdk.MessageResponse{
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
	var createdAt time.Time
	var checksum string

	contentType := "application/octet-stream" // will be overwritten with the actual content type once we start streaming the file data

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, sdk.MessageResponse{
				Message: "multipart read error",
				Error:   err.Error(),
			})
			return
		}

		switch part.FormName() {
		case "parent":
			buf, err := io.ReadAll(io.LimitReader(part, 1024))
			if err != nil {
				c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
					Message: "Unable to read multi part bytes for parent",
					Error:   err.Error(),
				})
				return
			}
			parent = string(buf)
		case "file":
			// only allow one file upload for now
			if fileSeen {
				c.JSON(http.StatusBadRequest, sdk.MessageResponse{Message: "only one file allowed"})
				return
			}
			fileSeen = true
			filename = filepath.Base(part.FileName())
			extension = strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))

			// Detect content type (read first 512 bytes) only if this is the first part
			buf := make([]byte, 512)
			n, err := io.ReadAtLeast(part, buf, 1)
			if err != nil && err != io.ErrUnexpectedEOF {
				c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
					Message: "Unable to read multi part bytes",
					Error:   err.Error(),
				})
				return
			}
			contentType = http.DetectContentType(buf[:n])

			// Create file record in database
			createdAt = time.Now().UTC()
			fileID, err = db.CreateFile(&sdk.File{
				Name:      filename,
				Extension: extension,
				MimeType:  contentType,
				Parent:    parent,
				CreatedBy: userIDInt,
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
					Message: "could not create file record",
					Error:   err.Error(),
				})
				return
			}

			if err := shared.EnsureBlobDir(s.fs, fileID); err != nil {
				c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
					Message: "could not ensure dir exists",
					Error:   err.Error(),
				})
				return
			}

			// Create destination file
			dstPath := shared.BlobPath(fileID)
			dst, err := s.fs.Create(dstPath)
			if err != nil {
				deleteErr := db.DeleteFile(fileID, userID)
				if deleteErr != nil {
					logger.Errorf("delete file: %v", deleteErr)
					c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
						Message: "could not delete file in db",
						Error:   deleteErr.Error(),
					})
					return
				}

				c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
					Message: "could not create file",
					Error:   err.Error(),
				})
				return
			}

			hasher := sha256.New()
			mw := io.MultiWriter(dst, hasher)

			r := bytes.NewReader(buf)
			written, err := io.Copy(mw, r)
			if err != nil {
				dst.Close()
				_ = s.fs.Remove(dstPath)
				deleteErr := db.DeleteFile(fileID, userID)
				if deleteErr != nil {
					logger.Errorf("delete file: %v", deleteErr)
					c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
						Message: "could not delete file in db",
						Error:   deleteErr.Error(),
					})
					return
				}

				c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
					Message: "Unable to read multi part bytes",
					Error:   err.Error(),
				})
				return
			}
			total += written

			written, err = io.Copy(mw, part)
			if err != nil {
				dst.Close()
				_ = s.fs.Remove(dstPath)
				deleteErr := db.DeleteFile(fileID, userID)
				if deleteErr != nil {
					logger.Errorf("delete file: %v", deleteErr)
					c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
						Message: "could not delete file in db",
						Error:   deleteErr.Error(),
					})
					return
				}

				var maxErr *http.MaxBytesError
				if errors.As(err, &maxErr) {
					c.JSON(http.StatusRequestEntityTooLarge, sdk.MessageResponse{
						Error: "File too large",
					})
					return
				}

				if errors.Is(err, http.ErrBodyReadAfterClose) {
					c.JSON(http.StatusRequestEntityTooLarge, sdk.MessageResponse{
						Error: "File too large",
					})
					return
				}

				c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
					Message: "Unable to read multi part bytes",
					Error:   err.Error(),
				})
				return
			}

			err = dst.Close()
			if err != nil {
				logger.Errorf("close upload dest: %v", err)
			}
			total += written
			checksum = hex.EncodeToString(hasher.Sum(nil))
		}

		err = part.Close()
		if err != nil {
			logger.Errorf("close upload part: %v", err)
		}
	}

	if fileID == "" {
		c.JSON(http.StatusBadRequest, sdk.MessageResponse{
			Message: "no file provided",
		})
		return
	}

	// Update file size in database
	err = db.UpdateFile(sdk.File{
		UUID:      fileID,
		FileSize:  total,
		Checksum:  checksum,
		Extension: extension,
		Name:      filename,
		MimeType:  contentType,
		Parent:    parent,
	}, userID)
	if err != nil {
		// what do we want to do if we cannot update the filesize?
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not update file size",
			Error:   err.Error(),
		})
		return
	}

	err = db.UpdateUsage(userIDInt, total)
	if err != nil {
		// todo should we rollback? Or just have a cron that'll reconcile?
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not update user quota usage",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, sdk.File{
		UUID:      fileID,
		Name:      filename,
		Extension: extension,
		MimeType:  contentType,
		FileSize:  total,
		Checksum:  checksum,
		Parent:    parent,
		CreatedBy: userIDInt,
		CreatedAt: createdAt,
	})
}

func (s *Server) ListFiles(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	files, err := db.ListFiles(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not list files",
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, files)
}

func (s *Server) SearchFiles(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	fileName := c.Param("fileName")
	if fileName == "" {
		c.JSON(http.StatusBadRequest, sdk.MessageResponse{
			Message: "search query can't be empty",
		})
		return
	}

	files, err := db.SearchChildFiles(c.Param("folderID"), userID, fileName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not search files",
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, files)
}

func (s *Server) MoveFile(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	var req sdk.MoveFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, sdk.MessageResponse{
			Message: "could not marshal all data to json",
			Error:   err.Error(),
		})
		return
	}

	file, err := db.GetFileByIDForUser(c.Param("fileID"), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, sdk.MessageResponse{
				Message: "file not found in db",
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get file",
			Error:   err.Error(),
		})
		return
	}

	if req.Parent != "" {
		if _, err := db.GetFolder(req.Parent, userID); err != nil {
			c.JSON(http.StatusBadRequest, sdk.MessageResponse{
				Message: "destination folder must exist",
				Error:   err.Error(),
			})
			return
		}
	}

	file.Parent = req.Parent

	if err := db.UpdateFile(*file, userID); err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not move file",
			Error:   err.Error(),
		})
		return
	}

	c.Status(http.StatusOK)
}

func (s *Server) UpdateFileName(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	newName := c.Param("fileName")
	if newName == "" {
		c.JSON(http.StatusBadRequest, sdk.MessageResponse{
			Message: "filename can't be empty",
		})
		return
	}

	file, err := db.GetFileByIDForUser(c.Param("fileID"), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, sdk.MessageResponse{
				Message: "file not found in db",
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get file",
			Error:   err.Error(),
		})
		return
	}

	file.Name = newName

	err = db.UpdateFile(*file, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
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
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	file, err := db.GetFileByIDForUser(c.Param("fileID"), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, sdk.MessageResponse{
				Message: "file not found in db",
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get file",
			Error:   err.Error(),
		})
		return
	}

	path := shared.BlobPath(file.UUID)
	fileData, err := s.fs.Open(path)
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			c.JSON(http.StatusNotFound, sdk.MessageResponse{
				Message: "could not find file in fs",
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
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

// DownloadFilesZip streams a zip archive of the requested files to the
// client. Files are read from disk and written into the zip one at a time
// so the archive is never buffered in memory or on disk before sending.
func (s *Server) DownloadFilesZip(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	ids := c.QueryArray("ids")
	folderIDs := c.QueryArray("folderIds")

	if len(ids) == 0 && len(folderIDs) == 0 {
		c.JSON(http.StatusBadRequest, sdk.MessageResponse{
			Message: "no file or folder ids provided",
		})
		return
	}

	// Folder downloads walk an entire subtree, so they can't be combined
	// with other files/folders in the same request — only one folder,
	// downloaded on its own, is allowed.
	if len(folderIDs) > 0 {
		if len(folderIDs) > 1 || len(ids) > 0 {
			c.JSON(http.StatusBadRequest, sdk.MessageResponse{
				Message: "a folder can't be downloaded alongside other files or folders",
			})
			return
		}

		s.downloadFolderZip(c, userID, folderIDs[0])
		return
	}

	files := make([]*sdk.File, 0, len(ids))
	for _, id := range ids {
		file, err := db.GetFileByIDForUser(id, userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, sdk.MessageResponse{
					Message: fmt.Sprintf("file not found in db: %s", id),
					Error:   err.Error(),
				})
				return
			}

			c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
				Message: "could not get file",
				Error:   err.Error(),
			})
			return
		}
		files = append(files, file)
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", `attachment; filename="download.zip"`)
	c.Header("Cache-Control", "no-cache")
	c.Header("Access-Control-Expose-Headers", "Content-Disposition")
	c.Status(http.StatusOK)
	c.Writer.Flush()

	zw := zip.NewWriter(c.Writer)
	defer func() {
		_ = zw.Close()
	}()

	names := make(map[string]int)
	for _, file := range files {
		path := shared.BlobPath(file.UUID)
		fileData, err := s.fs.Open(path)
		if err != nil {
			logger.Errorf("error opening file %s for zip download: %s", file.UUID, err.Error())
			continue
		}

		entryWriter, err := zw.CreateHeader(&zip.FileHeader{
			Name:     uniqueZipEntryName(names, file.Name),
			Method:   zip.Deflate,
			Modified: file.CreatedAt,
		})
		if err != nil {
			_ = fileData.Close()
			logger.Errorf("error creating zip entry for %s: %s", file.UUID, err.Error())
			continue
		}

		if _, err := io.Copy(entryWriter, fileData); err != nil {
			logger.Errorf("error writing file %s to zip: %s", file.UUID, err.Error())
		}
		_ = fileData.Close()
	}
}

// downloadFolderZip streams a zip archive of every file nested under
// folderID, preserving its directory structure. A single request may only
// target one folder (enforced by the caller), but multiple folder downloads
// can run concurrently across the server.
func (s *Server) downloadFolderZip(c *gin.Context, userID, folderID string) {
	folder, err := db.GetFolder(folderID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, sdk.MessageResponse{
				Message: "folder not found in db",
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get folder",
			Error:   err.Error(),
		})
		return
	}

	entries, err := db.ListFolderFilesForZip(folderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not list folder contents",
			Error:   err.Error(),
		})
		return
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, folder.Name))
	c.Header("Cache-Control", "no-cache")
	c.Header("Access-Control-Expose-Headers", "Content-Disposition")
	c.Status(http.StatusOK)
	c.Writer.Flush()

	zw := zip.NewWriter(c.Writer)
	defer func() {
		_ = zw.Close()
	}()

	names := make(map[string]int)
	for _, entry := range entries {
		path := shared.BlobPath(entry.UUID)
		fileData, err := s.fs.Open(path)
		if err != nil {
			logger.Errorf("error opening file %s for folder zip download: %s", entry.UUID, err.Error())
			continue
		}

		entryWriter, err := zw.CreateHeader(&zip.FileHeader{
			Name:     uniqueZipEntryName(names, entry.DirPath+"/"+entry.Name),
			Method:   zip.Deflate,
			Modified: entry.CreatedAt,
		})
		if err != nil {
			_ = fileData.Close()
			logger.Errorf("error creating zip entry for %s: %s", entry.UUID, err.Error())
			continue
		}

		if _, err := io.Copy(entryWriter, fileData); err != nil {
			logger.Errorf("error writing file %s to zip: %s", entry.UUID, err.Error())
		}
		_ = fileData.Close()
	}
}

// uniqueZipEntryName disambiguates duplicate file names within a single zip
// archive by appending " (n)" before the extension, mirroring how OS file
// managers handle name collisions.
func uniqueZipEntryName(seen map[string]int, name string) string {
	count := seen[name]
	seen[name] = count + 1
	if count == 0 {
		return name
	}

	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	return fmt.Sprintf("%s (%d)%s", base, count, ext)
}

// DeleteFile moves a file to the trash. The blob stays on disk and quota
// usage is untouched until the file is purged. Use RestoreFile to undo, or
// PurgeFile to permanently delete it.
func (s *Server) DeleteFile(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	if err = db.TrashFileForUser(c.Param("fileID"), userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, sdk.MessageResponse{
				Message: "file not found in db",
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "error deleting file from db",
			Error:   err.Error(),
		})
		return
	}

	c.Status(http.StatusOK)
}

// trashRestoreResponse builds the post-restore pagination totals shared by
// RestoreFile, RestoreFolder, and BulkRestore, so the UI can refresh its
// trash counts without re-fetching the whole list.
func trashRestoreResponse(userID, message string) (sdk.V1RestoreResponse, error) {
	totalFiles, err := db.CountTrashedFiles(userID)
	if err != nil {
		return sdk.V1RestoreResponse{}, err
	}
	totalFolders, err := db.CountTrashedFolders(userID)
	if err != nil {
		return sdk.V1RestoreResponse{}, err
	}
	return sdk.V1RestoreResponse{
		Message:      message,
		TotalFiles:   totalFiles,
		TotalFolders: totalFolders,
	}, nil
}

// RestoreFile restores a trashed file.
func (s *Server) RestoreFile(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	if err = db.RestoreFileForUser(c.Param("fileID"), userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, sdk.MessageResponse{
				Message: "file not found in trash",
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not restore file",
			Error:   err.Error(),
		})
		return
	}

	resp, err := trashRestoreResponse(userID, "File restored")
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not count trashed items",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// PurgeFile permanently deletes a trashed file: removes its blob from disk,
// deletes the db record, and reconciles quota usage.
func (s *Server) PurgeFile(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	f, err := db.PurgeFileForUser(c.Param("fileID"), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, sdk.MessageResponse{
				Message: "file not found in trash",
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not purge file",
			Error:   err.Error(),
		})
		return
	}

	if err = s.fs.Remove(shared.BlobPath(f.UUID)); err != nil && !errors.Is(err, afero.ErrFileNotFound) {
		logger.Errorf("purge file: remove blob %s: %v", f.UUID, err)
	}

	if err = db.UpdateUsage(f.CreatedBy, -f.FileSize); err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not update user quota usage",
			Error:   err.Error(),
		})
		return
	}

	c.Status(http.StatusOK)
}

// ListTrash lists the files and folders the user has trashed.
func (s *Server) ListTrash(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	sortDir := sdk.SortAsc
	if c.Query("sort") == string(sdk.SortDesc) {
		sortDir = sdk.SortDesc
	}

	sortColumn := "deleted_at"
	switch c.Query("sortBy") {
	case "name":
		sortColumn = "name"
	case "size":
		sortColumn = "file_size"
	}

	page, limit, offset := shared.ParsePagination(c.Query("page"), c.Query("limit"))

	files, err := db.ListTrashedFilesPage(userID, sortColumn, sortDir, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not list trashed files",
			Error:   err.Error(),
		})
		return
	}
	totalFiles, err := db.CountTrashedFiles(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not count trashed files",
			Error:   err.Error(),
		})
		return
	}

	folderSortColumn := sortColumn
	if folderSortColumn == "file_size" {
		folderSortColumn = "deleted_at"
	}

	folders, err := db.ListTrashedFoldersPage(userID, folderSortColumn, sortDir, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not list trashed folders",
			Error:   err.Error(),
		})
		return
	}
	totalFolders, err := db.CountTrashedFolders(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not count trashed folders",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, sdk.V1TrashResponse{
		Files:         files,
		Folders:       folders,
		RetentionDays: int(sweeper.Retention().Hours() / 24),
		Page:          page,
		Limit:         limit,
		TotalFiles:    totalFiles,
		TotalFolders:  totalFolders,
	})
}

// EmptyTrash permanently deletes everything the user has trashed.
func (s *Server) EmptyTrash(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	files, err := db.ListTrashedFiles(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not list trashed files",
			Error:   err.Error(),
		})
		return
	}
	for _, f := range files {
		purged, err := db.PurgeFileForUser(f.UUID, userID)
		if err != nil {
			logger.Errorf("empty trash: purge file %s: %v", f.UUID, err)
			continue
		}
		if err := s.fs.Remove(shared.BlobPath(purged.UUID)); err != nil && !errors.Is(err, afero.ErrFileNotFound) {
			logger.Errorf("empty trash: remove blob %s: %v", purged.UUID, err)
		}
		if err := db.UpdateUsage(purged.CreatedBy, -purged.FileSize); err != nil {
			logger.Errorf("empty trash: update usage for user %d: %v", purged.CreatedBy, err)
		}
	}

	folders, err := db.ListTrashedFolders(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not list trashed folders",
			Error:   err.Error(),
		})
		return
	}
	for _, folder := range folders {
		purgedFiles, err := db.PurgeFolder(folder.UUID, userID)
		if err != nil {
			logger.Errorf("empty trash: purge folder %s: %v", folder.UUID, err)
			continue
		}
		for _, f := range purgedFiles {
			if err := s.fs.Remove(shared.BlobPath(f.UUID)); err != nil && !errors.Is(err, afero.ErrFileNotFound) {
				logger.Errorf("empty trash: remove blob %s: %v", f.UUID, err)
			}
			if err := db.UpdateUsage(f.CreatedBy, -f.FileSize); err != nil {
				logger.Errorf("empty trash: update usage for user %d: %v", f.CreatedBy, err)
			}
		}
	}

	c.Status(http.StatusOK)
}
