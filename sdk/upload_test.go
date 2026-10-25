package sdk

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// formPart is one multipart part as the server reads it, in arrival order.
type formPart struct {
	Name     string
	Filename string
	Body     string
}

func readParts(t *testing.T, r *http.Request) []formPart {
	t.Helper()
	mr, err := r.MultipartReader()
	if err != nil {
		t.Fatalf("multipart reader: %v", err)
	}
	var parts []formPart
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("next part: %v", err)
		}
		b, _ := io.ReadAll(p)
		parts = append(parts, formPart{Name: p.FormName(), Filename: p.FileName(), Body: string(b)})
	}
	return parts
}

// The handler (handlers.CompleteUpload / UploadFile) reads parent before file,
// so the SDK must write parent first and never write it when it's empty.
func TestUploadFileMultipartShape(t *testing.T) {
	cases := []struct {
		label     string
		parent    string
		wantParts []formPart
	}{
		{
			label:  "with parent folder",
			parent: "folder-uuid",
			wantParts: []formPart{
				{Name: "parent", Body: "folder-uuid"},
				{Name: "file", Filename: "notes.txt", Body: "hello"},
			},
		},
		{
			label:  "root folder omits parent",
			parent: "",
			wantParts: []formPart{
				{Name: "file", Filename: "notes.txt", Body: "hello"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			var gotMethod, gotPath string
			var gotParts []formPart
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				gotParts = readParts(t, r)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"uuid":"new-uuid","name":"notes.txt","file_size":5,"checksum":"abc"}`))
			}))
			defer srv.Close()

			c := NewClient(srv.URL)
			got, err := c.UploadFile(http.Header{}, "notes.txt", strings.NewReader("hello"), tc.parent)
			if err != nil {
				t.Fatalf("UploadFile: %v", err)
			}

			if gotMethod != http.MethodPost || gotPath != "/v1/file" {
				t.Errorf("request = %s %s, want POST /v1/file", gotMethod, gotPath)
			}
			if len(gotParts) != len(tc.wantParts) {
				t.Fatalf("parts = %+v, want %+v", gotParts, tc.wantParts)
			}
			for i, want := range tc.wantParts {
				if gotParts[i] != want {
					t.Errorf("part %d = %+v, want %+v", i, gotParts[i], want)
				}
			}
			if got.UUID != "new-uuid" || got.FileSize != 5 || got.Checksum != "abc" {
				t.Errorf("returned File = %+v, want the record the server created", got)
			}
		})
	}
}

func TestUploadToSharedFolderShape(t *testing.T) {
	cases := []struct {
		label   string
		target  string
		wantURI string
	}{
		{label: "shared folder root", target: "", wantURI: "/api/share/folder/tok123/upload"},
		{label: "subfolder target", target: "sub uuid", wantURI: "/api/share/folder/tok123/upload?folder=sub+uuid"},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			var gotURI string
			var gotParts []formPart
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotURI = r.RequestURI
				gotParts = readParts(t, r)
				w.WriteHeader(http.StatusCreated)
			}))
			defer srv.Close()

			c := NewClient(srv.URL)
			if err := c.UploadToSharedFolder(http.Header{}, "tok123", "a.bin", strings.NewReader("xyz"), tc.target); err != nil {
				t.Fatalf("UploadToSharedFolder: %v", err)
			}

			if gotURI != tc.wantURI {
				t.Errorf("request URI = %q, want %q", gotURI, tc.wantURI)
			}
			want := []formPart{{Name: "file", Filename: "a.bin", Body: "xyz"}}
			if len(gotParts) != 1 || gotParts[0] != want[0] {
				t.Errorf("parts = %+v, want %+v", gotParts, want)
			}
		})
	}
}

// The chunk contract is: raw bytes as the body, Content-Range "bytes
// start-(end-1)/total", and X-Chunk-Checksum "sha256:<hex of the chunk>".
func TestUploadChunkHeaders(t *testing.T) {
	const chunk = "0123456789"
	sum := sha256.Sum256([]byte(chunk))
	wantChecksum := "sha256:" + hex.EncodeToString(sum[:])

	var gotMethod, gotPath, gotRange, gotChecksum string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotRange = r.Header.Get("Content-Range")
		gotChecksum = r.Header.Get("X-Chunk-Checksum")
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"uploadId":"up-1","bytesReceived":10}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	status, err := c.UploadChunk(http.Header{}, "up-1", 0, 10, 25, strings.NewReader(chunk))
	if err != nil {
		t.Fatalf("UploadChunk: %v", err)
	}

	if gotMethod != http.MethodPut || gotPath != "/v1/upload/up-1/chunk" {
		t.Errorf("request = %s %s, want PUT /v1/upload/up-1/chunk", gotMethod, gotPath)
	}
	if gotRange != "bytes 0-9/25" {
		t.Errorf("Content-Range = %q, want %q", gotRange, "bytes 0-9/25")
	}
	if gotChecksum != wantChecksum {
		t.Errorf("X-Chunk-Checksum = %q, want %q", gotChecksum, wantChecksum)
	}
	if string(gotBody) != chunk {
		t.Errorf("body = %q, want %q", gotBody, chunk)
	}
	if status.UploadID != "up-1" || status.BytesReceived != 10 {
		t.Errorf("status = %+v, want uploadId up-1 with 10 bytes", status)
	}
}

// A 409 means the chunk's offset didn't match what the server has. The SDK
// reports the server's real offset instead of an error, so callers can resend.
func TestUploadChunkConflictReturnsServerOffset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"uploadId":"up-1","bytesReceived":20}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	status, err := c.UploadChunk(http.Header{}, "up-1", 0, 10, 25, strings.NewReader("0123456789"))
	if err != nil {
		t.Fatalf("UploadChunk on 409 returned error: %v", err)
	}
	if status.BytesReceived != 20 {
		t.Errorf("BytesReceived = %d, want the server's 20", status.BytesReceived)
	}
}

// UploadFileChunked must follow init -> chunks -> complete, with the init body
// matching handlers' initUploadRequest (name, size, parent), and return the
// File that complete responds with.
func TestUploadFileChunkedProtocol(t *testing.T) {
	const content = "hello chunked world"
	var steps []string
	var initBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		steps = append(steps, key)
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/upload":
			if err := json.NewDecoder(r.Body).Decode(&initBody); err != nil {
				t.Errorf("decode init body: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"uploadId":"up-9","bytesReceived":0}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v1/upload/up-9/chunk":
			_, _ = io.ReadAll(r.Body)
			_, _ = w.Write([]byte(`{"uploadId":"up-9","bytesReceived":19}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/upload/up-9/complete":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"uuid":"file-9","name":"big.txt","file_size":19}`))
		default:
			t.Errorf("unexpected request %s", key)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	file, err := c.UploadFileChunked(http.Header{}, "big.txt", strings.NewReader(content), int64(len(content)), "")
	if err != nil {
		t.Fatalf("UploadFileChunked: %v", err)
	}

	wantSteps := []string{
		"POST /v1/upload",
		"PUT /v1/upload/up-9/chunk",
		"POST /v1/upload/up-9/complete",
	}
	if strings.Join(steps, "|") != strings.Join(wantSteps, "|") {
		t.Errorf("steps = %v, want %v", steps, wantSteps)
	}
	if initBody["name"] != "big.txt" || initBody["size"] != float64(len(content)) {
		t.Errorf("init body = %v, want name and size", initBody)
	}
	if _, ok := initBody["parent"]; ok {
		t.Errorf("init body has parent %v, want it omitted for the root", initBody["parent"])
	}
	if file.UUID != "file-9" || file.FileSize != 19 {
		t.Errorf("returned File = %+v, want the completed record", file)
	}
}
