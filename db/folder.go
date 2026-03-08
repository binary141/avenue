package db

import (
	"database/sql"

	"github.com/google/uuid"
)

const rootFolderID = "c32af1cc-aba9-4878-a305-5006dc7a5b76"

type Folder struct {
	ID      int64  `json:"id"`
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Parent  string `json:"parent"`   // always "" — parent_id is stored as BIGINT, not returned
	OwnerID int64  `json:"owner_id"` // was TEXT, now BIGINT
}

func CreateFolder(f *Folder) (string, error) {
	if f.UUID == "" {
		f.UUID = uuid.NewString()
	}
	err := DB.QueryRow(`
		INSERT INTO folders (uuid, name, parent_id, owner_id)
		VALUES ($1, $2,
			CASE WHEN $3 = '' THEN NULL
			     ELSE (SELECT id FROM folders WHERE uuid = $3)
			END,
			$4)
		RETURNING id
	`, f.UUID, f.Name, f.Parent, f.OwnerID).Scan(&f.ID)
	return f.UUID, err
}

func GetFolder(folderID, userID string) (*Folder, error) {
	var f Folder
	err := DB.QueryRow(
		`SELECT id, uuid, name, owner_id FROM folders WHERE uuid=$1 AND owner_id=$2::BIGINT`,
		folderID, userID,
	).Scan(&f.ID, &f.UUID, &f.Name, &f.OwnerID)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func UpdateFolder(f Folder) error {
	_, err := DB.Exec(
		`UPDATE folders SET name=$2 WHERE uuid=$1 AND owner_id=$3`,
		f.UUID, f.Name, f.OwnerID,
	)
	return err
}

func DeleteFolder(folderID, userID string) error {
	_, err := DB.Exec(
		`DELETE FROM folders WHERE uuid=$1 AND owner_id=$2::BIGINT`,
		folderID, userID,
	)
	return err
}

func ListChildFolder(parentID, ownerID string) ([]Folder, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if parentID == "" || parentID == rootFolderID {
		rows, err = DB.Query(
			`SELECT id, uuid, name, owner_id FROM folders WHERE parent_id IS NULL AND owner_id=$1::BIGINT`,
			ownerID,
		)
	} else {
		rows, err = DB.Query(`
			SELECT id, uuid, name, owner_id FROM folders
			WHERE parent_id = (SELECT id FROM folders WHERE uuid = $1)
			  AND owner_id = $2::BIGINT`,
			parentID, ownerID,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []Folder
	for rows.Next() {
		var f Folder
		if err := rows.Scan(&f.ID, &f.UUID, &f.Name, &f.OwnerID); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

func ListFolderParents(folderID, ownerID string) ([]Folder, error) {
	rows, err := DB.Query(`
		WITH RECURSIVE folder_breadcrumbs (id, uuid, name, parent_id, owner_id) AS (
			SELECT id, uuid, name, parent_id, owner_id
			FROM folders
			WHERE owner_id = $1::BIGINT AND uuid = $2

			UNION ALL

			SELECT p.id, p.uuid, p.name, p.parent_id, p.owner_id
			FROM folders p
			INNER JOIN folder_breadcrumbs c ON p.id = c.parent_id
		)
		SELECT id, uuid, name, owner_id FROM folder_breadcrumbs
	`, ownerID, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []Folder
	for rows.Next() {
		var f Folder
		if err := rows.Scan(&f.ID, &f.UUID, &f.Name, &f.OwnerID); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}
