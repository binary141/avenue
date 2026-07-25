package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"avenue/backend/db"
	"avenue/backend/logger"
	"avenue/backend/sdk"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
	"github.com/spf13/afero"
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

// DeleteFolder moves a folder, and everything nested inside it, to the
// trash. Use RestoreFolder to undo, or PurgeFolder to permanently delete it.
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

	err = db.TrashFolder(folderID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, sdk.MessageResponse{
				Message: "folder not found in db",
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not delete folder",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, sdk.MessageResponse{
		Message: "Folder moved to trash",
		Error:   "",
	})
}

// RestoreFolder restores a trashed folder and everything nested inside it.
func (s *Server) RestoreFolder(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	err = db.RestoreFolder(c.Param("folderID"), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, sdk.MessageResponse{
				Message: "folder not found in trash",
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not restore folder",
			Error:   err.Error(),
		})
		return
	}

	c.Status(http.StatusOK)
}

// PurgeFolder permanently deletes a trashed folder and everything nested
// inside it, removing file blobs from disk and reconciling quota usage.
func (s *Server) PurgeFolder(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not get user id",
			Error:   err.Error(),
		})
		return
	}

	purgedFiles, err := db.PurgeFolder(c.Param("folderID"), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, sdk.MessageResponse{
				Message: "folder not found in trash",
				Error:   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "could not purge folder",
			Error:   err.Error(),
		})
		return
	}

	for _, f := range purgedFiles {
		if err := s.fs.Remove(fmt.Sprintf("/%d/%s", f.CreatedBy, f.UUID)); err != nil && !errors.Is(err, afero.ErrFileNotFound) {
			logger.Errorf("purge folder: remove file blob %s: %v", f.UUID, err)
		}
		if err := db.UpdateUsage(f.CreatedBy, -f.FileSize); err != nil {
			logger.Errorf("purge folder: update usage for user %d: %v", f.CreatedBy, err)
		}
	}

	c.Status(http.StatusOK)
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

	sortDir := sdk.SortAsc
	if c.Query("sort") == string(sdk.SortDesc) {
		sortDir = sdk.SortDesc
	}

	sortColumn := "name"
	switch c.Query("sortBy") {
	case "size":
		sortColumn = "file_size"
	case "date":
		sortColumn = "created_at"
	}

	folderID := c.Param("folderID")
	folds, err := db.ListChildFolder(folderID, userID, sortDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, sdk.MessageResponse{
			Message: "Internal server error",
			Error:   err.Error(),
		})
		return
	}
	files, err := db.ListChildFile(folderID, userID, sortColumn, sortDir)
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
