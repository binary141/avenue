package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"

	"avenue/backend/db"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
	"github.com/spf13/afero"
)

type sharedFolderContentsResponse struct {
	FolderName string       `json:"folder_name"`
	FolderUUID string       `json:"folder_uuid"`
	Files      []db.File    `json:"files"`
	Folders    []db.Folder  `json:"folders"`
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

	link, err := db.CreateShareFolderLink(folderID, userID, req.ExpiresAt, req.RequireLogin)
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

	ownerID := fmt.Sprint(link.CreatedBy)
	files, err := db.ListChildFile(link.FolderUUID, ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}
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
		FolderName: link.FolderName,
		FolderUUID: link.FolderUUID,
		Files:      files,
		Folders:    folders,
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

	files, err := db.ListChildFile(subFolderUUID, ownerID)
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
		FolderName: subFolder.Name,
		FolderUUID: subFolder.UUID,
		Files:      files,
		Folders:    folders,
	})
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
