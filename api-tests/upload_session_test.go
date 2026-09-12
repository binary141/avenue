package apitests

import (
	"bytes"
	"testing"

	"avenue/backend/sdk"
)

// testData returns size bytes of deterministic, non-zero content, filled a
// byte at a time so it works for any size (including ones that don't divide
// evenly into a repeated pattern).
func testData(size int64) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte('a' + i%26)
	}
	return data
}

func TestChunkedUploadLifecycle(t *testing.T) {
	client, h := authedClient(t)
	name := uniqueName("chunked") + ".bin"

	// Three chunks' worth of data plus a partial last chunk, to exercise
	// more than one round trip through UploadChunk.
	size := int64(sdk.DefaultUploadChunkSize*2 + 1024)
	data := testData(size)

	file, err := client.UploadFileChunked(h, name, bytes.NewReader(data), size, "")
	if err != nil {
		t.Fatalf("UploadFileChunked: %v", err)
	}
	if file.FileSize != size {
		t.Errorf("expected uploaded file size %d, got %d", size, file.FileSize)
	}

	found := findItem(listRoot(t, client, h), "file", name)
	if found == nil {
		t.Fatalf("uploaded file %q not found in root listing", name)
	}

	t.Cleanup(func() {
		_ = client.DeleteFile(h, file.UUID)
		_ = client.PurgeFile(h, file.UUID)
	})
}

func TestChunkedUploadResync(t *testing.T) {
	client, h := authedClient(t)
	name := uniqueName("chunked-resync") + ".bin"
	size := int64(sdk.DefaultUploadChunkSize + 1024)
	data := testData(size)

	session, err := client.InitUpload(h, sdk.InitUploadRequest{Name: name, Size: size})
	if err != nil {
		t.Fatalf("InitUpload: %v", err)
	}
	t.Cleanup(func() { _ = client.CancelUpload(h, session.UploadID) })

	firstEnd := int64(sdk.DefaultUploadChunkSize)
	session, err = client.UploadChunk(h, session.UploadID, 0, firstEnd, size, bytes.NewReader(data[:firstEnd]))
	if err != nil {
		t.Fatalf("UploadChunk: %v", err)
	}
	if session.BytesReceived != firstEnd {
		t.Fatalf("expected bytesReceived=%d after first chunk, got %d", firstEnd, session.BytesReceived)
	}

	// A client that lost track of its own progress (e.g. after a reload)
	// should be able to ask the server where it left off.
	status, err := client.GetUploadStatus(h, session.UploadID)
	if err != nil {
		t.Fatalf("GetUploadStatus: %v", err)
	}
	if status.BytesReceived != firstEnd {
		t.Fatalf("expected GetUploadStatus to report %d, got %d", firstEnd, status.BytesReceived)
	}

	// Resending the same range (a duplicate retry) shouldn't advance the
	// session past where it already was, and shouldn't error either — the
	// server reports its real offset instead so the client can resync.
	dup, err := client.UploadChunk(h, session.UploadID, 0, firstEnd, size, bytes.NewReader(data[:firstEnd]))
	if err != nil {
		t.Fatalf("UploadChunk (duplicate): %v", err)
	}
	if dup.BytesReceived != firstEnd {
		t.Fatalf("expected duplicate chunk to report bytesReceived=%d, got %d", firstEnd, dup.BytesReceived)
	}

	session, err = client.UploadChunk(h, session.UploadID, firstEnd, size, size, bytes.NewReader(data[firstEnd:]))
	if err != nil {
		t.Fatalf("UploadChunk (final): %v", err)
	}
	if session.BytesReceived != size {
		t.Fatalf("expected bytesReceived=%d after final chunk, got %d", size, session.BytesReceived)
	}

	file, err := client.CompleteUpload(h, session.UploadID)
	if err != nil {
		t.Fatalf("CompleteUpload: %v", err)
	}
	t.Cleanup(func() {
		_ = client.DeleteFile(h, file.UUID)
		_ = client.PurgeFile(h, file.UUID)
	})
}
