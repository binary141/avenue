package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"avenue/backend/db"
	"avenue/backend/sdk"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
)

func validUserShareRole(role string) bool {
	return role == "viewer" || role == "editor"
}

// CreateFileUserShare shares a file with another user. Only the file's owner
// may share it.
func (s *Server) CreateFileUserShare(c *gin.Context) {
	s.createUserShare(c, sdk.FolderItemTypeFile, c.Param("fileID"))
}

// CreateFolderUserShare shares a folder, and everything inside it, with
// another user. Only the folder's owner may share it.
func (s *Server) CreateFolderUserShare(c *gin.Context) {
	s.createUserShare(c, sdk.FolderItemTypeFolder, c.Param("folderID"))
}

func (s *Server) createUserShare(c *gin.Context, resourceType, resourceID string) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not get user id", err)
		return
	}

	if !requireResourceAccess(c, resourceType, resourceID, userID, db.AccessOwner) {
		return
	}

	var req sdk.CreateUserShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond(c, http.StatusBadRequest, "could not marshal all data to json", err)
		return
	}
	if !validUserShareRole(req.Role) {
		respond(c, http.StatusBadRequest, `role must be "viewer" or "editor"`, nil)
		return
	}

	grantee, err := db.GetUserByEmail(strings.TrimSpace(req.Email))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond(c, http.StatusNotFound, "no user with that email", nil)
			return
		}
		respond(c, http.StatusInternalServerError, "could not look up user", err)
		return
	}
	if !grantee.CanLogin {
		respond(c, http.StatusNotFound, "no user with that email", nil)
		return
	}

	sharedBy, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}
	if grantee.ID == sharedBy {
		respond(c, http.StatusBadRequest, "you can't share with yourself", nil)
		return
	}

	shareID, err := db.UpsertUserShare(resourceType, resourceID, sharedBy, grantee.ID, req.Role)
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not create share", err)
		return
	}

	c.JSON(http.StatusCreated, sdk.UserShare{
		ID:               shareID,
		ResourceType:     resourceType,
		ResourceID:       resourceID,
		GranteeID:        grantee.ID,
		GranteeEmail:     grantee.Email,
		GranteeFirstName: grantee.FirstName,
		GranteeLastName:  grantee.LastName,
		Role:             req.Role,
	})
}

// ListFileUserShares lists who a file is shared with. Owner only.
func (s *Server) ListFileUserShares(c *gin.Context) {
	s.listUserShares(c, sdk.FolderItemTypeFile, c.Param("fileID"))
}

// ListFolderUserShares lists who a folder is shared with. Owner only.
func (s *Server) ListFolderUserShares(c *gin.Context) {
	s.listUserShares(c, sdk.FolderItemTypeFolder, c.Param("folderID"))
}

func (s *Server) listUserShares(c *gin.Context, resourceType, resourceID string) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not get user id", err)
		return
	}

	if !requireResourceAccess(c, resourceType, resourceID, userID, db.AccessOwner) {
		return
	}

	var shares []sdk.UserShare
	if resourceType == sdk.FolderItemTypeFolder {
		shares, err = db.ListUserSharesForFolder(resourceID)
	} else {
		shares, err = db.ListUserSharesForFile(resourceID)
	}
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not list shares", err)
		return
	}

	c.JSON(http.StatusOK, shares)
}

func requireResourceAccess(c *gin.Context, resourceType, resourceID, userID string, minAccess int) bool {
	if resourceType == sdk.FolderItemTypeFolder {
		_, ok := requireFolderAccess(c, resourceID, userID, minAccess)
		return ok
	}
	_, ok := requireFileAccess(c, resourceID, userID, minAccess)
	return ok
}

// UpdateUserShare changes a grant's role. Only the shared item's owner may.
func (s *Server) UpdateUserShare(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not get user id", err)
		return
	}

	shareID, err := strconv.ParseInt(c.Param("shareID"), 10, 64)
	if err != nil {
		respond(c, http.StatusBadRequest, "invalid share id", err)
		return
	}

	var req sdk.UpdateUserShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond(c, http.StatusBadRequest, "could not marshal all data to json", err)
		return
	}
	if !validUserShareRole(req.Role) {
		respond(c, http.StatusBadRequest, `role must be "viewer" or "editor"`, nil)
		return
	}

	if err := db.UpdateUserShareRole(shareID, userID, req.Role); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond(c, http.StatusNotFound, "share not found", nil)
			return
		}
		respond(c, http.StatusInternalServerError, "could not update share", err)
		return
	}

	c.Status(http.StatusOK)
}

// DeleteUserShare removes a grant: the shared item's owner revokes it, or the
// grantee leaves the share.
func (s *Server) DeleteUserShare(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not get user id", err)
		return
	}

	shareID, err := strconv.ParseInt(c.Param("shareID"), 10, 64)
	if err != nil {
		respond(c, http.StatusBadRequest, "invalid share id", err)
		return
	}

	if err := db.DeleteUserShare(shareID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond(c, http.StatusNotFound, "share not found", nil)
			return
		}
		respond(c, http.StatusInternalServerError, "could not delete share", err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListSharedWithMe lists the files and folders other users have shared
// directly with the caller.
func (s *Server) ListSharedWithMe(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not get user id", err)
		return
	}

	items, err := db.ListSharedWithUser(userID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not list shared items", err)
		return
	}

	c.JSON(http.StatusOK, items)
}
