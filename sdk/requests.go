package sdk

import "time"

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Password  string `json:"password"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

type CreateUserRequest struct {
	Email     string  `json:"email"`
	Password  *string `json:"password,omitempty"`
	FirstName string  `json:"firstName"`
	LastName  string  `json:"lastName"`
	IsAdmin   bool    `json:"isAdmin"`
	SendEmail bool    `json:"sendEmail"`
}

type UpdateProfileRequest struct {
	ID        int64   `json:"id"`
	Email     *string `json:"email,omitempty"`
	IsAdmin   *bool   `json:"isAdmin,omitempty"`
	Password  *string `json:"password,omitempty"`
	FirstName *string `json:"firstName,omitempty"`
	LastName  *string `json:"lastName,omitempty"`
	Quota     *int64  `json:"quota,omitempty"`
}

type UpdatePasswordRequest struct {
	Password string `json:"password"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

type CreateFolderRequest struct {
	Name   string `json:"name"`
	Parent string `json:"parent"`
}

type MoveFileRequest struct {
	Parent string `json:"parent"`
}

type CreateShareLinkRequest struct {
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	RequireLogin bool       `json:"require_login"`
	AllowUpload  bool       `json:"allow_upload"`
	MaxFileSize  int64      `json:"max_file_size"`
}
