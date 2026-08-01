package sdk

import "time"

type LoginRequest struct {
	Email    string `json:"email" binding:"required,min=4,max=64"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

type RegisterRequest struct {
	Password  string `json:"password" binding:"required,min=8,max=128"`
	FirstName string `json:"firstName" binding:"max=64"`
	LastName  string `json:"lastName" binding:"max=64"`
	Email     string `json:"email" binding:"required,min=4,max=512"`
	// Token is an admin-issued registration invite token. It's only required
	// when keyring's own self-serve registration is disabled server-side.
	Token string `json:"token"`
}

// CreateRegistrationTokenRequest issues an admin invite letting someone
// register while keyring's self-serve registration is disabled.
type CreateRegistrationTokenRequest struct {
	ExpiresInHours int `json:"expiresInHours" binding:"required,min=1"`
	MaxUses        int `json:"maxUses" binding:"omitempty,min=1"`
}

type CreateUserRequest struct {
	Email     string  `json:"email" binding:"required,min=4,max=512"`
	Password  *string `json:"password,omitempty" binding:"omitempty,min=8,max=128"`
	FirstName string  `json:"firstName" binding:"min=1,max=64"`
	LastName  string  `json:"lastName" binding:"min=1,max=64"`
	IsAdmin   bool    `json:"isAdmin"`
	SendEmail bool    `json:"sendEmail"`
	Quota     *int64  `json:"quota,omitempty" binding:"omitempty,min=0"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

type UpdateProfileRequest struct {
	ID              int64   `json:"id" binding:"required,min=1"`
	Email           *string `json:"email,omitempty" binding:"omitempty,email,min=4,max=512"`
	IsAdmin         *bool   `json:"isAdmin,omitempty"`
	Password        *string `json:"password,omitempty" binding:"omitempty,min=8,max=128"`
	CurrentPassword *string `json:"currentPassword,omitempty"`
	FirstName       *string `json:"firstName,omitempty" binding:"min=0,max=64"`
	LastName        *string `json:"lastName,omitempty" binding:"min=0,max=64"`
	Quota           *int64  `json:"quota,omitempty" binding:"omitempty,min=0"`
}

type UpdatePasswordRequest struct {
	Password        string `json:"password" binding:"required,min=8,max=128"`
	CurrentPassword string `json:"currentPassword" binding:"required"`
}

type OAuthExchangeRequest struct {
	Code string `json:"code" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=128"`
}

type CreateFolderRequest struct {
	Name   string `json:"name" binding:"required"`
	Parent string `json:"parent"`
}

type MoveFileRequest struct {
	Parent string `json:"parent"`
}

type BulkDeleteRequest struct {
	FileIDs   []string `json:"fileIds"`
	FolderIDs []string `json:"folderIds"`
}

type BulkRestoreRequest struct {
	FileIDs   []string `json:"fileIds"`
	FolderIDs []string `json:"folderIds"`
}

type BulkMoveRequest struct {
	FileIDs   []string `json:"fileIds"`
	FolderIDs []string `json:"folderIds"`
	Parent    string   `json:"parent"`
}

// DownloadFilesZipRequest selects what to bundle into a zip download. Either
// FileIDs or a single-entry FolderIDs may be set, but not both — a folder
// download walks its whole subtree and can't be combined with other items.
type DownloadFilesZipRequest struct {
	FileIDs   []string `form:"ids"`
	FolderIDs []string `form:"folderIds"`
}

type CreateShareLinkRequest struct {
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	RequireLogin bool       `json:"require_login"`
	AllowUpload  bool       `json:"allow_upload"`
	MaxFileSize  int64      `json:"max_file_size"`
}
