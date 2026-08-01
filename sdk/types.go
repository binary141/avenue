package sdk

import "time"

// These mirror the JSON shapes returned by the db package's types
// (db.File, db.Folder, db.User, db.ShareLink, ...) so this package has no
// dependency on the server's db/sqlx internals.

type File struct {
	ID        int64      `json:"id"`
	UUID      string     `json:"uuid"`
	Name      string     `json:"name"`
	Extension string     `json:"extension"`
	MimeType  string     `json:"mimeType"`
	FileSize  int64      `json:"file_size"`
	Checksum  string     `json:"checksum,omitempty"`
	Parent    string     `json:"parent"`
	CreatedBy int64      `json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type Folder struct {
	ID        int64      `json:"id"`
	UUID      string     `json:"uuid"`
	Name      string     `json:"name"`
	ParentID  int64      `json:"parent_id"`
	OwnerID   int64      `json:"owner_id"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// FolderItem is a single row from a unified folder+file listing query —
// used wherever folders and files must be paginated together as one
// deterministically-ordered set (drive listings, trash listings) instead of
// as two separately-paginated result sets. Type is either "folder" or
// "file"; fields that don't apply to a folder row (Extension, MimeType,
// Checksum, CreatedBy) are simply left zero-valued.
type FolderItem struct {
	Type      string     `json:"type"`
	ID        int64      `json:"id"`
	UUID      string     `json:"uuid"`
	Name      string     `json:"name"`
	Extension string     `json:"extension,omitempty"`
	MimeType  string     `json:"mimeType,omitempty"`
	FileSize  int64      `json:"file_size"`
	Checksum  string     `json:"checksum,omitempty"`
	ParentID  int64      `json:"parent_id,omitempty"`
	OwnerID   int64      `json:"owner_id,omitempty"`
	CreatedBy int64      `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

const (
	FolderItemTypeFolder = "folder"
	FolderItemTypeFile   = "file"
)

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

type SessionInfo struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	UserAgent string    `json:"userAgent"`
	IPAddress string    `json:"ipAddress"`
	IsCurrent bool      `json:"isCurrent"`
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

type V1OAuthProvidersResponse struct {
	Providers []string `json:"providers"`
}

type V1DashboardResponse struct {
	MaxFileSize          int64 `json:"maxFileSize"`
	FileSharingEnabled   bool  `json:"fileSharingEnabled"`
	FolderSharingEnabled bool  `json:"folderSharingEnabled"`
}

// V1FolderContentsResponse lists the contents of a folder as a single,
// unified, already-paginated Items list (folders and files interleaved by
// the requested sort), so the caller doesn't need to reconcile two
// separately-paginated result sets to know what page it's on.
type V1FolderContentsResponse struct {
	Items       []FolderItem `json:"items"`
	BreadCrumbs []Breadcrumb `json:"breadcrumbs"`
	Page        int          `json:"page"`
	Limit       int          `json:"limit"`
	Total       int          `json:"total"`
}

// V1TrashResponse lists the top-level trashed items for a user, i.e. the
// files and folders the user explicitly trashed, as a single unified,
// already-paginated Items list. Items that are only in the trash because an
// ancestor folder was trashed are not listed separately — they come back
// along with their parent on restore.
type V1TrashResponse struct {
	Items []FolderItem `json:"items"`
	// RetentionDays is how many days an item sits in the trash before the
	// sweeper permanently deletes it.
	RetentionDays int `json:"retentionDays"`
	Page          int `json:"page"`
	Limit         int `json:"limit"`
	Total         int `json:"total"`
}

// V1RestoreResponse is returned by the file/folder restore endpoints
// (single and bulk) so the UI can refresh its trash pagination totals
// without re-fetching the whole list.
type V1RestoreResponse struct {
	Message string `json:"message"`
	Total   int    `json:"total"`
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
