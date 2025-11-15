package persist

import "github.com/google/uuid"

type Folder struct {
	FolderID string `gorm:"primaryKey, type:uuid, column:folder_id"`
	Name     string `gorm:"not null"`
	Parent   string
	OwnerId  int `gorm:"not null, column:owner_id"`
}

func (p *Persist) CreateFolder(f *Folder) (string, error) {
	if f.FolderID == "" {
		f.FolderID = uuid.NewString()
	}
	return f.FolderID, p.db.Create(f).Error
}

func (p *Persist) GetFolder(id string) (*Folder, error) {
	var f Folder
	err := p.db.Where("folder_id = ?", id).First(&f).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (p *Persist) ListChildFolder(parentId string) ([]Folder, error) {
	var f []Folder
	err := p.db.Where("parent = ?").Find(f).Error
	return f, err
}
