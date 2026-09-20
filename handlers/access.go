package handlers

import (
	"net/http"
	"strconv"

	"avenue/backend/db"

	"github.com/gin-gonic/gin"
)

// requireFolderAccess checks that userID has at least minAccess on the folder
// (see db.FolderAccess). A user with no access at all — including for a folder
// that doesn't exist — gets a 404 so folder ids can't be probed; a user who
// can see it but not do this to it gets a 403. On failure it writes the
// response and returns ok=false; otherwise it returns the user's access level.
func requireFolderAccess(c *gin.Context, folderID, userID string, minAccess int) (level int, ok bool) {
	level, err := db.FolderAccess(folderID, userID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not check folder access", err)
		return 0, false
	}
	return level, checkAccess(c, level, minAccess, "folder")
}

// requireFileAccess is requireFolderAccess for a file.
func requireFileAccess(c *gin.Context, fileID, userID string, minAccess int) (level int, ok bool) {
	level, err := db.FileAccess(fileID, userID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not check file access", err)
		return 0, false
	}
	return level, checkAccess(c, level, minAccess, "file")
}

func checkAccess(c *gin.Context, level, minAccess int, kind string) bool {
	switch {
	case level == db.AccessNone:
		respond(c, http.StatusNotFound, kind+" not found", nil)
		return false
	case level < minAccess:
		respond(c, http.StatusForbidden, "you don't have permission to do that to this "+kind, nil)
		return false
	}
	return true
}

// resolveUploadOwner works out who owns a new file being uploaded into
// parent (a folder UUID, or "" for the caller's own root) and checks the
// caller is allowed to put it there: they need edit access to the folder.
//
// The owner is the caller for their own drive, and the folder's owner for a
// folder shared with the caller — the file lives in the owner's tree, so it's
// recorded as theirs and counts against their quota, not the uploader's.
// On failure it writes the response and returns ok=false.
func resolveUploadOwner(c *gin.Context, userID, parent string) (ownerID int64, ok bool) {
	if parent == "" {
		id, err := strconv.ParseInt(userID, 10, 64)
		if err != nil {
			respond(c, http.StatusInternalServerError, "", err)
			return 0, false
		}
		return id, true
	}

	if _, ok := requireFolderAccess(c, parent, userID, db.AccessEditor); !ok {
		return 0, false
	}

	ownerID, err := db.FolderOwnerID(parent)
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not get folder owner", err)
		return 0, false
	}
	return ownerID, true
}

// requireRestructure checks that userID may trash or move the file or folder
// (see db.CanRestructureFile), on top of whatever access check the caller
// already made. Editors can restructure things inside a shared folder, but
// not the shared item itself, which stays the owner's to trash or move.
func requireRestructure(c *gin.Context, can func(id, userID string) (bool, error), id, userID, kind string) bool {
	ok, err := can(id, userID)
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not check "+kind+" access", err)
		return false
	}
	if !ok {
		respond(c, http.StatusForbidden, "only the owner can move or delete a "+kind+" that was shared with you directly", nil)
		return false
	}
	return true
}
