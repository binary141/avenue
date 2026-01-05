package persist

import (
	"avenue/backend/shared"
	"time"

	"github.com/google/uuid"
)

type File struct {
	ID        string    `gorm:"primaryKey, type:uuid" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Extension string    `gorm:"not null" json:"extension"`
	MimeType  string    `gorm:"not null" json:"mimeType"`
	FileSize  int64     `gorm:"column:file_size" json:"file_size"`
	Parent    string    `json:"parent"`
	CreatedBy string    `gorm:"column:created_by;type:bigint" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	DeletedAt time.Time `json:"deleted_at"`
}

// CreateFile creates a new file record in the database.
func (p *Persist) CreateFile(file *File) (string, error) {
	if file.ID == "" {
		file.ID = uuid.NewString()
	}
	return file.ID, p.db.Create(file).Error
}

func (p *Persist) GetFileByID(id string, creatorID string) (*File, error) {
	var file File
	err := p.db.Where("id = ?", id).Where("created_by = ?", creatorID).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (p *Persist) ListFiles(creatorID string) ([]File, error) {
	var files []File
	err := p.db.Where("created_by = ?", creatorID).Find(&files).Error
	return files, err
}

func (p *Persist) DeleteFile(id string, creatorID string) error {
	return p.db.Where("id = ?", id).Where("created_by = ?", creatorID).Delete(&File{}).Error
}

func (p *Persist) ListChildFile(parentID string, creatorID string) ([]File, error) {
	var f []File
	db := p.db
	if parentID != shared.ROOTFOLDERID {
		db = db.Where("parent = ?", parentID)
	} else {
		db = db.Where("parent = ''")
	}
	err := db.Where("created_by = ?", creatorID).Find(&f).Error
	return f, err
}

func (p *Persist) UpdateFile(f File, mask []string) error {
	return p.db.Model(&File{}).Where("id = ?", f.ID).Select(mask).Updates(f).Error
}
