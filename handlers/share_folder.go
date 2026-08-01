package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"avenue/backend/db"
	"avenue/backend/sdk"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
	"github.com/spf13/afero"
)

// CreateFolderShareLink — POST /v1/folder/:folderID/share
func (s *Server) CreateFolderShareLink(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}

	folderID := c.Param("folderID")
	if _, ok := fetchOwnedFolder(c, folderID, userID); !ok {
		return
	}

	var req sdk.CreateShareLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		respond(c, http.StatusBadRequest, "", err)
		return
	}

	link, err := db.CreateShareFolderLink(folderID, userID, req.ExpiresAt, req.RequireLogin, req.AllowUpload, req.MaxFileSize)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}

	c.JSON(http.StatusCreated, sdk.V1ShareLinkResponse{
		Token:     link.Token,
		ExpiresAt: link.ExpiresAt,
		CreatedAt: link.CreatedAt,
	})
}

// ListFolderShares — GET /v1/folder/:folderID/shares
func (s *Server) ListFolderShares(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}

	folderID := c.Param("folderID")
	if _, ok := fetchOwnedFolder(c, folderID, userID); !ok {
		return
	}

	links, err := db.ListShareFoldersByFolder(folderID, userID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}
	if links == nil {
		links = []sdk.ShareFolderLink{}
	}
	c.JSON(http.StatusOK, links)
}

// ListUserFolderShares — GET /v1/folder-shares
func (s *Server) ListUserFolderShares(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}

	links, err := db.ListShareFoldersByUser(userID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}
	if links == nil {
		links = []sdk.ShareFolderLink{}
	}
	c.JSON(http.StatusOK, links)
}

// ListExpiredUserFolderShares — GET /v1/folder-shares/expired
func (s *Server) ListExpiredUserFolderShares(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}

	links, err := db.ListExpiredShareFoldersByUser(userID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}
	if links == nil {
		links = []sdk.ShareFolderLink{}
	}
	c.JSON(http.StatusOK, links)
}

// RevokeShareFolderLink — DELETE /v1/share/folder/:token
func (s *Server) RevokeShareFolderLink(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}

	token := c.Param("token")
	if err := db.DeleteShareFolderLink(token, userID); err != nil {
		respond(c, http.StatusInternalServerError, "", err)
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
		respond(c, http.StatusInternalServerError, "", err)
		return
	}
	ownerID := fmt.Sprint(link.CreatedBy)
	folders, err := db.ListChildFolder(link.FolderUUID, ownerID, sdk.SortAsc, shared.MAXPAGELIMIT, 0)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}

	if files == nil {
		files = []sdk.File{}
	}
	if folders == nil {
		folders = []sdk.Folder{}
	}

	c.JSON(http.StatusOK, sdk.V1SharedFolderContentsResponse{
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
		respond(c, http.StatusNotFound, "folder not found in shared tree", nil)
		return
	}

	ownerID := fmt.Sprint(link.CreatedBy)
	subFolder, err := db.GetFolder(subFolderUUID, ownerID)
	if err != nil {
		respond(c, http.StatusNotFound, "folder not found", nil)
		return
	}

	files, err := db.ListChildFilePublic(subFolderUUID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}
	folders, err := db.ListChildFolder(subFolderUUID, ownerID, sdk.SortAsc, shared.MAXPAGELIMIT, 0)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}

	if files == nil {
		files = []sdk.File{}
	}
	if folders == nil {
		folders = []sdk.Folder{}
	}

	c.JSON(http.StatusOK, sdk.V1SharedFolderContentsResponse{
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
		respond(c, http.StatusForbidden, "uploads are not allowed for this share link", nil)
		return
	}

	// Determine creator: authenticated user or folder owner
	creatorID, authed := s.getAuthenticatedUserID(c)
	if !authed {
		creatorID = link.CreatedBy
	}
	creatorIDStr := fmt.Sprint(creatorID)

	creator, err := db.GetLocalUserByIDStr(creatorIDStr)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}

	maxFileSize := effectiveMaxFileSize(link.MaxFileSize)

	if creator.Quota != 0 {
		totalUsed, err := db.GetUserUsage(creatorID)
		if err != nil {
			respond(c, http.StatusInternalServerError, "", err)
			return
		}
		if totalUsed >= creator.Quota {
			respond(c, http.StatusUnprocessableEntity, "", errors.New("creator quota reached"))
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
			respond(c, http.StatusNotFound, "target folder not found in shared tree", nil)
			return
		}
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxFileSize)

	mr, err := c.Request.MultipartReader()
	if err != nil {
		respond(c, http.StatusBadRequest, "", err)
		return
	}

	var fileID string
	var filename, extension string
	var total int64
	var checksum string
	contentType := "application/octet-stream"

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			respond(c, http.StatusBadRequest, "", err)
			return
		}

		if part.FormName() == "file" {
			filename = filepath.Base(part.FileName())
			extension = strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))

			buf := make([]byte, 512)
			n, err := io.ReadAtLeast(part, buf, 1)
			if err != nil && err != io.ErrUnexpectedEOF {
				respond(c, http.StatusInternalServerError, "", err)
				return
			}
			contentType = http.DetectContentType(buf[:n])

			fileID, err = db.CreateFile(&sdk.File{
				Name:      filename,
				Extension: extension,
				MimeType:  contentType,
				Parent:    targetFolderUUID,
				CreatedBy: creatorID,
			})
			if err != nil {
				respond(c, http.StatusInternalServerError, "", err)
				return
			}

			if err := shared.EnsureBlobDir(s.fs, fileID); err != nil {
				_ = db.DeleteFile(fileID, creatorIDStr)
				respond(c, http.StatusInternalServerError, "", err)
				return
			}

			dst, err := s.fs.Create(shared.BlobPath(fileID))
			if err != nil {
				_ = db.DeleteFile(fileID, creatorIDStr)
				respond(c, http.StatusInternalServerError, "", err)
				return
			}

			hasher := sha256.New()
			mw := io.MultiWriter(dst, hasher)

			written, err := io.Copy(mw, bytes.NewReader(buf[:n]))
			if err != nil {
				_ = db.DeleteFile(fileID, creatorIDStr)
				respond(c, http.StatusInternalServerError, "", err)
				return
			}
			total += written

			written, err = io.Copy(mw, part)
			_ = dst.Close()
			if err != nil {
				_ = db.DeleteFile(fileID, creatorIDStr)
				var maxErr *http.MaxBytesError
				if errors.As(err, &maxErr) || errors.Is(err, http.ErrBodyReadAfterClose) {
					respond(c, http.StatusRequestEntityTooLarge, "", errors.New("file too large"))
					return
				}
				respond(c, http.StatusInternalServerError, "", err)
				return
			}
			total += written
			checksum = hex.EncodeToString(hasher.Sum(nil))
		}

		_ = part.Close()
	}

	if fileID == "" {
		respond(c, http.StatusBadRequest, "no file provided", nil)
		return
	}

	if err := db.UpdateFile(sdk.File{
		UUID:      fileID,
		FileSize:  total,
		Checksum:  checksum,
		Extension: extension,
		Name:      filename,
		MimeType:  contentType,
		Parent:    targetFolderUUID,
	}, creatorIDStr); err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}

	if err := db.UpdateUsage(creatorID, total); err != nil {
		respond(c, http.StatusInternalServerError, "", err)
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
		respond(c, http.StatusNotFound, "file not found in shared folder", nil)
		return
	}

	file, err := db.GetFileByIDPublic(fileUUID)
	if err != nil {
		respond(c, http.StatusNotFound, "file not found", nil)
		return
	}

	path := shared.BlobPath(file.UUID)
	fileData, err := s.fs.Open(path)
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			respond(c, http.StatusNotFound, "file not found on disk", nil)
			return
		}
		respond(c, http.StatusInternalServerError, "", err)
		return
	}
	defer func() { _ = fileData.Close() }()

	c.Header("Content-Type", file.MimeType)
	c.Header("Content-Disposition", contentDispositionAttachment(file.Name))
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
func uploadLimitForLink(link sdk.ShareFolderLink) int64 {
	limit := effectiveMaxFileSize(link.MaxFileSize)

	owner, err := db.GetLocalUserByIDStr(fmt.Sprint(link.CreatedBy))
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
func (s *Server) resolveShareFolderLink(c *gin.Context) (sdk.ShareFolderLink, bool) {
	token := c.Param("token")
	link, err := db.GetShareFolderLink(token)
	if err != nil {
		respond(c, http.StatusNotFound, "share link not found or expired", nil)
		return sdk.ShareFolderLink{}, false
	}
	if link.RequireLogin && !s.isAuthenticated(c) {
		respond(c, http.StatusUnauthorized, "authentication required", nil)
		return sdk.ShareFolderLink{}, false
	}
	return link, true
}
