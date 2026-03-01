package db

import (
	"time"

	"github.com/google/uuid"
)

type File struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Extension string    `json:"extension"`
	MimeType  string    `json:"mimeType"`
	FileSize  int64     `json:"file_size"`
	Parent    string    `json:"parent"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

func CreateFile(file *File) (string, error) {
	if file.ID == "" {
		file.ID = uuid.NewString()
	}
	_, err := DB.Exec(`
		INSERT INTO files (id, name, extension, mime_type, file_size, parent, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now())
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
		SELECT id, name, extension, mime_type, file_size, parent, created_by, created_at
		FROM files WHERE id=$1 AND created_by=$2
	`, id, creatorID).Scan(&f.ID, &f.Name, &f.Extension, &f.MimeType, &f.FileSize, &f.Parent, &f.CreatedBy, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func ListFiles(creatorID string) ([]File, error) {
	rows, err := DB.Query(`
		SELECT id, name, extension, mime_type, file_size, parent, created_by, created_at
		FROM files WHERE created_by=$1
	`, creatorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.Name, &f.Extension, &f.MimeType, &f.FileSize, &f.Parent, &f.CreatedBy, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func DeleteFile(id, creatorID string) error {
	_, err := DB.Exec(`DELETE FROM files WHERE id=$1 AND created_by=$2`, id, creatorID)
	return err
}

func ListChildFile(parentID, creatorID string) ([]File, error) {
	rows, err := DB.Query(`
		SELECT id, name, extension, mime_type, file_size, parent, created_by, created_at
		FROM files WHERE parent=$1 AND created_by=$2
	`, parentID, creatorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.Name, &f.Extension, &f.MimeType, &f.FileSize, &f.Parent, &f.CreatedBy, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func UpdateFile(f File) error {
	_, err := DB.Exec(`
		UPDATE files SET name=$2, extension=$3, mime_type=$4, file_size=$5, parent=$6
		WHERE id=$1
	`, f.ID, f.Name, f.Extension, f.MimeType, f.FileSize, f.Parent)
	return err
}
