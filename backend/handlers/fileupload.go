package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/afero"
)

type UploadReq struct {
	Name      string `json:"name" binding:"required"`
	Extension int    `json:"extension"  binding:"required"`
	Data      string `json:"data" binding:"required"`
}

func (s *Server) Upload(c *gin.Context) {
	var req UploadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"message": "could not marshal all data to json",
		})
	}
	afero.WriteFile()
	// afero.WriteFile(os.Stdout, []byte(req.Data), 0644)
}
