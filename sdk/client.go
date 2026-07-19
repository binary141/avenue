// Package sdk is a lightweight Go client for the Avenue HTTP API. Every call
// takes the caller-supplied headers (for authentication) and returns the
// decoded response type alongside an error, mirroring net/http idioms.
package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client is an Avenue API client. The zero value is not usable; construct
// one with NewClient.
type Client struct {
	// BaseURL is the scheme+host (and optional path prefix) of the Avenue
	// server, e.g. "http://localhost:8080". No trailing slash.
	BaseURL string
	// HTTPClient is used to perform requests. Defaults to http.DefaultClient
	// when nil.
	HTTPClient *http.Client
}

// NewClient returns a Client targeting baseURL.
func NewClient(baseURL string) *Client {
	return &Client{BaseURL: baseURL}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// APIError is returned when the server responds with a non-2xx status. It
// carries the decoded error body (see handlers.Response) plus the status
// code.
type APIError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message"`
	Error_     string `json:"error"`
}

func (e *APIError) Error() string {
	if e.Error_ != "" {
		return fmt.Sprintf("avenue: %d: %s", e.StatusCode, e.Error_)
	}
	if e.Message != "" {
		return fmt.Sprintf("avenue: %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("avenue: %d", e.StatusCode)
}

// request performs an HTTP request against the Avenue API. body is JSON
// marshaled when non-nil; out is JSON unmarshaled from the response body
// when non-nil. Callers get back an *APIError for non-2xx responses.
func (c *Client) request(h http.Header, method, path string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return err
	}
	for k, v := range h {
		req.Header[k] = v
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		_ = json.NewDecoder(resp.Body).Decode(apiErr)
		return apiErr
	}

	if out == nil {
		return nil
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// rawRequest performs an HTTP request and returns the raw *http.Response for
// endpoints that don't return JSON (file downloads). The caller is
// responsible for closing the response body. Non-2xx responses are
// translated into an *APIError and the body is closed for you.
func (c *Client) rawRequest(h http.Header, method, path string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequest(method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	for k, v := range h {
		req.Header[k] = v
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		defer func() { _ = resp.Body.Close() }()
		apiErr := &APIError{StatusCode: resp.StatusCode}
		_ = json.NewDecoder(resp.Body).Decode(apiErr)
		return nil, apiErr
	}

	return resp, nil
}
