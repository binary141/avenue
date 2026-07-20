package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"slices"
	"strconv"

	"avenue/backend/db"
	"avenue/backend/sdk"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
)

func (s *Server) CreateFolder(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}
	var req sdk.CreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, sdk.MessageResponse{
			Message: "could not marshal all data to json",
			Error:   err.Error(),
		})
		return
	}

	var parentID int64
	if req.Parent != "" {
		parentFolder, err := db.GetFolder(req.Parent, userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, sdk.MessageResponse{
				Message: "parent folder must exist",
				Error:   err.Error(),
			})
			return
		}
		parentID = parentFolder.ID
	}

	ownerIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{Error: err.Error()})
		return
	}

	_, err = db.CreateFolder(&sdk.Folder{
		Name:     req.Name,
		OwnerID:  ownerIDInt,
		ParentID: parentID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
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
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	folderID := c.Param("folderID")

	folds, err := db.ListChildFolder(folderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}
	files, err := db.ListChildFile(folderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}

	if len(folds) != 0 || len(files) != 0 {
		c.JSON(http.StatusBadRequest, sdk.MessageResponse{
			Message: "",
			Error:   "Folder still contains files or folders",
		})
		return
	}

	err = db.DeleteFolder(folderID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, sdk.MessageResponse{
			Message: "",
			Error:   "Folder still contains files or folders",
		})
		return
	}

	c.JSON(http.StatusOK, sdk.MessageResponse{
		Message: "Folder deleted successfully",
		Error:   "",
	})
}

func (s *Server) UpdateFolderName(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	newName := c.Param("folderName")
	if newName == "" {
		c.JSON(http.StatusBadRequest, sdk.MessageResponse{
			Error: "folder name can't be empty",
		})
		return
	}

	folder, err := db.GetFolder(c.Param("folderID"), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, sdk.MessageResponse{
				Message: "folder not found in db",
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get folder",
			Error:   err.Error(),
		})
		return
	}

	folder.Name = newName

	err = db.UpdateFolder(*folder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
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
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	folderID := c.Param("folderID")
	folds, err := db.ListChildFolder(folderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}
	files, err := db.ListChildFile(folderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}

	var x sdk.V1FolderContentsResponse

	folderParents, err := db.ListFolderParents(folderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}

	x.Folders = folds
	x.Files = files

	for _, f := range folderParents {
		if folderID == "" && f.UUID == shared.ROOTFOLDERID {
			// an empty folderID in the request if for the root folder
			continue
		}

		x.BreadCrumbs = append(x.BreadCrumbs, sdk.Breadcrumb{
			Label:    f.Name,
			FolderID: f.UUID,
		})
	}

	if folderID != "" && folderID != shared.ROOTFOLDERID {
		x.BreadCrumbs = append(x.BreadCrumbs, sdk.Breadcrumb{
			Label:    "/",
			FolderID: "",
		})
	}

	slices.Reverse(x.BreadCrumbs)

	c.JSON(http.StatusOK, x)
}
