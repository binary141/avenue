package handlers

import (
	"net/http"

	"avenue/backend/persist"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
)

// all files live in a per user file system
// all files will be a uuid
// all files will map uuid to name extension etc in software
// add file size

// folders table
// folder will know its parent
// top level folders will have a parent of null
// files can be top level

type CreateFolderReq struct {
	Name   string `json:"name" binding:"required"`
	Parent string `json:"parent"`
}

func (s *Server) CreateFolder(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}
	var req CreateFolderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Message: "could not marshal all data to json",
			Error:   err.Error(),
		})
		return
	}

	if req.Parent != "" {
		_, err = s.persist.GetFolder(req.Parent)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Message: "parent folder must exist",
				Error:   err.Error(),
			})
			return
		}
	}

	_, err = s.persist.CreateFolder(&persist.Folder{
		Name:    req.Name,
		OwnerID: userID,
		Parent:  req.Parent,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}
	c.Status(http.StatusCreated)
}

func (s *Server) ListFolderContents(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	folderID := c.Param("folderID")
	folds, err := s.persist.ListChildFolder(folderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}
	files, err := s.persist.ListChildFile(folderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}
	var x struct {
		Files    []persist.File   `json:"files"`
		Foleders []persist.Folder `json:"folders"`
	}
	// ret := mustSet("", "folders", folds)
	// ret = mustSet(ret, "files", files)
	x.Foleders = folds
	x.Files = files
	c.JSON(http.StatusOK, x)
}

// func mustSet(json, key string, val interface{}) string {
// 	ret, err := sjson.Set(json, key, val)
// 	if err != nil {
// 		panic("this is not possible")
// 	}
// 	return ret
// }
