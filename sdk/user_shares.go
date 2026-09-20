package sdk

import (
	"fmt"
	"net/http"
)

// CreateFileUserShare shares a file with another user, found by email. Only
// the file's owner may do this. Sharing again with the same user changes
// their role.
func (c *Client) CreateFileUserShare(h http.Header, fileID string, req CreateUserShareRequest) (UserShare, error) {
	var out UserShare
	err := c.request(h, http.MethodPost, fmt.Sprintf("/v1/file/%s/user-shares", fileID), req, &out)
	return out, err
}

// CreateFolderUserShare shares a folder, and everything inside it, with
// another user. Only the folder's owner may do this.
func (c *Client) CreateFolderUserShare(h http.Header, folderID string, req CreateUserShareRequest) (UserShare, error) {
	var out UserShare
	err := c.request(h, http.MethodPost, fmt.Sprintf("/v1/folder/%s/user-shares", folderID), req, &out)
	return out, err
}

// ListFileUserShares lists the users a file is shared with. Owner only.
func (c *Client) ListFileUserShares(h http.Header, fileID string) ([]UserShare, error) {
	var out []UserShare
	err := c.request(h, http.MethodGet, fmt.Sprintf("/v1/file/%s/user-shares", fileID), nil, &out)
	return out, err
}

// ListFolderUserShares lists the users a folder is shared with. Owner only.
func (c *Client) ListFolderUserShares(h http.Header, folderID string) ([]UserShare, error) {
	var out []UserShare
	err := c.request(h, http.MethodGet, fmt.Sprintf("/v1/folder/%s/user-shares", folderID), nil, &out)
	return out, err
}

// UpdateUserShare changes the role of a user share. Only the shared item's
// owner may do this.
func (c *Client) UpdateUserShare(h http.Header, shareID int64, req UpdateUserShareRequest) error {
	return c.request(h, http.MethodPatch, fmt.Sprintf("/v1/user-shares/%d", shareID), req, nil)
}

// DeleteUserShare removes a user share: the owner revokes it, or the grantee
// leaves the share.
func (c *Client) DeleteUserShare(h http.Header, shareID int64) error {
	return c.request(h, http.MethodDelete, fmt.Sprintf("/v1/user-shares/%d", shareID), nil, nil)
}

// ListSharedWithMe lists the files and folders other users have shared
// directly with the caller.
func (c *Client) ListSharedWithMe(h http.Header) ([]SharedWithMeItem, error) {
	var out []SharedWithMeItem
	err := c.request(h, http.MethodGet, "/v1/shared-with-me", nil, &out)
	return out, err
}
