package handlers

import (
	"avenue/backend/logger"
	"avenue/backend/sdk"

	"github.com/gin-gonic/gin"
)

func respond(c *gin.Context, status int, err error) {
	r := sdk.MessageResponse{Error: err.Error()}

	if status >= 500 {
		logger.Errorf("HTTP %d: %s", status, r.Error)
	}

	c.AbortWithStatusJSON(status, r)
}
