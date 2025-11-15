package persist

import "github.com/google/uuid"

type Folder struct {
	FolderID string `gorm:"primaryKey, type:uuid, column:folder_id" json:"folder_id"`
	Name     string `gorm:"not null" json:"name"`
	Parent   string `json:"parent"`
	OwnerId  int    `gorm:"not null, column:owner_id" json:"owner_id"`
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
	db := p.db
	if parentId != "-1" {
		db = db.Where("parent = ?", parentId)
	} else {
		db = db.Where("parent = ''")
	}
	err := db.Find(&f).Error
	return f, err
}
