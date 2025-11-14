package persist

import "time"

type File struct {
	ID        int    `gorm:"primaryKey"`
	Name      string `gorm:"not null"`
	Extension string `gorm:"not null"`
	Path      string `gorm:"not null"`
	CreatedAt time.Time
}

// CreateFile creates a new file record in the database.
func (p *Persist) CreateFile(file *File) error {
	return p.db.Create(file).Error
}

// GetFileByID retrieves a file by its ID.
func (p *Persist) GetFileByID(id int) (*File, error) {
	var file File
	err := p.db.First(&file, id).Error
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
func (p *Persist) DeleteFile(id uint) error {
	return p.db.Delete(&File{}, id).Error
}
