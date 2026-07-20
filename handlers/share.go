package handlers

import (
	"database/sql"
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
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{Error: err.Error()})
		return
	}

	fileID := c.Param("fileID")

	_, err = db.GetFileByID(fileID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, sdk.MessageResponse{Message: "file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{Error: err.Error()})
		return
	}

	var req sdk.CreateShareLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, sdk.MessageResponse{Error: err.Error()})
		return
	}

	link, err := db.CreateShareLink(fileID, userID, req.ExpiresAt, req.RequireLogin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{Error: err.Error()})
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
		c.JSON(http.StatusNotFound, sdk.MessageResponse{Message: "share link not found or expired"})
		return
	}

	if link.RequireLogin && !s.isAuthenticated(c) {
		c.JSON(http.StatusUnauthorized, sdk.MessageResponse{Message: "authentication required"})
		return
	}

	file, err := db.GetFileByIDPublic(link.FileID)
	if err != nil {
		c.JSON(http.StatusNotFound, sdk.MessageResponse{Message: "file not found"})
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
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{Error: err.Error()})
		return
	}

	fileID := c.Param("fileID")
	_, err = db.GetFileByID(fileID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, sdk.MessageResponse{Message: "file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{Error: err.Error()})
		return
	}

	links, err := db.ListSharesByFile(fileID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{Error: err.Error()})
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
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{Error: err.Error()})
		return
	}

	links, err := db.ListSharesByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{Error: err.Error()})
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
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{Error: err.Error()})
		return
	}

	links, err := db.ListExpiredSharesByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{Error: err.Error()})
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
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{Error: err.Error()})
		return
	}

	token := c.Param("token")
	if err := db.DeleteShareLink(token, userID); err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{Error: err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func (s *Server) DownloadSharedFile(c *gin.Context) {
	token := c.Param("token")

	link, err := db.GetShareLink(token)
	if err != nil {
		c.JSON(http.StatusNotFound, sdk.MessageResponse{Message: "share link not found or expired"})
		return
	}

	if link.RequireLogin && !s.isAuthenticated(c) {
		c.JSON(http.StatusUnauthorized, sdk.MessageResponse{Message: "authentication required"})
		return
	}

	file, err := db.GetFileByIDPublic(link.FileID)
	if err != nil {
		c.JSON(http.StatusNotFound, sdk.MessageResponse{Message: "file not found"})
		return
	}

	path := fmt.Sprintf("/%d/%s", link.CreatedBy, file.UUID)
	fileData, err := s.fs.Open(path)
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			c.JSON(http.StatusNotFound, sdk.MessageResponse{Message: "file not found on disk"})
			return
		}
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{Error: err.Error()})
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
