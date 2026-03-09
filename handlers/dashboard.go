package handlers

import (
	"avenue/backend/db"
	"avenue/backend/shared"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) DashboardInfo(c *gin.Context) {
	serverMax := shared.GetEnvInt64("MAX_FILE_BYTE_SIZE", shared.DEFAULTMAXFILESIZE)
	maxFileSize := serverMax

	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err == nil {
		user, err := db.GetUserByIDStr(userID)
		if err == nil && user.Quota > 0 {
			remaining := user.Quota - user.SpaceUsed
			if remaining < maxFileSize {
				maxFileSize = remaining
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"maxFileSize": maxFileSize,
	})
}
