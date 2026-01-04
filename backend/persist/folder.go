package persist

import (
	"avenue/backend/shared"

	"github.com/google/uuid"
)

type Folder struct {
	FolderID string `gorm:"primaryKey, type:uuid, column:folder_id" json:"folder_id"`
	Name     string `gorm:"not null" json:"name"`
	Parent   string `json:"parent"`
	OwnerID  string `gorm:"not null, column:owner_id;type:bigint" json:"owner_id"`
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

func (p *Persist) ListChildFolder(parentID string, ownerID string) ([]Folder, error) {
	var f []Folder
	db := p.db
	if parentID != shared.ROOTFOLDERID {
		db = db.Where("parent = ?", parentID)
	} else {
		db = db.Where("parent = ''")
	}
	err := db.Where("owner_id = ?", ownerID).Find(&f).Error
	return f, err
}

func (p *Persist) ListFolderParents(folderID, ownerID string) ([]Folder, error) {
	sqlQuery := `
with recursive folder_breadcrumbs (folder_id, name, parent, owner_id) as (
    select 
        folder_id,
        name, 
        parent,
        owner_id
    from folders
    where 
        owner_id = ?
        and folder_id = ?

    union all

    select 
        parent.folder_id,
        parent.name, 
        parent.parent,
        parent.owner_id
    from folders parent
    inner join folder_breadcrumbs child
        on parent.folder_id = child.parent
    )
    select * from folder_breadcrumbs;
	`

	var folders []Folder

	result := p.db.Raw(sqlQuery, ownerID, folderID).Scan(&folders)

	if result.Error != nil {
		return nil, result.Error
	}

	return folders, nil
}
