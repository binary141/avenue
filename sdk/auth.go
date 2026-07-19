package sdk

import "net/http"

// Ping checks that the server is reachable.
func (c *Client) Ping(h http.Header) (MessageResponse, error) {
	var out MessageResponse
	err := c.request(h, http.MethodGet, "/v1/ping", nil, &out)
	return out, err
}

// LoginMeta reports whether self-registration is enabled.
func (c *Client) LoginMeta(h http.Header) (V1LoginMetaResponse, error) {
	var out V1LoginMetaResponse
	err := c.request(h, http.MethodGet, "/loginMeta", nil, &out)
	return out, err
}

// Login authenticates with an email/password and returns the new session.
func (c *Client) Login(h http.Header, req LoginRequest) (V1LoginResponse, error) {
	var out V1LoginResponse
	err := c.request(h, http.MethodPost, "/login", req, &out)
	return out, err
}

// Logout ends the caller's current session.
func (c *Client) Logout(h http.Header) (MessageResponse, error) {
	var out MessageResponse
	err := c.request(h, http.MethodPost, "/v1/logout", nil, &out)
	return out, err
}

// Register self-registers a new user account (only when registration is
// enabled on the server).
func (c *Client) Register(h http.Header, req RegisterRequest) (User, error) {
	var out User
	err := c.request(h, http.MethodPost, "/register", req, &out)
	return out, err
}

// ForgotPassword emails a password-reset link, if the email matches an
// account. Always succeeds on the wire to avoid leaking account existence.
func (c *Client) ForgotPassword(h http.Header, email string) error {
	req := ForgotPasswordRequest{Email: email}
	return c.request(h, http.MethodPost, "/forgot-password", req, nil)
}

// ResetPassword consumes a password-reset token and sets a new password.
func (c *Client) ResetPassword(h http.Header, token, newPassword string) error {
	req := ResetPasswordRequest{Token: token, NewPassword: newPassword}
	return c.request(h, http.MethodPost, "/reset-password", req, nil)
}
