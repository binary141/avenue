package sdk

import (
	"fmt"
	"net/http"
)

// CreateFolder creates a new folder. parent is the destination folder's
// UUID, or "" for the root folder.
func (c *Client) CreateFolder(h http.Header, name, parent string) error {
	req := CreateFolderRequest{Name: name, Parent: parent}
	return c.request(h, http.MethodPost, "/v1/folder", req, nil)
}

// DeleteFolder deletes an empty folder. Fails if the folder still contains
// files or subfolders.
func (c *Client) DeleteFolder(h http.Header, folderID string) (MessageResponse, error) {
	var out MessageResponse
	err := c.request(h, http.MethodDelete, fmt.Sprintf("/v1/folder/%s", folderID), nil, &out)
	return out, err
}

// UpdateFolderName renames a folder.
func (c *Client) UpdateFolderName(h http.Header, folderID, newName string) error {
	return c.request(h, http.MethodPatch, fmt.Sprintf("/v1/folder/%s/%s", folderID, newName), nil, nil)
}

// ListFolderContents lists the files and subfolders of a folder, along with
// its breadcrumb trail. Pass "" for folderID to list the root folder.
func (c *Client) ListFolderContents(h http.Header, folderID string) (V1FolderContentsResponse, error) {
	var out V1FolderContentsResponse
	err := c.request(h, http.MethodGet, fmt.Sprintf("/v1/folder/list/%s", folderID), nil, &out)
	return out, err
}
