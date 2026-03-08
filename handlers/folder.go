package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"slices"
	"strconv"

	"avenue/backend/db"
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
		_, err = db.GetFolder(req.Parent, userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Message: "parent folder must exist",
				Error:   err.Error(),
			})
			return
		}
	}

	ownerIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	_, err = db.CreateFolder(&db.Folder{
		Name:    req.Name,
		OwnerID: ownerIDInt,
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

	folds, err := db.ListChildFolder(folderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}
	files, err := db.ListChildFile(folderID, userID)
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

	err = db.DeleteFolder(folderID, userID)
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

func (s *Server) UpdateFolderName(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	newName := c.Param("folderName")
	if newName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Error: "folder name can't be empty",
		})
		return
	}

	folder, err := db.GetFolder(c.Param("folderID"), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, Response{
				Message: "folder not found in db",
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not get folder",
			Error:   err.Error(),
		})
		return
	}

	folder.Name = newName

	err = db.UpdateFolder(*folder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "could not update folder",
			Error:   err.Error(),
		})
		return
	}

	c.Status(http.StatusOK)
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
	folds, err := db.ListChildFolder(folderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}
	files, err := db.ListChildFile(folderID, userID)
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
		Files       []db.File     `json:"files"`
		Folders     []db.Folder   `json:"folders"`
		BreadCrumbs []Breadcrumb  `json:"breadcrumbs"`
	}

	folderParents, err := db.ListFolderParents(folderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}

	x.Folders = folds
	x.Files = files

	for _, f := range folderParents {
		if folderID == "" && f.FolderID == shared.ROOTFOLDERID {
			// an empty folderID in the request if for the root folder
			continue
		}

		x.BreadCrumbs = append(x.BreadCrumbs, Breadcrumb{
			Label:    f.Name,
			FolderID: f.FolderID,
		})
	}

	if folderID != "" && folderID != shared.ROOTFOLDERID {
		x.BreadCrumbs = append(x.BreadCrumbs, Breadcrumb{
			Label:    "/",
			FolderID: "",
		})
	}

	slices.Reverse(x.BreadCrumbs)

	c.JSON(http.StatusOK, x)
}
