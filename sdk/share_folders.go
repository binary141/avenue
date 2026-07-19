package sdk

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

// CreateFolderShareLink creates a public share link for a folder. Requires
// folder sharing to be enabled server-side.
func (c *Client) CreateFolderShareLink(h http.Header, folderID string, req CreateShareLinkRequest) (V1ShareLinkResponse, error) {
	var out V1ShareLinkResponse
	err := c.request(h, http.MethodPost, fmt.Sprintf("/v1/folder/%s/share", folderID), req, &out)
	return out, err
}

// ListFolderShares lists active share links for a single folder.
func (c *Client) ListFolderShares(h http.Header, folderID string) ([]ShareFolderLink, error) {
	var out []ShareFolderLink
	err := c.request(h, http.MethodGet, fmt.Sprintf("/v1/folder/%s/shares", folderID), nil, &out)
	return out, err
}

// ListUserFolderShares lists all active folder share links created by the
// authenticated user.
func (c *Client) ListUserFolderShares(h http.Header) ([]ShareFolderLink, error) {
	var out []ShareFolderLink
	err := c.request(h, http.MethodGet, "/v1/folder-shares", nil, &out)
	return out, err
}

// ListExpiredUserFolderShares lists expired folder share links created by
// the authenticated user.
func (c *Client) ListExpiredUserFolderShares(h http.Header) ([]ShareFolderLink, error) {
	var out []ShareFolderLink
	err := c.request(h, http.MethodGet, "/v1/folder-shares/expired", nil, &out)
	return out, err
}

// RevokeShareFolderLink revokes a folder share link.
func (c *Client) RevokeShareFolderLink(h http.Header, token string) error {
	return c.request(h, http.MethodDelete, fmt.Sprintf("/v1/share/folder/%s", token), nil, nil)
}

// GetSharedFolderContents lists the files and subfolders at the root of a
// shared folder via its public share token.
func (c *Client) GetSharedFolderContents(h http.Header, token string) (V1SharedFolderContentsResponse, error) {
	var out V1SharedFolderContentsResponse
	err := c.request(h, http.MethodGet, fmt.Sprintf("/api/share/folder/%s", token), nil, &out)
	return out, err
}

// BrowseSharedSubFolder lists the files and subfolders of a subfolder
// within a shared folder tree.
func (c *Client) BrowseSharedSubFolder(h http.Header, token, subFolderUUID string) (V1SharedFolderContentsResponse, error) {
	var out V1SharedFolderContentsResponse
	err := c.request(h, http.MethodGet, fmt.Sprintf("/api/share/folder/%s/browse/%s", token, subFolderUUID), nil, &out)
	return out, err
}

// DownloadSharedFolderFile streams a file within a shared folder tree. The
// caller must close the returned response body.
func (c *Client) DownloadSharedFolderFile(h http.Header, token, fileUUID string) (*http.Response, error) {
	return c.rawRequest(h, http.MethodGet, fmt.Sprintf("/api/share/folder/%s/file/%s", token, fileUUID), nil, "")
}

// UploadToSharedFolder uploads a file into a shared folder (or one of its
// subfolders, via targetFolderUUID) using a public share token. Requires
// the share link to have uploads enabled. Pass "" for targetFolderUUID to
// upload into the shared folder's root.
func (c *Client) UploadToSharedFolder(h http.Header, token, filename string, data io.Reader, targetFolderUUID string) error {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

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

	path := fmt.Sprintf("/api/share/folder/%s/upload", token)
	if targetFolderUUID != "" {
		path += "?folder=" + url.QueryEscape(targetFolderUUID)
	}

	resp, err := c.rawRequest(h, http.MethodPost, path, &buf, mw.FormDataContentType())
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}
