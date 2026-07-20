package sdk

import "time"

type LoginRequest struct {
	Email    string `json:"email" binding:"required,min=4,max=64"`
	Password string `json:"password" binding:"required,min=4,max=64"`
}

type RegisterRequest struct {
	Password  string `json:"password" binding:"required,min=4,max=64"`
	FirstName string `json:"firstName" binding:"max=64"`
	LastName  string `json:"lastName" binding:"max=64"`
	Email     string `json:"email" binding:"required,min=4,max=512"`
}

type CreateUserRequest struct {
	Email     string  `json:"email" binding:"required,min=4,max=512"`
	Password  *string `json:"password,omitempty" binding:"omitempty,min=4,max=64"`
	FirstName string  `json:"firstName" binding:"min=1,max=64"`
	LastName  string  `json:"lastName" binding:"min=1,max=64"`
	IsAdmin   bool    `json:"isAdmin"`
	SendEmail bool    `json:"sendEmail"`
}

type UpdateProfileRequest struct {
	ID        int64   `json:"id" binding:"required,min=1"`
	Email     *string `json:"email,omitempty" binding:"omitempty,email,min=4,max=512"`
	IsAdmin   *bool   `json:"isAdmin,omitempty"`
	Password  *string `json:"password,omitempty" binding:"omitempty,min=4,max=64"`
	FirstName *string `json:"firstName,omitempty" binding:"min=0,max=64"`
	LastName  *string `json:"lastName,omitempty" binding:"min=0,max=64"`
	Quota     *int64  `json:"quota,omitempty" binding:"omitempty,min=0"`
}

type UpdatePasswordRequest struct {
	Password string `json:"password" binding:"required,min=8,max=128"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

type CreateFolderRequest struct {
	Name   string `json:"name" binding:"required"`
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
