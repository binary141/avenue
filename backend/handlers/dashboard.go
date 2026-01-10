package handlers

import (
	"avenue/backend/shared"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) DashboardInfo(c *gin.Context) {
	maxFileSize := shared.GetEnvInt64("MAX_FILE_BYTE_SIZE", shared.DEFAULTMAXFILESIZE)

	c.JSON(http.StatusOK, gin.H{
		"maxFileSize": maxFileSize,
	})
}
