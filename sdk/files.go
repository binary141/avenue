package sdk

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

// UploadFile uploads a file's contents. parent is the destination folder's
// UUID, or "" for the root folder.
func (c *Client) UploadFile(h http.Header, filename string, data io.Reader, parent string) error {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	if parent != "" {
		if err := mw.WriteField("parent", parent); err != nil {
			return err
		}
	}

	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(fw, data); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}

	resp, err := c.rawRequest(h, http.MethodPost, "/v1/file", &buf, mw.FormDataContentType())
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

// ListFiles lists every file owned by the authenticated user.
func (c *Client) ListFiles(h http.Header) ([]File, error) {
	var out []File
	err := c.request(h, http.MethodGet, "/v1/file/list", nil, &out)
	return out, err
}

// SearchFiles searches for files by name within a folder. Pass "" for
// folderID to search the root folder.
func (c *Client) SearchFiles(h http.Header, folderID, fileName string) ([]File, error) {
	var out []File
	path := fmt.Sprintf("/v1/folder/files/%s", fileName)
	if folderID != "" {
		path = fmt.Sprintf("/v1/folder/%s/files/%s", folderID, fileName)
	}
	err := c.request(h, http.MethodGet, path, nil, &out)
	return out, err
}

// DownloadFile streams a file's contents. The caller must close the
// returned response body. The file name is available on
// resp.Header.Get("Content-Disposition").
func (c *Client) DownloadFile(h http.Header, fileID string) (*http.Response, error) {
	return c.rawRequest(h, http.MethodGet, fmt.Sprintf("/v1/file/%s", fileID), nil, "")
}

// MoveFile moves a file to a new parent folder. parent is the destination
// folder's UUID, or "" to move the file to the root.
func (c *Client) MoveFile(h http.Header, fileID, parent string) error {
	req := MoveFileRequest{Parent: parent}
	return c.request(h, http.MethodPatch, fmt.Sprintf("/v1/file/%s/move", fileID), req, nil)
}

// UpdateFileName renames a file.
func (c *Client) UpdateFileName(h http.Header, fileID, newName string) error {
	return c.request(h, http.MethodPatch, fmt.Sprintf("/v1/file/%s/%s", fileID, newName), nil, nil)
}

// DeleteFile moves a file to the trash.
func (c *Client) DeleteFile(h http.Header, fileID string) error {
	return c.request(h, http.MethodDelete, fmt.Sprintf("/v1/file/%s", fileID), nil, nil)
}

// RestoreFile restores a trashed file.
func (c *Client) RestoreFile(h http.Header, fileID string) error {
	return c.request(h, http.MethodPatch, fmt.Sprintf("/v1/file/%s/restore", fileID), nil, nil)
}

// PurgeFile permanently deletes a trashed file.
func (c *Client) PurgeFile(h http.Header, fileID string) error {
	return c.request(h, http.MethodDelete, fmt.Sprintf("/v1/file/%s/purge", fileID), nil, nil)
}

// ListTrash lists the files and folders the user has trashed. page and
// limit control pagination of both lists; pass 0 for either to use the
// server-side defaults.
func (c *Client) ListTrash(h http.Header, page, limit int) (V1TrashResponse, error) {
	var out V1TrashResponse
	path := "/v1/trash?" + paginationQuery(page, limit)
	err := c.request(h, http.MethodGet, path, nil, &out)
	return out, err
}

// BulkDelete moves a batch of files and folders to the trash in a single
// request.
func (c *Client) BulkDelete(h http.Header, req BulkDeleteRequest) (MessageResponse, error) {
	var out MessageResponse
	err := c.request(h, http.MethodDelete, "/v1/files/bulk-delete", req, &out)
	return out, err
}

// BulkRestore restores a batch of trashed files and folders in a single
// request.
func (c *Client) BulkRestore(h http.Header, req BulkRestoreRequest) (V1RestoreResponse, error) {
	var out V1RestoreResponse
	err := c.request(h, http.MethodPatch, "/v1/files/bulk-restore", req, &out)
	return out, err
}

// BulkMove moves a batch of files and folders to a new parent folder in a
// single request.
func (c *Client) BulkMove(h http.Header, req BulkMoveRequest) (MessageResponse, error) {
	var out MessageResponse
	err := c.request(h, http.MethodPatch, "/v1/files/bulk-move", req, &out)
	return out, err
}

// EmptyTrash permanently deletes everything the user has trashed.
func (c *Client) EmptyTrash(h http.Header) error {
	return c.request(h, http.MethodPost, "/v1/trash/empty", nil, nil)
}

// DownloadFilesZip streams a zip archive of the requested files, or of a
// single folder's subtree, as a raw response. The caller must close the
// returned response body. The archive name is available on
// resp.Header.Get("Content-Disposition").
func (c *Client) DownloadFilesZip(h http.Header, req DownloadFilesZipRequest) (*http.Response, error) {
	q := url.Values{}
	for _, id := range req.FileIDs {
		q.Add("ids", id)
	}
	for _, id := range req.FolderIDs {
		q.Add("folderIds", id)
	}
	return c.rawRequest(h, http.MethodGet, "/v1/files/zip?"+q.Encode(), nil, "")
}
