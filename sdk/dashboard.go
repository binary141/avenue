package sdk

import "net/http"

// DashboardInfo returns server-wide/caller-specific dashboard metadata
// (upload limits, which sharing features are enabled).
func (c *Client) DashboardInfo(h http.Header) (V1DashboardResponse, error) {
	var out V1DashboardResponse
	err := c.request(h, http.MethodGet, "/v1/dashboard", nil, &out)
	return out, err
}
