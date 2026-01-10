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
		_, err = s.persist.GetFolder(req.Parent, userID)
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

func (s *Server) DeleteFolder(c *gin.Context) {
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

	if len(folds) != 0 || len(files) != 0 {
		c.JSON(http.StatusBadRequest, Response{
			Message: "",
			Error:   "Folder still contains files or folders",
		})
		return
	}

	err = s.persist.DeleteFolder(folderID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Message: "",
			Error:   "Folder still contains files or folders",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Message: "Folder deleted successfully",
		Error:   "",
	})
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

	type Breadcrumb struct {
		Label    string `json:"label"`
		FolderID string `json:"folder_id"`
	}

	var x struct {
		Files       []persist.File   `json:"files"`
		Folders     []persist.Folder `json:"folders"`
		BreadCrumbs []Breadcrumb     `json:"breadcrumbs"`
	}

	folderParents, err := s.persist.ListFolderParents(folderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}

	x.Folders = folds
	x.Files = files

	for i, f := range folderParents {
		// skip the first folder as that is the one that is being queried
		if i == 0 {
			continue
		}

		x.BreadCrumbs = append(x.BreadCrumbs, Breadcrumb{
			Label:    f.Name,
			FolderID: f.FolderID,
		})
	}

	if folderID != shared.ROOTFOLDERID {
		x.BreadCrumbs = append(x.BreadCrumbs, Breadcrumb{
			Label:    "/",
			FolderID: shared.ROOTFOLDERID,
		})
	}

	c.JSON(http.StatusOK, x)
}

// func mustSet(json, key string, val interface{}) string {
// 	ret, err := sjson.Set(json, key, val)
// 	if err != nil {
// 		panic("this is not possible")
// 	}
// 	return ret
// }
