package handlers

import (
	"database/sql"
	"errors"
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
		respond(c, http.StatusInternalServerError, "could not get user id", err)
		return
	}
	var req sdk.CreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond(c, http.StatusBadRequest, "could not marshal all data to json", err)
		return
	}

	var parentID int64
	if req.Parent != "" {
		parentFolder, err := db.GetFolder(req.Parent, userID)
		if err != nil {
			respond(c, http.StatusBadRequest, "parent folder must exist", err)
			return
		}
		parentID = parentFolder.ID
	}

	ownerIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}

	_, err = db.CreateFolder(&sdk.Folder{
		Name:     req.Name,
		OwnerID:  ownerIDInt,
		ParentID: parentID,
	})
	if err != nil {
		respond(c, http.StatusInternalServerError, "Internal server error", err)
		return
	}
	c.Status(http.StatusCreated)
}

// DeleteFolder moves a folder, and everything nested inside it, to the
// trash. Use RestoreFolder to undo, or PurgeFolder to permanently delete it.
func (s *Server) DeleteFolder(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not get user id", err)
		return
	}

	folderID := c.Param("folderID")

	err = db.TrashFolder(folderID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond(c, http.StatusNotFound, "folder not found in db", err)
			return
		}

		respond(c, http.StatusInternalServerError, "could not delete folder", err)
		return
	}

	respond(c, http.StatusOK, "Folder moved to trash", nil)
}

// BulkDelete moves a batch of files and folders to the trash in a single
// request, instead of requiring one DELETE call per item.
func (s *Server) BulkDelete(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not get user id", err)
		return
	}

	var req sdk.BulkDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond(c, http.StatusBadRequest, "could not marshal all data to json", err)
		return
	}

	if len(req.FileIDs) == 0 && len(req.FolderIDs) == 0 {
		respond(c, http.StatusBadRequest, "no file or folder ids provided", nil)
		return
	}

	if err := db.BulkTrash(req.FileIDs, req.FolderIDs, userID); err != nil {
		respond(c, http.StatusInternalServerError, "could not delete items", err)
		return
	}

	respond(c, http.StatusOK, "Items moved to trash", nil)
}

// BulkRestore restores a batch of trashed files and folders in a single
// request, instead of requiring one restore call per item.
func (s *Server) BulkRestore(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not get user id", err)
		return
	}

	var req sdk.BulkRestoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond(c, http.StatusBadRequest, "could not marshal all data to json", err)
		return
	}

	if len(req.FileIDs) == 0 && len(req.FolderIDs) == 0 {
		respond(c, http.StatusBadRequest, "no file or folder ids provided", nil)
		return
	}

	if err := db.BulkRestore(req.FileIDs, req.FolderIDs, userID); err != nil {
		respond(c, http.StatusInternalServerError, "could not restore items", err)
		return
	}

	resp, err := trashRestoreResponse(userID, "Items restored")
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not count trashed items", err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// BulkMove moves a batch of files and folders to a new parent folder in a
// single request, instead of requiring one move call per item.
func (s *Server) BulkMove(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not get user id", err)
		return
	}

	var req sdk.BulkMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond(c, http.StatusBadRequest, "could not marshal all data to json", err)
		return
	}

	if len(req.FileIDs) == 0 && len(req.FolderIDs) == 0 {
		respond(c, http.StatusBadRequest, "no file or folder ids provided", nil)
		return
	}

	if req.Parent != "" {
		if _, err := db.GetFolder(req.Parent, userID); err != nil {
			respond(c, http.StatusBadRequest, "destination folder must exist", err)
			return
		}
	}

	if err := db.BulkMove(req.FileIDs, req.FolderIDs, req.Parent, userID); err != nil {
		respond(c, http.StatusInternalServerError, "could not move items", err)
		return
	}

	respond(c, http.StatusOK, "Items moved", nil)
}

// RestoreFolder restores a trashed folder and everything nested inside it.
func (s *Server) RestoreFolder(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not get user id", err)
		return
	}

	err = db.RestoreFolder(c.Param("folderID"), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond(c, http.StatusNotFound, "folder not found in trash", err)
			return
		}

		respond(c, http.StatusInternalServerError, "could not restore folder", err)
		return
	}

	resp, err := trashRestoreResponse(userID, "Folder restored")
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not count trashed items", err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// PurgeFolder permanently deletes a trashed folder and everything nested
// inside it, removing file blobs from disk and reconciling quota usage.
func (s *Server) PurgeFolder(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not get user id", err)
		return
	}

	purgedFiles, err := db.PurgeFolder(c.Param("folderID"), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond(c, http.StatusNotFound, "folder not found in trash", err)
			return
		}

		respond(c, http.StatusInternalServerError, "could not purge folder", err)
		return
	}

	for _, f := range purgedFiles {
		if err := s.fs.Remove(shared.BlobPath(f.UUID)); err != nil && !errors.Is(err, afero.ErrFileNotFound) {
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
		respond(c, http.StatusInternalServerError, "could not get user id", err)
		return
	}

	newName := c.Param("folderName")
	if newName == "" {
		respond(c, http.StatusBadRequest, "", errors.New("folder name can't be empty"))
		return
	}

	folder, err := db.GetFolder(c.Param("folderID"), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond(c, http.StatusNotFound, "folder not found in db", err)
			return
		}

		respond(c, http.StatusInternalServerError, "could not get folder", err)
		return
	}

	folder.Name = newName

	err = db.UpdateFolder(*folder)
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not update folder", err)
		return
	}

	c.Status(http.StatusOK)
}

// buildBreadcrumbs turns a folder's ancestor chain (root-first, as returned
// by db.ListFolderParents) into the UI's breadcrumb trail (leaf-first),
// skipping the synthetic root entry when listing the root folder itself and
// appending a trailing "/" crumb (linking back to the root) when browsing a
// non-root folder.
func buildBreadcrumbs(folderID string, folderParents []sdk.Folder) []sdk.Breadcrumb {
	var crumbs []sdk.Breadcrumb

	for _, f := range folderParents {
		if folderID == "" && f.UUID == shared.ROOTFOLDERID {
			// an empty folderID in the request if for the root folder
			continue
		}

		crumbs = append(crumbs, sdk.Breadcrumb{
			Label:    f.Name,
			FolderID: f.UUID,
		})
	}

	if folderID != "" && folderID != shared.ROOTFOLDERID {
		crumbs = append(crumbs, sdk.Breadcrumb{
			Label:    "/",
			FolderID: "",
		})
	}

	slices.Reverse(crumbs)

	return crumbs
}

func (s *Server) ListFolderContents(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not get user id", err)
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
	case "type":
		// Groups folders before files regardless of sort direction, then
		// orders by name within each group using the requested direction.
		sortColumn = "(CASE WHEN item_type = 'folder' THEN 0 ELSE 1 END), name"
	}

	page, limit, offset := shared.ParsePagination(c.Query("page"), c.Query("limit"))

	itemType := c.Query("type")
	if itemType != "" && itemType != sdk.FolderItemTypeFolder && itemType != sdk.FolderItemTypeFile {
		respond(c, http.StatusBadRequest, "type must be 'folder' or 'file'", nil)
		return
	}

	folderID := c.Param("folderID")
	items, err := db.ListFolderItems(folderID, userID, sortColumn, sortDir, limit, offset, itemType)
	if err != nil {
		respond(c, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	var totalFolders, totalFiles int
	if itemType != sdk.FolderItemTypeFile {
		totalFolders, err = db.CountChildFolders(folderID, userID)
		if err != nil {
			respond(c, http.StatusInternalServerError, "Internal server error", err)
			return
		}
	}
	if itemType != sdk.FolderItemTypeFolder {
		totalFiles, err = db.CountChildFiles(folderID, userID)
		if err != nil {
			respond(c, http.StatusInternalServerError, "Internal server error", err)
			return
		}
	}

	var x sdk.V1FolderContentsResponse

	folderParents, err := db.ListFolderParents(folderID, userID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	x.Items = items
	x.Page = page
	x.Limit = limit
	x.Total = totalFolders + totalFiles

	x.BreadCrumbs = buildBreadcrumbs(folderID, folderParents)

	c.JSON(http.StatusOK, x)
}
