package sdk

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
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

// DeleteFile deletes a file.
func (c *Client) DeleteFile(h http.Header, fileID string) error {
	return c.request(h, http.MethodDelete, fmt.Sprintf("/v1/file/%s", fileID), nil, nil)
}
