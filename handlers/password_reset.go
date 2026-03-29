package handlers

import (
	"bytes"
	"embed"
	"errors"
	"html/template"
	"net/http"

	"avenue/backend/db"
	"avenue/backend/email"
	"avenue/backend/logger"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

//go:embed templates/forgot_password.html
var forgotPasswordTmplFS embed.FS

var forgotPasswordTmpl = template.Must(
	template.New("forgot_password.html").ParseFS(forgotPasswordTmplFS, "templates/forgot_password.html"),
)

func forgotPasswordHTML(resetURL string) string {
	var buf bytes.Buffer
	if err := forgotPasswordTmpl.Execute(&buf, resetURL); err != nil {
		return resetURL
	}
	return buf.String()
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

func (s *Server) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	user, err := db.GetUserByEmail(req.Email)
	if err != nil {
		// Return success regardless so we don't leak which emails are registered.
		c.Status(http.StatusNoContent)
		return
	}

	if !user.CanLogin {
		c.Status(http.StatusNoContent)
		return
	}

	token, err := db.CreatePasswordResetToken(user.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{Error: "could not create reset token"})
		return
	}

	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	resetURL := scheme + "://" + c.Request.Host + "/reset-password?token=" + token

	if err := email.Send(email.Message{
		To:      user.Email,
		Subject: "Reset your Avenue password",
		HTML:    forgotPasswordHTML(resetURL),
		Text:    "You requested a password reset for your Avenue account.\n\nClick the link below to set a new password:\n\n" + resetURL + "\n\nThis link expires in 1 hour. If you did not request this, you can safely ignore this email.",
	}); err != nil {
		if !errors.Is(err, email.NotConfigured) {
			logger.Errorf("email(forgot password): %v", err)
		}
	}

	c.Status(http.StatusNoContent)
}

func (s *Server) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	userID, err := db.ConsumePasswordResetToken(req.Token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, Response{Error: "invalid or expired reset token"})
		return
	}

	user, err := db.GetUserByID(userID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{Error: "user not found"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{Error: "could not hash password"})
		return
	}

	if err := db.UpdatePassword(userID, string(hashed)); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, Response{Error: "could not update password"})
		return
	}

	if err := email.Send(email.Message{
		To:      user.Email,
		Subject: "Your password was changed",
		Text:    "Your Avenue account password was just changed via a password reset. If you did not make this change, please contact your administrator immediately.",
	}); err != nil {
		logger.Errorf("email(reset password confirmation): %v", err)
	}

	c.Status(http.StatusNoContent)
}
