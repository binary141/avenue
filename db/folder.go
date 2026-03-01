package db

import (
	"database/sql"

	"github.com/google/uuid"
)

const rootFolderID = "c32af1cc-aba9-4878-a305-5006dc7a5b76"

type Folder struct {
	FolderID string `json:"folder_id"`
	Name     string `json:"name"`
	Parent   string `json:"parent"`
	OwnerID  string `json:"owner_id"`
}

func CreateFolder(f *Folder) (string, error) {
	if f.FolderID == "" {
		f.FolderID = uuid.NewString()
	}
	_, err := DB.Exec(
		`INSERT INTO folders (folder_id, name, parent, owner_id) VALUES ($1, $2, $3, $4)`,
		f.FolderID, f.Name, f.Parent, f.OwnerID,
	)
	return f.FolderID, err
}

func GetFolder(folderID, userID string) (*Folder, error) {
	var f Folder
	err := DB.QueryRow(
		`SELECT folder_id, name, parent, owner_id FROM folders WHERE folder_id=$1 AND owner_id=$2`,
		folderID, userID,
	).Scan(&f.FolderID, &f.Name, &f.Parent, &f.OwnerID)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func UpdateFolder(f Folder) error {
	_, err := DB.Exec(
		`UPDATE folders SET name=$2 WHERE folder_id=$1 AND owner_id=$3`,
		f.FolderID, f.Name, f.OwnerID,
	)
	return err
}

func DeleteFolder(folderID, userID string) error {
	_, err := DB.Exec(
		`DELETE FROM folders WHERE folder_id=$1 AND owner_id=$2`,
		folderID, userID,
	)
	return err
}

func ListChildFolder(parentID, ownerID string) ([]Folder, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if parentID != rootFolderID {
		rows, err = DB.Query(
			`SELECT folder_id, name, parent, owner_id FROM folders WHERE parent=$1 AND owner_id=$2`,
			parentID, ownerID,
		)
	} else {
		rows, err = DB.Query(
			`SELECT folder_id, name, parent, owner_id FROM folders WHERE parent='' AND owner_id=$1`,
			ownerID,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []Folder
	for rows.Next() {
		var f Folder
		if err := rows.Scan(&f.FolderID, &f.Name, &f.Parent, &f.OwnerID); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

func ListFolderParents(folderID, ownerID string) ([]Folder, error) {
	rows, err := DB.Query(`
		WITH RECURSIVE folder_breadcrumbs (folder_id, name, parent, owner_id) AS (
			SELECT folder_id, name, parent, owner_id
			FROM folders
			WHERE owner_id = $1 AND folder_id = $2

			UNION ALL

			SELECT p.folder_id, p.name, p.parent, p.owner_id
			FROM folders p
			INNER JOIN folder_breadcrumbs c ON p.folder_id = c.parent
		)
		SELECT folder_id, name, parent, owner_id FROM folder_breadcrumbs
	`, ownerID, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []Folder
	for rows.Next() {
		var f Folder
		if err := rows.Scan(&f.FolderID, &f.Name, &f.Parent, &f.OwnerID); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}
