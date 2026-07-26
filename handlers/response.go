package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"avenue/backend/db"
	"avenue/backend/logger"
	"avenue/backend/sdk"

	"github.com/gin-gonic/gin"
)

func respond(c *gin.Context, status int, message string, err error) {
	r := sdk.MessageResponse{Message: message}

	if err != nil {
		r.Error = err.Error()
	}

	if status >= 500 {
		logger.Errorf("HTTP %d: %s: %s", status, r.Message, r.Error)
	}

	c.AbortWithStatusJSON(status, r)
}

// fetchOwnedFile fetches the file identified by fileID, requiring that
// userID own it. On failure it writes the 404/500 response and returns
// ok=false.
func fetchOwnedFile(c *gin.Context, fileID, userID string) (*sdk.File, bool) {
	file, err := db.GetFileByID(fileID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond(c, http.StatusNotFound, "file not found", nil)
			return nil, false
		}
		respond(c, http.StatusInternalServerError, "", err)
		return nil, false
	}
	return file, true
}

// fetchOwnedFolder fetches the folder identified by folderID, requiring that
// userID own it. On failure it writes the 404/500 response and returns
// ok=false.
func fetchOwnedFolder(c *gin.Context, folderID, userID string) (*sdk.Folder, bool) {
	folder, err := db.GetFolder(folderID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond(c, http.StatusNotFound, "folder not found", nil)
			return nil, false
		}
		respond(c, http.StatusInternalServerError, "", err)
		return nil, false
	}
	return folder, true
}
