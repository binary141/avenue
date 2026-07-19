package sdk

import (
	"fmt"
	"net/http"
)

// GetProfile returns the authenticated user's own profile.
func (c *Client) GetProfile(h http.Header) (User, error) {
	var out User
	err := c.request(h, http.MethodGet, "/v1/user/profile", nil, &out)
	return out, err
}

// GetUsers lists all users. Requires an admin caller.
func (c *Client) GetUsers(h http.Header) ([]User, error) {
	var out []User
	err := c.request(h, http.MethodGet, "/v1/users", nil, &out)
	return out, err
}

// CreateUser creates a new user account. Requires an admin caller. Returns
// the full, refreshed user list.
func (c *Client) CreateUser(h http.Header, req CreateUserRequest) ([]User, error) {
	var out []User
	err := c.request(h, http.MethodPost, "/v1/user", req, &out)
	return out, err
}

// UpdateProfile updates a user's profile. req.ID selects the target user;
// non-admin callers may only update themselves.
func (c *Client) UpdateProfile(h http.Header, req UpdateProfileRequest) (User, error) {
	var out User
	err := c.request(h, http.MethodPut, "/v1/user/profile", req, &out)
	return out, err
}

// UpdateProfileByID is the PATCH /v1/user/:userID variant of UpdateProfile.
func (c *Client) UpdateProfileByID(h http.Header, userID string, req UpdateProfileRequest) (User, error) {
	var out User
	err := c.request(h, http.MethodPatch, fmt.Sprintf("/v1/user/%s", userID), req, &out)
	return out, err
}

// UpdatePassword changes the authenticated user's own password.
func (c *Client) UpdatePassword(h http.Header, newPassword string) (User, error) {
	req := UpdatePasswordRequest{Password: newPassword}
	var out User
	err := c.request(h, http.MethodPatch, "/v1/user/password", req, &out)
	return out, err
}

// AdminSendPasswordReset emails a password-reset link to the target user.
// Requires an admin caller.
func (c *Client) AdminSendPasswordReset(h http.Header, userID string) error {
	return c.request(h, http.MethodPost, fmt.Sprintf("/v1/user/%s/send-reset-email", userID), nil, nil)
}
