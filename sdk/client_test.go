package sdk

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// recordingServer answers every request with 200 + "null" (which decodes into
// either a struct or a slice) and hands the request to check so each test can
// assert on what the SDK actually sent.
func recordingServer(t *testing.T, check func(r *http.Request, body []byte)) (*Client, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		check(r, body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("null"))
	}))
	return NewClient(srv.URL), srv.Close
}

func TestUpdatePasswordSendsCurrentPassword(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody UpdatePasswordRequest

	c, done := recordingServer(t, func(r *http.Request, body []byte) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
	})
	defer done()

	if _, err := c.UpdatePassword(http.Header{}, "old-password", "new-password"); err != nil {
		t.Fatalf("UpdatePassword: %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != "/v1/user/password" {
		t.Errorf("request = %s %s, want PATCH /v1/user/password", gotMethod, gotPath)
	}
	if gotBody.CurrentPassword != "old-password" {
		t.Errorf("currentPassword = %q, want %q", gotBody.CurrentPassword, "old-password")
	}
	if gotBody.Password != "new-password" {
		t.Errorf("password = %q, want %q", gotBody.Password, "new-password")
	}
}

func TestNameInPathIsEscaped(t *testing.T) {
	const name = "my file?#100%.txt"
	const wantEscaped = "my%20file%3F%23100%25.txt"

	cases := []struct {
		label string
		call  func(c *Client) error
		want  string
	}{
		{
			label: "UpdateFileName",
			call: func(c *Client) error {
				return c.UpdateFileName(http.Header{}, "file-uuid", name)
			},
			want: "/v1/file/file-uuid/" + wantEscaped,
		},
		{
			label: "UpdateFolderName",
			call: func(c *Client) error {
				return c.UpdateFolderName(http.Header{}, "folder-uuid", name)
			},
			want: "/v1/folder/folder-uuid/" + wantEscaped,
		},
		{
			label: "SearchFiles in folder",
			call: func(c *Client) error {
				_, err := c.SearchFiles(http.Header{}, "folder-uuid", name)
				return err
			},
			want: "/v1/folder/folder-uuid/files/" + wantEscaped,
		},
		{
			label: "SearchFiles in root",
			call: func(c *Client) error {
				_, err := c.SearchFiles(http.Header{}, "", name)
				return err
			},
			want: "/v1/folder/files/" + wantEscaped,
		},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			var gotURI string
			c, done := recordingServer(t, func(r *http.Request, _ []byte) {
				gotURI = r.RequestURI
			})
			defer done()

			if err := tc.call(c); err != nil {
				t.Fatalf("%s: %v", tc.label, err)
			}
			if gotURI != tc.want {
				t.Errorf("request URI = %q, want %q", gotURI, tc.want)
			}
		})
	}
}
