package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"avenue/backend/db"
	"avenue/backend/logger"
	"avenue/backend/sdk"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
)

func (s *Server) ForgotPassword(c *gin.Context) {
	var req sdk.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	// Rate limit before touching keyring so the check can't be used to
	// distinguish registered from unregistered emails by timing/behavior.
	// Still returns 204 on limit so the response looks identical either way.
	if !forgotPasswordIPLimiter.allow(c.ClientIP()) || !forgotPasswordEmailLimiter.allow(normalizeEmailKey(req.Email)) {
		c.Status(http.StatusNoContent)
		return
	}

	if err := s.keyringClient("").ForgotPassword(c.Request.Context(), req.Email); err != nil {
		logger.Errorf("forgot password: %v", err)
	}

	c.Status(http.StatusNoContent)
}

func (s *Server) ResetPassword(c *gin.Context) {
	var req sdk.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	if err := s.keyringClient("").ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		respond(c, http.StatusBadRequest, "", errors.New("invalid or expired reset token"))
		return
	}

	c.Status(http.StatusNoContent)
}

// AdminSendPasswordReset emails a password-reset link to the target user.
// Keyring has no admin-triggered variant of this, so it goes through the
// same public ForgotPassword flow keyring's own self-serve reset uses.
func (s *Server) AdminSendPasswordReset(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	token, err := shared.GetSessionIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusBadRequest, "", errors.New("session token not found"))
		return
	}

	targetLocalID, err := strconv.ParseInt(c.Param("userID"), 10, 64)
	if err != nil {
		respond(c, http.StatusBadRequest, "", errors.New("invalid user id"))
		return
	}

	targetLocal, err := db.GetLocalUserByID(targetLocalID)
	if err != nil {
		respond(c, http.StatusNotFound, "", errors.New("user not found"))
		return
	}

	kc := s.keyringClient(token)
	target, err := findKeyringUserByID(c.Request.Context(), kc, targetLocal.KeyringUserID)
	if err != nil {
		respond(c, http.StatusNotFound, "", errors.New("user not found"))
		return
	}

	if err := kc.ForgotPassword(c.Request.Context(), target.Email); err != nil {
		respond(c, http.StatusInternalServerError, "", fmt.Errorf("send reset email: %w", err))
		return
	}

	c.Status(http.StatusNoContent)
}
