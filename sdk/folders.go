package sdk

import (
	"fmt"
	"net/http"
	"net/url"
)

// CreateFolder creates a new folder. parent is the destination folder's
// UUID, or "" for the root folder.
func (c *Client) CreateFolder(h http.Header, name, parent string) error {
	req := CreateFolderRequest{Name: name, Parent: parent}
	return c.request(h, http.MethodPost, "/v1/folder", req, nil)
}

// DeleteFolder moves a folder, and everything nested inside it, to the trash.
func (c *Client) DeleteFolder(h http.Header, folderID string) (MessageResponse, error) {
	var out MessageResponse
	err := c.request(h, http.MethodDelete, fmt.Sprintf("/v1/folder/%s", folderID), nil, &out)
	return out, err
}

// RestoreFolder restores a trashed folder and everything nested inside it.
func (c *Client) RestoreFolder(h http.Header, folderID string) error {
	return c.request(h, http.MethodPatch, fmt.Sprintf("/v1/folder/%s/restore", folderID), nil, nil)
}

// PurgeFolder permanently deletes a trashed folder and everything nested
// inside it.
func (c *Client) PurgeFolder(h http.Header, folderID string) error {
	return c.request(h, http.MethodDelete, fmt.Sprintf("/v1/folder/%s/purge", folderID), nil, nil)
}

// UpdateFolderName renames a folder.
func (c *Client) UpdateFolderName(h http.Header, folderID, newName string) error {
	return c.request(h, http.MethodPatch, fmt.Sprintf("/v1/folder/%s/%s", folderID, url.PathEscape(newName)), nil, nil)
}

// ListFolderContents lists the files and subfolders of a folder, along with
// its breadcrumb trail, as one unified page of Items. Pass "" for folderID to
// list the root folder. opts.SortBy accepts SortByName, SortBySize and
// SortByDate; opts.Type limits the list to folders or files.
func (c *Client) ListFolderContents(h http.Header, folderID string, opts ListOptions) (V1FolderContentsResponse, error) {
	var out V1FolderContentsResponse
	path := fmt.Sprintf("/v1/folder/list/%s?%s", folderID, opts.query())
	err := c.request(h, http.MethodGet, path, nil, &out)
	return out, err
}
