package handlers

import (
	"avenue/backend/persist"
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/spf13/afero"
)

type UploadReq struct {
	Name      string `json:"name" binding:"required"`
	Extension string `json:"extension"  binding:"required"`
	Data      string `json:"data" binding:"required"`
	// Path      string `json:"path" binding:"required"`
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

	filePath := fmt.Sprintf("temp/")
	fileName := fmt.Sprintf("%s.%s", req.Name, req.Extension)
	f, err := fs.Create(filePath + fileName)
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
	err = s.persist.CreateFile(&persist.File{
		Name:      req.Name,
		Extension: req.Extension,
		Path:      filePath,
	})
	if err != nil {
		c.JSON(500, Response{
			Message: "could not create file record",
			Error:   err.Error(),
		})
		// we failed to create the file in the db may as well delete the file from the filesystem
		fs.Remove(filePath + fileName)
		return
	}
	c.JSON(200, Response{
		Message: "file uploaded successfully",
		Error:   "",
	})
}

func (s *Server) ListFiles(c *gin.Context) {
	files, err := s.persist.ListFiles()
	if err != nil {
		c.JSON(500, Response{
			Message: "could not list files",
			Error:   err.Error(),
		})
		return
	}
	c.JSON(200, files)
}

func (s *Server) GetFile(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("fileID"))
	if err != nil {
		c.JSON(400, Response{
			Message: "could not convert ascii to int",
			Error:   err.Error(),
		})
	}
	file, err := s.persist.GetFileByID(id)
	if err != nil {
		c.JSON(500, Response{
			Message: "could not get file",
			Error:   err.Error(),
		})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	filePath := fmt.Sprintf("%s%s.%s", file.Path, file.Name, file.Extension)
	log.Printf("getting file: %v", filePath)
	fileData, err := s.fs.Open(filePath)
	if err != nil {
		c.JSON(500, Response{
			Message: "could not read file",
			Error:   err.Error(),
		})
		return
	}
	defer fileData.Close()

	f := bufio.NewReader(fileData)
	b := make([]byte, 4096)
	for {
		n, err := f.Read(b)
		if n > 0 {
			c.SSEvent("data", b)
		}
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			c.JSON(500, Response{
				Message: "error reading bufio",
				Error:   err.Error(),
			})
			return
		}
	}

	c.JSON(200, file)
}
