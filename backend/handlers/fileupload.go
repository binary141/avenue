package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/afero"
)

type UploadReq struct {
	Name      string `json:"name" binding:"required"`
	Extension string `json:"extension"  binding:"required"`
	Data      string `json:"data" binding:"required"`
}

func (s *Server) Upload(c *gin.Context) {
	var req UploadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"message": "could not marshal all data to json",
			"error":   err.Error(),
		})
	}
	fs := afero.NewOsFs()
	f, err := fs.Create("temp/test.txt")
	if err != nil {
		c.JSON(500, gin.H{
			"message": "could not create file",
			"error":   err.Error(),
		})
	}
	defer f.Close()
	_, err = f.Write([]byte(req.Data))
	if err != nil {
		c.JSON(500, gin.H{
			"message": "could not write to file",
			"error":   err.Error(),
		})
	}
	// afero.WriteFile(os.Stdout, []byte(req.Data), 0644)
}
