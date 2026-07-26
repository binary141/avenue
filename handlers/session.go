package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"avenue/backend/db"
	"avenue/backend/sdk"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
)

// ListSessions lists the authenticated user's active sessions, marking the
// one used to make this request so the UI can avoid letting the user
// accidentally revoke the session they're currently using.
func (s *Server) ListSessions(c *gin.Context) {
	ctx := c.Request.Context()
	userIDStr, err := shared.GetUserIDFromContext(ctx)
	if err != nil {
		respond(c, http.StatusBadRequest, errors.New("user id not found"))
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		respond(c, http.StatusInternalServerError, errors.New("invalid user id"))
		return
	}

	currentSessionID, _ := shared.GetSessionIDFromContext(ctx)

	sessions, err := db.ListSessionsForUser(userID)
	if err != nil {
		respond(c, http.StatusInternalServerError, fmt.Errorf("list sessions: %w", err))
		return
	}

	out := make([]sdk.SessionInfo, 0, len(sessions))
	for _, sess := range sessions {
		out = append(out, sdk.SessionInfo{
			ID:        sess.ID,
			CreatedAt: sess.CreatedAt,
			ExpiresAt: time.Unix(sess.ExpiresAt, 0),
			UserAgent: sess.UserAgent,
			IPAddress: sess.IPAddress,
			IsCurrent: sess.SessionID == currentSessionID,
		})
	}

	c.JSON(http.StatusOK, out)
}

// RevokeSession revokes a single session belonging to the authenticated
// user. Revoking the session making this request is rejected — use Logout
// for that instead.
func (s *Server) RevokeSession(c *gin.Context) {
	ctx := c.Request.Context()
	userIDStr, err := shared.GetUserIDFromContext(ctx)
	if err != nil {
		respond(c, http.StatusBadRequest, errors.New("user id not found"))
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		respond(c, http.StatusInternalServerError, errors.New("invalid user id"))
		return
	}

	sessionID, err := strconv.ParseInt(c.Param("sessionID"), 10, 64)
	if err != nil {
		respond(c, http.StatusBadRequest, errors.New("invalid session id"))
		return
	}

	currentSessionID, _ := shared.GetSessionIDFromContext(ctx)
	sessions, err := db.ListSessionsForUser(userID)
	if err != nil {
		respond(c, http.StatusInternalServerError, fmt.Errorf("list sessions: %w", err))
		return
	}
	for _, sess := range sessions {
		if sess.ID == sessionID && sess.SessionID == currentSessionID {
			respond(c, http.StatusBadRequest, errors.New("cannot revoke your current session; use logout instead"))
			return
		}
	}

	if err := db.DeleteSessionByIDForUser(sessionID, userID); err != nil {
		respond(c, http.StatusInternalServerError, fmt.Errorf("revoke session: %w", err))
		return
	}

	c.Status(http.StatusNoContent)
}

// RevokeOtherSessions revokes every session for the authenticated user
// except the one making this request.
func (s *Server) RevokeOtherSessions(c *gin.Context) {
	ctx := c.Request.Context()
	userIDStr, err := shared.GetUserIDFromContext(ctx)
	if err != nil {
		respond(c, http.StatusBadRequest, errors.New("user id not found"))
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		respond(c, http.StatusInternalServerError, errors.New("invalid user id"))
		return
	}

	currentSessionID, err := shared.GetSessionIDFromContext(ctx)
	if err != nil {
		respond(c, http.StatusBadRequest, errors.New("current session not found"))
		return
	}

	if err := db.DeleteOtherSessions(userID, currentSessionID); err != nil {
		respond(c, http.StatusInternalServerError, fmt.Errorf("revoke other sessions: %w", err))
		return
	}

	c.Status(http.StatusNoContent)
}
