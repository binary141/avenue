package handlers

import (
	"avenue/backend/logger"

	"github.com/gin-gonic/gin"
)

func respond(c *gin.Context, status int, err error) {
	r := Response{Error: err.Error()}

	if status >= 500 {
		logger.Errorf("HTTP %d: %s", status, r.Error)
	}

	c.AbortWithStatusJSON(status, r)
}
