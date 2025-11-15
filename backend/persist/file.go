package persist

import (
	"time"

	"github.com/google/uuid"
)

type File struct {
	ID         string `gorm:"primaryKey, type:uuid"`
	Name       string `gorm:"not null"`
	Extension  string `gorm:"not null"`
	FileSize   int    `gorm:"column:file_size"`
	Parent     string
	CreatedAt  time.Time
	DeleteTime time.Time
}

// CreateFile creates a new file record in the database.
func (p *Persist) CreateFile(file *File) (string, error) {
	if file.ID == "" {
		file.ID = uuid.NewString()
	}
	return file.ID, p.db.Create(file).Error
}

// GetFileByID retrieves a file by its ID.
func (p *Persist) GetFileByID(id string) (*File, error) {
	var file File
	err := p.db.Where("id = ?", id).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// GetAllFiles retrieves all files from the database.
func (p *Persist) ListFiles() ([]File, error) {
	var files []File
	err := p.db.Find(&files).Error
	return files, err
}

// DeleteFile deletes a file by its ID.
func (p *Persist) DeleteFile(id string) error {
	return p.db.Where("id = ?", id).Delete(&File{}).Error
}

func (p *Persist) ListChildFile(parentId string) ([]File, error) {
	var f []File
	err := p.db.Where("parent = ?").Find(f).Error
	return f, err
}

func (p *Persist) UpdateFile(f File, mask []string) error {
	return p.db.Model(&File{}).Where("id = ?", f.ID).Select(mask).Updates(f).Error
}
