package handlers

import (
	"avenue/backend/persist"
	"avenue/backend/shared"
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/spf13/afero"
)

type UploadReq struct {
	Name      string `json:"name" binding:"required"`
	Extension string `json:"extension"  binding:"required"`
	Data      string `json:"data" binding:"required"`
	Parent    string `json:"parent"`
}
type Response struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

func (s *Server) Upload(c *gin.Context) {

	// TODO: stream file uploads
	// TODO: file size
	userId, err := shared.GetUserIdFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}
	var req UploadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Message: "could not marshal all data to json",
			Error:   err.Error(),
		})
		return
	}
	fileId, err := s.persist.CreateFile(&persist.File{
		Name:      req.Name,
		Extension: req.Extension,
		Parent:    req.Parent,
		// Path:      filePath,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not create file record",
			Error:   err.Error(),
		})
		return
	}
	exists, err := afero.DirExists(s.fs, fmt.Sprint("/%s", userId))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "error ",
			Error:   err.Error(),
		})
		return
	}
	if !exists {
		err := s.fs.Mkdir(fmt.Sprintf("/%s", userId), os.ModePerm)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Message: "error could not make dir",
				Error:   err.Error(),
			})
			return
		}
	}

	filepath := fmt.Sprintf("/%s/%s", userId, fileId)
	f, err := s.fs.Create(filepath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not create file",
			Error:   err.Error(),
		})
		return
	}
	defer f.Close()
	size, err := f.Write([]byte(req.Data))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not write to file",
			Error:   err.Error(),
		})
		return
	}

	err = s.persist.UpdateFile(persist.File{
		ID:       fileId,
		FileSize: size,
	}, []string{"file_size"})
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not update file size",
			Error:   err.Error(),
		})
		return
	}

	c.Status(http.StatusCreated)
}

func (s *Server) ListFiles(c *gin.Context) {
	files, err := s.persist.ListFiles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not list files",
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, files)
}

func (s *Server) GetFile(c *gin.Context) {
	userId, err := shared.GetUserIdFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}
	log.Printf("user id: %s", c.Param("fileID"))
	file, err := s.persist.GetFileByID(c.Param("fileID"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not get file",
			Error:   err.Error(),
		})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	filePath := fmt.Sprintf("/%s/%s", userId, file.ID)
	log.Printf("getting file: %v", filePath)
	fileData, err := s.fs.Open(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
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
			c.JSON(http.StatusInternalServerError, Response{
				Message: "error reading bufio",
				Error:   err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, file)
}

func (s *Server) DeleteFile(c *gin.Context) {
	userId, err := shared.GetUserIdFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}
	f, err := s.persist.GetFileByID(c.Param("fileID"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "error getting file",
			Error:   err.Error(),
		})
		return
	}
	if err = s.fs.Remove(fmt.Sprintf("/%s/%s", userId, f.ID)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "error deleting file from file system",
			Error:   err.Error(),
		})
		return
	}

	if err = s.persist.DeleteFile(c.Param("fileID")); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "error deleting file from db",
			Error:   err.Error(),
		})
		return
	}

	c.Status(http.StatusOK)
}
