package db

import (
	"database/sql"

	"avenue/backend/sdk"
)

// scanFolderItems reads the rows produced by ListFolderItems/ListTrashedItems
// — both select the same column shape (item_type, id, uuid, name, extension,
// mime_type, file_size, checksum, parent_id, owner_id, created_by,
// created_at, deleted_at) from their respective UNION ALL queries.
func scanFolderItems(rows *sql.Rows) ([]sdk.FolderItem, error) {
	defer rows.Close()

	var items []sdk.FolderItem
	for rows.Next() {
		var item sdk.FolderItem
		var checksum sql.NullString
		var deletedAt sql.NullTime
		if err := rows.Scan(
			&item.Type, &item.ID, &item.UUID, &item.Name, &item.Extension, &item.MimeType,
			&item.FileSize, &checksum, &item.ParentID, &item.OwnerID, &item.CreatedBy,
			&item.CreatedAt, &deletedAt,
		); err != nil {
			return nil, err
		}
		item.Checksum = checksum.String
		if deletedAt.Valid {
			item.DeletedAt = &deletedAt.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ListFolderItems lists the folders and/or files directly inside parentID
// (or the root, if parentID is "" or the root sentinel), owned by ownerID,
// as a single deterministically-ordered, paginated list — combining what
// were previously two separately-paginated queries (ListChildFolder,
// ListChildFile) so a page boundary can't split inconsistently between the
// two types. sortBy must already be validated by the caller to be one of
// "name", "file_size", "created_at", or the folders-first grouping
// expression used for the "type" sort; sortBy/sortDir are interpolated
// directly since placeholders can't parameterize ORDER BY, and sortDir is a
// typed sdk.SortDirection so callers can't pass arbitrary strings. A
// trailing "uuid ASC" tiebreak keeps LIMIT/OFFSET pagination stable when the
// sort column has duplicate values. itemType filters the result to just
// sdk.FolderItemTypeFolder or sdk.FolderItemTypeFile; an empty string
// returns both, unioned as before.
func ListFolderItems(parentID, ownerID, sortBy string, sortDir sdk.SortDirection, limit, offset int, itemType string) ([]sdk.FolderItem, error) {
	orderBy := " ORDER BY " + sortBy + " " + string(sortDir) + ", uuid ASC"
	includeFolders := itemType == "" || itemType == sdk.FolderItemTypeFolder
	includeFiles := itemType == "" || itemType == sdk.FolderItemTypeFile

	var (
		rows *sql.Rows
		err  error
	)

	if parentID == "" || parentID == rootFolderID {
		folderQuery := `
			SELECT 'folder' AS item_type, f.id, f.uuid, f.name, '' AS extension, '' AS mime_type,
			       0::BIGINT AS file_size, NULL::TEXT AS checksum, COALESCE(f.parent_id, 0) AS parent_id,
			       f.owner_id, 0::BIGINT AS created_by, f.created_at, f.deleted_at
			FROM folders f
			WHERE f.parent_id IS NULL AND f.owner_id = $1::BIGINT AND f.deleted_at IS NULL`
		fileQuery := `
			SELECT 'file' AS item_type, fi.id, fi.uuid, fi.name, fi.extension, fi.mime_type,
			       fi.file_size, fi.checksum, COALESCE(fi.parent_id, 0) AS parent_id,
			       0::BIGINT AS owner_id, fi.created_by, fi.created_at, fi.deleted_at
			FROM files fi
			WHERE fi.parent_id IS NULL AND fi.created_by = $1::BIGINT AND fi.deleted_at IS NULL`

		query := "SELECT * FROM (" + combineQueries(includeFolders, includeFiles, folderQuery, fileQuery) + `) combined` + orderBy + ` LIMIT $2 OFFSET $3`
		rows, err = DB.Query(query, ownerID, limit, offset)
	} else {
		folderQuery := `
			SELECT 'folder' AS item_type, f.id, f.uuid, f.name, '' AS extension, '' AS mime_type,
			       0::BIGINT AS file_size, NULL::TEXT AS checksum, COALESCE(f.parent_id, 0) AS parent_id,
			       f.owner_id, 0::BIGINT AS created_by, f.created_at, f.deleted_at
			FROM folders f
			WHERE f.parent_id = (SELECT id FROM folders WHERE uuid = $1 AND owner_id = $2::BIGINT)
			  AND f.owner_id = $2::BIGINT AND f.deleted_at IS NULL`
		fileQuery := `
			SELECT 'file' AS item_type, fi.id, fi.uuid, fi.name, fi.extension, fi.mime_type,
			       fi.file_size, fi.checksum, COALESCE(fi.parent_id, 0) AS parent_id,
			       0::BIGINT AS owner_id, fi.created_by, fi.created_at, fi.deleted_at
			FROM files fi
			WHERE fi.parent_id = (SELECT id FROM folders WHERE uuid = $1 AND owner_id = $2::BIGINT) AND fi.deleted_at IS NULL`

		query := "SELECT * FROM (" + combineQueries(includeFolders, includeFiles, folderQuery, fileQuery) + `) combined` + orderBy + ` LIMIT $3 OFFSET $4`
		rows, err = DB.Query(query, parentID, ownerID, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	return scanFolderItems(rows)
}

// combineQueries joins folderQuery and/or fileQuery with UNION ALL depending
// on which are included, so a type filter can drop a branch entirely instead
// of filtering the combined result.
func combineQueries(includeFolders, includeFiles bool, folderQuery, fileQuery string) string {
	switch {
	case includeFolders && includeFiles:
		return folderQuery + " UNION ALL " + fileQuery
	case includeFolders:
		return folderQuery
	default:
		return fileQuery
	}
}

// ListTrashedItems returns the top-level trashed folders and files for
// userID — i.e. items the user explicitly trashed, not items that are only
// in the trash because an ancestor folder was trashed — as a single
// deterministically-ordered, paginated list. sortBy must already be
// validated by the caller to be one of "name", "file_size", or
// "deleted_at".
func ListTrashedItems(userID, sortBy string, sortDir sdk.SortDirection, limit, offset int) ([]sdk.FolderItem, error) {
	orderBy := " ORDER BY " + sortBy + " " + string(sortDir) + ", uuid ASC LIMIT $2 OFFSET $3"

	rows, err := DB.Query(`
		SELECT * FROM (
			SELECT 'folder' AS item_type, f.id, f.uuid, f.name, '' AS extension, '' AS mime_type,
			       0::BIGINT AS file_size, NULL::TEXT AS checksum, COALESCE(f.parent_id, 0) AS parent_id,
			       f.owner_id, 0::BIGINT AS created_by, f.created_at, f.deleted_at
			FROM folders f
			LEFT JOIN folders p ON p.id = f.parent_id
			WHERE f.owner_id = $1::BIGINT AND f.deleted_at IS NOT NULL
			  AND (p.id IS NULL OR p.deleted_at IS NULL)

			UNION ALL

			SELECT 'file' AS item_type, fi.id, fi.uuid, fi.name, fi.extension, fi.mime_type,
			       fi.file_size, fi.checksum, COALESCE(fi.parent_id, 0) AS parent_id,
			       0::BIGINT AS owner_id, fi.created_by, fi.created_at, fi.deleted_at
			FROM files fi
			LEFT JOIN folders p ON p.id = fi.parent_id
			WHERE fi.created_by = $1::BIGINT AND fi.deleted_at IS NOT NULL
			  AND (p.id IS NULL OR p.deleted_at IS NULL)
		) combined
	`+orderBy, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return scanFolderItems(rows)
}
