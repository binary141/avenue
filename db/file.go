package db

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type File struct {
	ID        string    `json:"id"`         // uuid column
	Name      string    `json:"name"`
	Extension string    `json:"extension"`
	MimeType  string    `json:"mimeType"`
	FileSize  int64     `json:"file_size"`
	Parent    string    `json:"parent"`     // always "" — parent_id stored as BIGINT, not returned
	CreatedBy int64     `json:"created_by"` // was TEXT, now BIGINT
	CreatedAt time.Time `json:"created_at"`
}

func CreateFile(file *File) (string, error) {
	if file.ID == "" {
		file.ID = uuid.NewString()
	}
	_, err := DB.Exec(`
		INSERT INTO files (uuid, name, extension, mime_type, file_size, parent_id, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5,
			CASE WHEN $6 = '' THEN NULL
			     ELSE (SELECT id FROM folders WHERE uuid = $6)
			END,
			$7, now())
	`, file.ID, file.Name, file.Extension, file.MimeType, file.FileSize, file.Parent, file.CreatedBy)
	if err != nil {
		return "", err
	}

	if err := UpdateUsage(file.CreatedBy, file.FileSize); err != nil {
		return "", err
	}

	return file.ID, nil
}

func GetFileByID(id, creatorID string) (*File, error) {
	var f File
	err := DB.QueryRow(`
		SELECT uuid, name, extension, mime_type, file_size, created_by, created_at
		FROM files WHERE uuid=$1 AND created_by=$2::BIGINT
	`, id, creatorID).Scan(&f.ID, &f.Name, &f.Extension, &f.MimeType, &f.FileSize, &f.CreatedBy, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func GetFileByIDPublic(id string) (*File, error) {
	var f File
	err := DB.QueryRow(`
		SELECT uuid, name, extension, mime_type, file_size, created_by, created_at
		FROM files WHERE uuid=$1
	`, id).Scan(&f.ID, &f.Name, &f.Extension, &f.MimeType, &f.FileSize, &f.CreatedBy, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func ListFiles(creatorID string) ([]File, error) {
	rows, err := DB.Query(`
		SELECT uuid, name, extension, mime_type, file_size, created_by, created_at
		FROM files WHERE created_by=$1::BIGINT
	`, creatorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.Name, &f.Extension, &f.MimeType, &f.FileSize, &f.CreatedBy, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func DeleteFile(id, creatorID string) error {
	_, err := DB.Exec(`DELETE FROM files WHERE uuid=$1 AND created_by=$2::BIGINT`, id, creatorID)
	return err
}

func ListChildFile(parentID, creatorID string) ([]File, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if parentID == "" {
		rows, err = DB.Query(`
			SELECT uuid, name, extension, mime_type, file_size, created_by, created_at
			FROM files WHERE parent_id IS NULL AND created_by=$1::BIGINT
		`, creatorID)
	} else {
		rows, err = DB.Query(`
			SELECT uuid, name, extension, mime_type, file_size, created_by, created_at
			FROM files
			WHERE parent_id = (SELECT id FROM folders WHERE uuid = $1)
			  AND created_by = $2::BIGINT
		`, parentID, creatorID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.Name, &f.Extension, &f.MimeType, &f.FileSize, &f.CreatedBy, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func UpdateFile(f File) error {
	_, err := DB.Exec(`
		UPDATE files SET name=$2, extension=$3, mime_type=$4, file_size=$5,
			parent_id = CASE WHEN $6 = '' THEN NULL
			                 ELSE (SELECT id FROM folders WHERE uuid = $6)
			            END
		WHERE uuid=$1
	`, f.ID, f.Name, f.Extension, f.MimeType, f.FileSize, f.Parent)
	return err
}
