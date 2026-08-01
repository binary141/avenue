package db

import (
	"strconv"
	"time"
)

// LocalUser is avenue's local mirror of a keyring user: just enough to be
// the FK anchor for files/folders/shares (via its auto-inc ID, unchanged
// from before) and to hold the quota/usage tracking that stays internal to
// this service. Identity (email, name, password, admin role) lives in
// keyring and is never duplicated here.
type LocalUser struct {
	ID            int64     `json:"id"`
	KeyringUserID int64     `json:"keyringUserId"`
	KeyringUUID   string    `json:"keyringUuid"`
	Quota         int64     `json:"quota"`
	SpaceUsed     int64     `json:"spaceUsed"`
	CreatedAt     time.Time `json:"createdAt"`
}

// GetOrCreateLocalUser returns the local row mirroring the given keyring
// user, creating it (with zero quota) on first sight. Called on every
// authenticated request, so it stays a single round trip.
func GetOrCreateLocalUser(keyringUserID int64, keyringUUID string) (LocalUser, error) {
	var u LocalUser
	err := DB.QueryRow(`
		INSERT INTO users (keyring_user_id, keyring_uuid, created_at, updated_at)
		VALUES ($1, $2, now(), now())
		ON CONFLICT (keyring_user_id) WHERE deleted_at IS NULL DO UPDATE SET keyring_user_id = EXCLUDED.keyring_user_id
		RETURNING id, keyring_user_id, keyring_uuid, quota, space_used, created_at
	`, keyringUserID, keyringUUID).Scan(&u.ID, &u.KeyringUserID, &u.KeyringUUID, &u.Quota, &u.SpaceUsed, &u.CreatedAt)
	return u, err
}

// GetLocalUserByIDStr is GetLocalUserByID taking the string form of the ID
// found in the request context (shared.GetUserIDFromContext).
func GetLocalUserByIDStr(idStr string) (LocalUser, error) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return LocalUser{}, err
	}
	return GetLocalUserByID(id)
}

func GetLocalUserByID(id int64) (LocalUser, error) {
	var u LocalUser
	err := DB.QueryRow(`
		SELECT id, keyring_user_id, keyring_uuid, quota, space_used, created_at
		FROM users WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&u.ID, &u.KeyringUserID, &u.KeyringUUID, &u.Quota, &u.SpaceUsed, &u.CreatedAt)
	return u, err
}

// ListLocalUsersByKeyringID returns every local user row, keyed by keyring
// user ID, so a caller holding a keyring-sourced user list can attach
// quota/space_used to each entry in one query.
func ListLocalUsersByKeyringID() (map[int64]LocalUser, error) {
	rows, err := DB.Query(`
		SELECT id, keyring_user_id, keyring_uuid, quota, space_used, created_at
		FROM users WHERE deleted_at IS NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64]LocalUser)
	for rows.Next() {
		var u LocalUser
		if err := rows.Scan(&u.ID, &u.KeyringUserID, &u.KeyringUUID, &u.Quota, &u.SpaceUsed, &u.CreatedAt); err != nil {
			return nil, err
		}
		out[u.KeyringUserID] = u
	}
	return out, rows.Err()
}

// UpdateQuota sets the byte quota for the local user identified by its
// avenue-local ID. A quota of 0 means unlimited (mirrors the previous
// behavior in handlers/dashboard.go).
func UpdateQuota(localID int64, quota int64) error {
	_, err := DB.Exec(`UPDATE users SET quota=$2, updated_at=now() WHERE id=$1`, localID, quota)
	return err
}

func GetUserUsage(localID int64) (int64, error) {
	var spaceUsed int64
	err := DB.QueryRow(`SELECT space_used FROM users WHERE id=$1 AND deleted_at IS NULL`, localID).Scan(&spaceUsed)
	return spaceUsed, err
}

func UpdateUsage(localID int64, delta int64) error {
	_, err := DB.Exec(`
		UPDATE users SET space_used = GREATEST(0, space_used + $2), updated_at=now() WHERE id=$1
	`, localID, delta)
	return err
}
