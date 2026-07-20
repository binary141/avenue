package sdk

import "time"

// These mirror the JSON shapes returned by the db package's types
// (db.File, db.Folder, db.User, db.ShareLink, ...) so this package has no
// dependency on the server's db/sqlx internals.

type File struct {
	ID        int64     `json:"id"`
	UUID      string    `json:"uuid"`
	Name      string    `json:"name"`
	Extension string    `json:"extension"`
	MimeType  string    `json:"mimeType"`
	FileSize  int64     `json:"file_size"`
	Parent    string    `json:"parent"`
	CreatedBy int64     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type Folder struct {
	ID       int64  `json:"id"`
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	ParentID int64  `json:"parent_id"`
	OwnerID  int64  `json:"owner_id"`
}

type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Password  string    `json:"-"` // server-only: bcrypt hash, never sent to clients
	CanLogin  bool      `json:"canLogin"`
	IsAdmin   bool      `json:"isAdmin"`
	Quota     int64     `json:"quota"`
	SpaceUsed int64     `json:"spaceUsed"`
	CreatedAt time.Time `json:"createdAt"`
}

type ShareLink struct {
	ID           int64      `json:"id"`
	Token        string     `json:"token"`
	FileID       string     `json:"file_id"`
	CreatedBy    int64      `json:"created_by"`
	ExpiresAt    *time.Time `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	RequireLogin bool       `json:"require_login"`
	LastAccessed *time.Time `json:"last_accessed"`
}

type ShareLinkWithFileName struct {
	ID           int64      `json:"id"`
	Token        string     `json:"token"`
	FileID       string     `json:"file_id"`
	FileName     string     `json:"file_name"`
	CreatedBy    int64      `json:"created_by"`
	ExpiresAt    *time.Time `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	RequireLogin bool       `json:"require_login"`
	LastAccessed *time.Time `json:"last_accessed"`
}

type ShareFolderLink struct {
	ID           int64      `json:"id"`
	Token        string     `json:"token"`
	FolderUUID   string     `json:"folder_uuid"`
	FolderName   string     `json:"folder_name"`
	FolderIntID  int64      `json:"-"` // server-only: integer FK used for subtree checks
	CreatedBy    int64      `json:"created_by"`
	ExpiresAt    *time.Time `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	RequireLogin bool       `json:"require_login"`
	AllowUpload  bool       `json:"allow_upload"`
	MaxFileSize  int64      `json:"max_file_size"`
	LastAccessed *time.Time `json:"last_accessed"`
}

type Breadcrumb struct {
	Label    string `json:"label"`
	FolderID string `json:"folder_id"`
}

// -- endpoint-specific response types --

type V1LoginResponse struct {
	Message   string `json:"Message"`
	UserID    int64  `json:"User-Id"`
	SessionID string `json:"session_id"`
	UserData  User   `json:"user_data"`
}

type V1LoginMetaResponse struct {
	RegistrationEnabled string `json:"registration_enabled"`
}

type V1DashboardResponse struct {
	MaxFileSize          int64 `json:"maxFileSize"`
	FileSharingEnabled   bool  `json:"fileSharingEnabled"`
	FolderSharingEnabled bool  `json:"folderSharingEnabled"`
}

type V1FolderContentsResponse struct {
	Files       []File       `json:"files"`
	Folders     []Folder     `json:"folders"`
	BreadCrumbs []Breadcrumb `json:"breadcrumbs"`
}

type V1ShareLinkResponse struct {
	Token     string     `json:"token"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type V1ShareLinkMetaResponse struct {
	FileName  string     `json:"file_name"`
	FileSize  int64      `json:"file_size"`
	MimeType  string     `json:"mime_type"`
	ExpiresAt *time.Time `json:"expires_at"`
	Token     string     `json:"token"`
}

type V1SharedFolderContentsResponse struct {
	FolderName  string   `json:"folder_name"`
	FolderUUID  string   `json:"folder_uuid"`
	Files       []File   `json:"files"`
	Folders     []Folder `json:"folders"`
	AllowUpload bool     `json:"allow_upload"`
	MaxFileSize int64    `json:"max_file_size"`
}

// MessageResponse is the generic {message, error} envelope used by several
// endpoints (handlers.Response).
type MessageResponse struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}
