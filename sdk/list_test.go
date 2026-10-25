package sdk

import (
	"encoding/json"
	"net/http"
	"testing"
)

// The handlers read these query parameters: page, limit, sort (only "desc"
// is special), sortBy (name, size, date for folders; name, size for the
// trash), and type (folders only). The SDK must send exactly those names.

func TestListFolderContentsQuery(t *testing.T) {
	cases := []struct {
		label    string
		folderID string
		opts     ListOptions
		wantPath string
	}{
		{
			label:    "all options",
			folderID: "folder-uuid",
			opts: ListOptions{
				Page:   2,
				Limit:  10,
				Sort:   SortDesc,
				SortBy: SortBySize,
				Type:   FolderItemTypeFile,
			},
			wantPath: "/v1/folder/list/folder-uuid?limit=10&page=2&sort=desc&sortBy=size&type=file",
		},
		{
			label:    "date ordering on folders",
			folderID: "folder-uuid",
			opts:     ListOptions{SortBy: SortByDate, Sort: SortAsc},
			wantPath: "/v1/folder/list/folder-uuid?sort=asc&sortBy=date",
		},
		{
			label:    "zero options send no query",
			folderID: "folder-uuid",
			wantPath: "/v1/folder/list/folder-uuid?",
		},
		{
			label:    "root folder",
			folderID: "",
			opts:     ListOptions{Page: 1, Limit: 200},
			wantPath: "/v1/folder/list/?limit=200&page=1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			var gotURI string
			c, done := recordingServer(t, func(r *http.Request, _ []byte) {
				gotURI = r.RequestURI
			})
			defer done()

			if _, err := c.ListFolderContents(http.Header{}, tc.folderID, tc.opts); err != nil {
				t.Fatalf("ListFolderContents: %v", err)
			}
			if gotURI != tc.wantPath {
				t.Errorf("request URI = %q, want %q", gotURI, tc.wantPath)
			}
		})
	}
}

func TestListTrashQuery(t *testing.T) {
	cases := []struct {
		label    string
		opts     ListOptions
		wantPath string
	}{
		{
			label:    "name ordering, ascending",
			opts:     ListOptions{Page: 1, Limit: 50, Sort: SortAsc, SortBy: SortByName},
			wantPath: "/v1/trash?limit=50&page=1&sort=asc&sortBy=name",
		},
		{
			label:    "zero options send no query",
			wantPath: "/v1/trash?",
		},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			var gotURI string
			c, done := recordingServer(t, func(r *http.Request, _ []byte) {
				gotURI = r.RequestURI
			})
			defer done()

			if _, err := c.ListTrash(http.Header{}, tc.opts); err != nil {
				t.Fatalf("ListTrash: %v", err)
			}
			if gotURI != tc.wantPath {
				t.Errorf("request URI = %q, want %q", gotURI, tc.wantPath)
			}
		})
	}
}

// TestFolderContentsDecodesHandlerShape decodes a body in the exact shape
// handlers.ListFolderContents writes (sdk.V1FolderContentsResponse, with
// FolderItem rows), to catch JSON tag drift between the two sides.
func TestFolderContentsDecodesHandlerShape(t *testing.T) {
	body := `{
		"items": [
			{"type":"folder","id":1,"uuid":"f-1","name":"docs","file_size":0,"parent_id":2,"owner_id":3,"created_at":"2026-01-02T03:04:05Z"},
			{"type":"file","id":4,"uuid":"u-4","name":"a.txt","extension":".txt","mimeType":"text/plain","file_size":12,"checksum":"abc","parent_id":2,"owner_id":3,"created_by":3,"created_at":"2026-01-02T03:04:05Z"}
		],
		"breadcrumbs": [{"label":"Drive","folder_id":""}],
		"page": 1,
		"limit": 50,
		"total": 2,
		"access": "owner"
	}`

	var got V1FolderContentsResponse
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(got.Items))
	}
	if got.Items[0].Type != FolderItemTypeFolder || got.Items[1].Type != FolderItemTypeFile {
		t.Errorf("item types = %q, %q", got.Items[0].Type, got.Items[1].Type)
	}
	if got.Items[1].Checksum != "abc" || got.Items[1].ParentID != 2 {
		t.Errorf("file row fields not decoded: %+v", got.Items[1])
	}
	if got.Total != 2 || got.Page != 1 || got.Limit != 50 || got.Access != "owner" {
		t.Errorf("paging/access not decoded: %+v", got)
	}
	if len(got.BreadCrumbs) != 1 || got.BreadCrumbs[0].Label != "Drive" {
		t.Errorf("breadcrumbs not decoded: %+v", got.BreadCrumbs)
	}
}
