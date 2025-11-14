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
type Response struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

func (s *Server) Upload(c *gin.Context) {
	var req UploadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, Response{
			Message: "could not marshal all data to json",
			Error:   err.Error(),
		})
		return
	}
	fs := afero.NewOsFs()
	f, err := fs.Create("temp/test.txt")
	if err != nil {
		c.JSON(500, Response{
			Message: "could not create file",
			Error:   err.Error(),
		})
		return
	}
	defer f.Close()
	_, err = f.Write([]byte(req.Data))
	if err != nil {
		c.JSON(500, Response{
			Message: "could not write to file",
			Error:   err.Error(),
		})
		return
	}
	c.JSON(200, Response{
		Message: "file uploaded successfully",
		Error:   "",
	})
}
