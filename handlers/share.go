package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"avenue/backend/db"
	"avenue/backend/sdk"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
	"github.com/spf13/afero"
)

func (s *Server) CreateShareLink(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}

	fileID := c.Param("fileID")

	if _, ok := fetchOwnedFile(c, fileID, userID); !ok {
		return
	}

	var req sdk.CreateShareLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		respond(c, http.StatusBadRequest, "", err)
		return
	}

	link, err := db.CreateShareLink(fileID, userID, req.ExpiresAt, req.RequireLogin)
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

func (s *Server) GetShareLinkMeta(c *gin.Context) {
	token := c.Param("token")

	link, err := db.GetShareLink(token)
	if err != nil {
		respond(c, http.StatusNotFound, "share link not found or expired", nil)
		return
	}

	if link.RequireLogin && !s.isAuthenticated(c) {
		respond(c, http.StatusUnauthorized, "authentication required", nil)
		return
	}

	file, err := db.GetFileByIDPublic(link.FileID)
	if err != nil {
		respond(c, http.StatusNotFound, "file not found", nil)
		return
	}

	c.JSON(http.StatusOK, sdk.V1ShareLinkMetaResponse{
		FileName:  file.Name,
		FileSize:  file.FileSize,
		MimeType:  file.MimeType,
		ExpiresAt: link.ExpiresAt,
		Token:     link.Token,
	})
}

func (s *Server) ListFileShares(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}

	fileID := c.Param("fileID")
	if _, ok := fetchOwnedFile(c, fileID, userID); !ok {
		return
	}

	links, err := db.ListSharesByFile(fileID, userID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}
	if links == nil {
		links = []sdk.ShareLink{}
	}
	c.JSON(http.StatusOK, links)
}

func (s *Server) ListUserShares(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}

	links, err := db.ListSharesByUser(userID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}
	if links == nil {
		links = []sdk.ShareLinkWithFileName{}
	}
	c.JSON(http.StatusOK, links)
}

func (s *Server) ListExpiredUserShares(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}

	links, err := db.ListExpiredSharesByUser(userID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}
	if links == nil {
		links = []sdk.ShareLinkWithFileName{}
	}
	c.JSON(http.StatusOK, links)
}

func (s *Server) RevokeShareLink(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}

	token := c.Param("token")
	if err := db.DeleteShareLink(token, userID); err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}
	c.Status(http.StatusOK)
}

func (s *Server) DownloadSharedFile(c *gin.Context) {
	token := c.Param("token")

	link, err := db.GetShareLink(token)
	if err != nil {
		respond(c, http.StatusNotFound, "share link not found or expired", nil)
		return
	}

	if link.RequireLogin && !s.isAuthenticated(c) {
		respond(c, http.StatusUnauthorized, "authentication required", nil)
		return
	}

	file, err := db.GetFileByIDPublic(link.FileID)
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
	defer func() {
		_ = fileData.Close()
	}()

	c.Header("Content-Type", file.MimeType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.Name))
	c.Header("Cache-Control", "no-cache")
	c.Header("Content-Length", fmt.Sprintf("%d", file.FileSize))

	c.Writer.Flush()

	if _, err := io.Copy(c.Writer, fileData); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
}
