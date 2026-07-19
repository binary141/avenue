package sdk

import (
	"fmt"
	"net/http"
)

// CreateShareLink creates a public share link for a file. Requires file
// sharing to be enabled server-side.
func (c *Client) CreateShareLink(h http.Header, fileID string, req CreateShareLinkRequest) (V1ShareLinkResponse, error) {
	var out V1ShareLinkResponse
	err := c.request(h, http.MethodPost, fmt.Sprintf("/v1/file/%s/share", fileID), req, &out)
	return out, err
}

// GetShareLinkMeta fetches metadata about a shared file (name, size, mime
// type) without downloading it. Public endpoint.
func (c *Client) GetShareLinkMeta(h http.Header, token string) (V1ShareLinkMetaResponse, error) {
	var out V1ShareLinkMetaResponse
	err := c.request(h, http.MethodGet, fmt.Sprintf("/api/share/%s", token), nil, &out)
	return out, err
}

// ListFileShares lists active share links for a single file.
func (c *Client) ListFileShares(h http.Header, fileID string) ([]ShareLink, error) {
	var out []ShareLink
	err := c.request(h, http.MethodGet, fmt.Sprintf("/v1/file/%s/shares", fileID), nil, &out)
	return out, err
}

// ListUserShares lists all active file share links created by the
// authenticated user.
func (c *Client) ListUserShares(h http.Header) ([]ShareLinkWithFileName, error) {
	var out []ShareLinkWithFileName
	err := c.request(h, http.MethodGet, "/v1/shares", nil, &out)
	return out, err
}

// ListExpiredUserShares lists expired file share links created by the
// authenticated user.
func (c *Client) ListExpiredUserShares(h http.Header) ([]ShareLinkWithFileName, error) {
	var out []ShareLinkWithFileName
	err := c.request(h, http.MethodGet, "/v1/shares/expired", nil, &out)
	return out, err
}

// RevokeShareLink revokes a file share link.
func (c *Client) RevokeShareLink(h http.Header, token string) error {
	return c.request(h, http.MethodDelete, fmt.Sprintf("/v1/share/%s", token), nil, nil)
}

// DownloadSharedFile streams a shared file's contents via its public share
// token. The caller must close the returned response body.
func (c *Client) DownloadSharedFile(h http.Header, token string) (*http.Response, error) {
	return c.rawRequest(h, http.MethodGet, fmt.Sprintf("/api/share/%s/download", token), nil, "")
}
