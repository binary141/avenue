package sdk

import (
	"fmt"
	"net/http"
)

// ListSessions lists the authenticated user's active sessions, with the
// session used to make the request marked via IsCurrent.
func (c *Client) ListSessions(h http.Header) ([]SessionInfo, error) {
	var out []SessionInfo
	err := c.request(h, http.MethodGet, "/v1/user/sessions", nil, &out)
	return out, err
}

// RevokeSession revokes a single session by ID. The caller's current session
// cannot be revoked this way; use Logout instead.
func (c *Client) RevokeSession(h http.Header, sessionID int64) error {
	return c.request(h, http.MethodDelete, fmt.Sprintf("/v1/user/sessions/%d", sessionID), nil, nil)
}

// RevokeOtherSessions revokes every session for the authenticated user except
// the one making the request.
func (c *Client) RevokeOtherSessions(h http.Header) error {
	return c.request(h, http.MethodDelete, "/v1/user/sessions", nil, nil)
}
