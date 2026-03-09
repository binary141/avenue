package handlers

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"avenue/backend/db"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
	"github.com/spf13/afero"
)

type sharedFolderContentsResponse struct {
	FolderName  string      `json:"folder_name"`
	FolderUUID  string      `json:"folder_uuid"`
	Files       []db.File   `json:"files"`
	Folders     []db.Folder `json:"folders"`
	AllowUpload bool        `json:"allow_upload"`
	MaxFileSize int64       `json:"max_file_size"`
}

// CreateFolderShareLink — POST /v1/folder/:folderID/share
func (s *Server) CreateFolderShareLink(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	folderID := c.Param("folderID")
	_, err = db.GetFolder(folderID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, Response{Message: "folder not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	var req createShareLinkReq
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
		return
	}

	link, err := db.CreateShareFolderLink(folderID, userID, req.ExpiresAt, req.RequireLogin, req.AllowUpload, req.MaxFileSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, shareLinkResponse{
		Token:     link.Token,
		ExpiresAt: link.ExpiresAt,
		CreatedAt: link.CreatedAt,
	})
}

// ListFolderShares — GET /v1/folder/:folderID/shares
func (s *Server) ListFolderShares(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	folderID := c.Param("folderID")
	_, err = db.GetFolder(folderID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, Response{Message: "folder not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	links, err := db.ListShareFoldersByFolder(folderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}
	if links == nil {
		links = []db.ShareFolderLink{}
	}
	c.JSON(http.StatusOK, links)
}

// ListUserFolderShares — GET /v1/folder-shares
func (s *Server) ListUserFolderShares(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	links, err := db.ListShareFoldersByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}
	if links == nil {
		links = []db.ShareFolderLink{}
	}
	c.JSON(http.StatusOK, links)
}

// ListExpiredUserFolderShares — GET /v1/folder-shares/expired
func (s *Server) ListExpiredUserFolderShares(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	links, err := db.ListExpiredShareFoldersByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}
	if links == nil {
		links = []db.ShareFolderLink{}
	}
	c.JSON(http.StatusOK, links)
}

// RevokeShareFolderLink — DELETE /v1/share/folder/:token
func (s *Server) RevokeShareFolderLink(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	token := c.Param("token")
	if err := db.DeleteShareFolderLink(token, userID); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

// GetSharedFolderContents — GET /share/folder/:token
func (s *Server) GetSharedFolderContents(c *gin.Context) {
	link, ok := s.resolveShareFolderLink(c)
	if !ok {
		return
	}

	files, err := db.ListChildFilePublic(link.FolderUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}
	ownerID := fmt.Sprint(link.CreatedBy)
	folders, err := db.ListChildFolder(link.FolderUUID, ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	if files == nil {
		files = []db.File{}
	}
	if folders == nil {
		folders = []db.Folder{}
	}

	c.JSON(http.StatusOK, sharedFolderContentsResponse{
		FolderName:  link.FolderName,
		FolderUUID:  link.FolderUUID,
		Files:       files,
		Folders:     folders,
		AllowUpload: link.AllowUpload,
		MaxFileSize: uploadLimitForLink(link),
	})
}

// BrowseSharedSubFolder — GET /share/folder/:token/browse/:subFolderUUID
func (s *Server) BrowseSharedSubFolder(c *gin.Context) {
	link, ok := s.resolveShareFolderLink(c)
	if !ok {
		return
	}

	subFolderUUID := c.Param("subFolderUUID")
	inTree, err := db.IsFolderInSubtree(link.FolderIntID, subFolderUUID)
	if err != nil || !inTree {
		c.JSON(http.StatusNotFound, Response{Message: "folder not found in shared tree"})
		return
	}

	ownerID := fmt.Sprint(link.CreatedBy)
	subFolder, err := db.GetFolder(subFolderUUID, ownerID)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{Message: "folder not found"})
		return
	}

	files, err := db.ListChildFilePublic(subFolderUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}
	folders, err := db.ListChildFolder(subFolderUUID, ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	if files == nil {
		files = []db.File{}
	}
	if folders == nil {
		folders = []db.Folder{}
	}

	c.JSON(http.StatusOK, sharedFolderContentsResponse{
		FolderName:  subFolder.Name,
		FolderUUID:  subFolder.UUID,
		Files:       files,
		Folders:     folders,
		AllowUpload: link.AllowUpload,
		MaxFileSize: uploadLimitForLink(link),
	})
}

// UploadToSharedFolder — POST /share/folder/:token/upload
func (s *Server) UploadToSharedFolder(c *gin.Context) {
	link, ok := s.resolveShareFolderLink(c)
	if !ok {
		return
	}

	if !link.AllowUpload {
		c.JSON(http.StatusForbidden, Response{Message: "uploads are not allowed for this share link"})
		return
	}

	// Determine creator: authenticated user or folder owner
	creatorID, authed := s.getAuthenticatedUserID(c)
	if !authed {
		creatorID = link.CreatedBy
	}
	creatorIDStr := fmt.Sprint(creatorID)

	creator, err := db.GetUserByIDStr(creatorIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	maxFileSize := effectiveMaxFileSize(link.MaxFileSize)

	if creator.Quota != 0 {
		totalUsed, err := db.GetUserUsage(creatorID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
			return
		}
		if totalUsed >= creator.Quota {
			c.JSON(http.StatusUnprocessableEntity, Response{Error: "creator quota reached"})
			return
		}
		remaining := creator.Quota - totalUsed
		if remaining < maxFileSize {
			maxFileSize = remaining
		}
	}

	// Determine target folder (query param ?folder=<uuid>, must be in shared subtree)
	targetFolderUUID := c.Query("folder")
	if targetFolderUUID == "" {
		targetFolderUUID = link.FolderUUID
	} else {
		inTree, err := db.IsFolderInSubtree(link.FolderIntID, targetFolderUUID)
		if err != nil || !inTree {
			c.JSON(http.StatusNotFound, Response{Message: "target folder not found in shared tree"})
			return
		}
	}

	if err := ensureDir(s.fs, fmt.Sprintf("/%s", creatorIDStr)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxFileSize)

	mr, err := c.Request.MultipartReader()
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
		return
	}

	var fileID string
	var filename, extension string
	var total int64
	contentType := "application/octet-stream"

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
			return
		}

		if part.FormName() == "file" {
			filename = filepath.Base(part.FileName())
			extension = strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))

			buf := make([]byte, 512)
			n, err := io.ReadAtLeast(part, buf, 1)
			if err != nil && err != io.ErrUnexpectedEOF {
				c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
				return
			}
			contentType = http.DetectContentType(buf[:n])

			fileID, err = db.CreateFile(&db.File{
				Name:      filename,
				Extension: extension,
				MimeType:  contentType,
				Parent:    targetFolderUUID,
				CreatedBy: creatorID,
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
				return
			}

			dst, err := s.fs.Create(fmt.Sprintf("/%s/%s", creatorIDStr, fileID))
			if err != nil {
				_ = db.DeleteFile(fileID, creatorIDStr)
				c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
				return
			}

			written, err := io.Copy(dst, bytes.NewReader(buf[:n]))
			if err != nil {
				_ = db.DeleteFile(fileID, creatorIDStr)
				c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
				return
			}
			total += written

			written, err = io.Copy(dst, part)
			_ = dst.Close()
			if err != nil {
				_ = db.DeleteFile(fileID, creatorIDStr)
				var maxErr *http.MaxBytesError
				if errors.As(err, &maxErr) || errors.Is(err, http.ErrBodyReadAfterClose) {
					c.JSON(http.StatusRequestEntityTooLarge, Response{Error: "file too large"})
					return
				}
				c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
				return
			}
			total += written
		}

		_ = part.Close()
	}

	if fileID == "" {
		c.JSON(http.StatusBadRequest, Response{Message: "no file provided"})
		return
	}

	if err := db.UpdateFile(db.File{
		UUID:      fileID,
		FileSize:  total,
		Extension: extension,
		Name:      filename,
		MimeType:  contentType,
		Parent:    targetFolderUUID,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	if err := db.UpdateUsage(creatorID, total); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

// DownloadSharedFolderFile — GET /share/folder/:token/file/:fileUUID
func (s *Server) DownloadSharedFolderFile(c *gin.Context) {
	link, ok := s.resolveShareFolderLink(c)
	if !ok {
		return
	}

	fileUUID := c.Param("fileUUID")
	inTree, err := db.IsFileInSubtree(link.FolderIntID, fileUUID)
	if err != nil || !inTree {
		c.JSON(http.StatusNotFound, Response{Message: "file not found in shared folder"})
		return
	}

	file, err := db.GetFileByIDPublic(fileUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{Message: "file not found"})
		return
	}

	path := fmt.Sprintf("/%d/%s", link.CreatedBy, file.UUID)
	fileData, err := s.fs.Open(path)
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			c.JSON(http.StatusNotFound, Response{Message: "file not found on disk"})
			return
		}
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}
	defer func() { _ = fileData.Close() }()

	c.Header("Content-Type", file.MimeType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.Name))
	c.Header("Cache-Control", "no-cache")
	c.Header("Content-Length", fmt.Sprintf("%d", file.FileSize))
	c.Writer.Flush()

	if _, err := io.Copy(c.Writer, fileData); err != nil {
		c.Status(http.StatusInternalServerError)
	}
}

// effectiveMaxFileSize returns linkMax if it is non-zero, otherwise the server default.
func effectiveMaxFileSize(linkMax int64) int64 {
	if linkMax > 0 {
		return linkMax
	}
	return shared.GetEnvInt64("MAX_FILE_BYTE_SIZE", shared.DEFAULTMAXFILESIZE)
}

// uploadLimitForLink returns the smallest of: the link's configured max (or server
// default) and the folder owner's remaining quota. Falls back to effectiveMaxFileSize
// if quota information cannot be retrieved.
func uploadLimitForLink(link db.ShareFolderLink) int64 {
	limit := effectiveMaxFileSize(link.MaxFileSize)

	owner, err := db.GetUserByIDStr(fmt.Sprint(link.CreatedBy))
	if err != nil || owner.Quota == 0 {
		return limit
	}

	used, err := db.GetUserUsage(link.CreatedBy)
	if err != nil {
		return limit
	}

	remaining := owner.Quota - used
	if remaining <= 0 {
		return 0
	}
	if remaining < limit {
		return remaining
	}
	return limit
}

// resolveShareFolderLink is a helper that fetches the share folder link, handles
// not-found and auth checks, and returns false (having written the response) on failure.
func (s *Server) resolveShareFolderLink(c *gin.Context) (db.ShareFolderLink, bool) {
	token := c.Param("token")
	link, err := db.GetShareFolderLink(token)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{Message: "share link not found or expired"})
		return db.ShareFolderLink{}, false
	}
	if link.RequireLogin && !s.isAuthenticated(c) {
		c.JSON(http.StatusUnauthorized, Response{Message: "authentication required"})
		return db.ShareFolderLink{}, false
	}
	return link, true
}
