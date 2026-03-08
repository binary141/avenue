package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"avenue/backend/db"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
	"github.com/spf13/afero"
)

type createShareLinkReq struct {
	ExpiresAt *time.Time `json:"expires_at"`
}

type shareLinkResponse struct {
	Token     string     `json:"token"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type shareLinkMetaResponse struct {
	FileName  string     `json:"file_name"`
	FileSize  int64      `json:"file_size"`
	MimeType  string     `json:"mime_type"`
	ExpiresAt *time.Time `json:"expires_at"`
	Token     string     `json:"token"`
}

func (s *Server) CreateShareLink(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	fileID := c.Param("fileID")

	_, err = db.GetFileByID(fileID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, Response{Message: "file not found"})
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

	link, err := db.CreateShareLink(fileID, userID, req.ExpiresAt)
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

func (s *Server) GetShareLinkMeta(c *gin.Context) {
	token := c.Param("token")

	link, err := db.GetShareLink(token)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{Message: "share link not found or expired"})
		return
	}

	file, err := db.GetFileByIDPublic(link.FileID)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{Message: "file not found"})
		return
	}

	c.JSON(http.StatusOK, shareLinkMetaResponse{
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
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	fileID := c.Param("fileID")
	_, err = db.GetFileByID(fileID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, Response{Message: "file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	links, err := db.ListSharesByFile(fileID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}
	if links == nil {
		links = []db.ShareLink{}
	}
	c.JSON(http.StatusOK, links)
}

func (s *Server) ListUserShares(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	links, err := db.ListSharesByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}
	if links == nil {
		links = []db.ShareLinkWithFileName{}
	}
	c.JSON(http.StatusOK, links)
}

func (s *Server) RevokeShareLink(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	token := c.Param("token")
	if err := db.DeleteShareLink(token, userID); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func (s *Server) DownloadSharedFile(c *gin.Context) {
	token := c.Param("token")

	link, err := db.GetShareLink(token)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{Message: "share link not found or expired"})
		return
	}

	file, err := db.GetFileByIDPublic(link.FileID)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{Message: "file not found"})
		return
	}

	path := fmt.Sprintf("/%d/%s", link.CreatedBy, file.ID)
	fileData, err := s.fs.Open(path)
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			c.JSON(http.StatusNotFound, Response{Message: "file not found on disk"})
			return
		}
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
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
