package sdk

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// DefaultUploadChunkSize is the chunk size UploadFileChunked splits a file
// into. It matches the web client's chunk size, chosen to balance
// per-request overhead against how much a single failed chunk has to re-send.
const DefaultUploadChunkSize = 16 * 1024 * 1024

// InitUpload starts a new chunked upload session for a file of the given
// size. Send its bytes with UploadChunk, then call CompleteUpload once every
// byte has landed. Prefer UploadFileChunked for the common case of sending a
// whole file in one call.
func (c *Client) InitUpload(h http.Header, req InitUploadRequest) (UploadSessionStatus, error) {
	var out UploadSessionStatus
	err := c.request(h, http.MethodPost, "/v1/upload", req, &out)
	return out, err
}

// GetUploadStatus reports how many bytes of an upload session the server has
// received so far, so a client that lost its own progress (e.g. resuming
// after a crash) can carry on from the right offset instead of restarting.
func (c *Client) GetUploadStatus(h http.Header, uploadID string) (UploadSessionStatus, error) {
	var out UploadSessionStatus
	err := c.request(h, http.MethodGet, fmt.Sprintf("/v1/upload/%s", uploadID), nil, &out)
	return out, err
}

// UploadChunk sends one chunk of an upload session: the bytes [start, end)
// out of a file totalSize bytes long. It must start exactly where the
// server's bytes_received left off; if it doesn't (e.g. a retried chunk
// already landed), the server responds 409 with its real offset instead of
// an error, so the returned status is always populated — check
// out.BytesReceived == end to confirm the chunk was accepted, and resend
// from out.BytesReceived otherwise.
//
// data must support Seek because the chunk is hashed up front (its digest is
// sent as a request header, so it has to be known before the body starts
// streaming) and then re-read from the start to actually send.
func (c *Client) UploadChunk(h http.Header, uploadID string, start, end, totalSize int64, data io.ReadSeeker) (UploadSessionStatus, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, data); err != nil {
		return UploadSessionStatus{}, err
	}
	if _, err := data.Seek(0, io.SeekStart); err != nil {
		return UploadSessionStatus{}, err
	}
	checksum := hex.EncodeToString(hasher.Sum(nil))

	req, err := http.NewRequest(http.MethodPut, c.BaseURL+fmt.Sprintf("/v1/upload/%s/chunk", uploadID), data)
	if err != nil {
		return UploadSessionStatus{}, err
	}
	for k, v := range h {
		req.Header[k] = v
	}
	req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end-1, totalSize))
	req.Header.Set("X-Chunk-Checksum", "sha256:"+checksum)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return UploadSessionStatus{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		_ = json.NewDecoder(resp.Body).Decode(apiErr)
		return UploadSessionStatus{}, apiErr
	}

	var out UploadSessionStatus
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return UploadSessionStatus{}, err
	}
	return out, nil
}

// CompleteUpload finalizes an upload session once every byte has been sent,
// returning the created file record.
func (c *Client) CompleteUpload(h http.Header, uploadID string) (File, error) {
	var out File
	err := c.request(h, http.MethodPost, fmt.Sprintf("/v1/upload/%s/complete", uploadID), nil, &out)
	return out, err
}

// CancelUpload aborts an in-progress upload session, discarding whatever
// bytes it has received so far.
func (c *Client) CancelUpload(h http.Header, uploadID string) error {
	return c.request(h, http.MethodDelete, fmt.Sprintf("/v1/upload/%s", uploadID), nil, nil)
}

// UploadFileChunked uploads a file of known size via the chunked-upload
// protocol (init, then DefaultUploadChunkSize chunks, then complete), rather
// than the single-request UploadFile. Prefer this for large files: a dropped
// connection only loses the current chunk, and a caller that keeps the
// returned upload ID around can resume a failed attempt with GetUploadStatus
// and UploadChunk instead of starting over.
func (c *Client) UploadFileChunked(h http.Header, name string, data io.ReaderAt, size int64, parent string) (File, error) {
	session, err := c.InitUpload(h, InitUploadRequest{Name: name, Size: size, Parent: parent})
	if err != nil {
		return File{}, err
	}

	for session.BytesReceived < size {
		start := session.BytesReceived
		end := start + DefaultUploadChunkSize
		if end > size {
			end = size
		}

		chunk := io.NewSectionReader(data, start, end-start)
		session, err = c.UploadChunk(h, session.UploadID, start, end, size, chunk)
		if err != nil {
			return File{}, err
		}
		if session.BytesReceived == start {
			return File{}, fmt.Errorf("avenue: chunk at offset %d was not accepted", start)
		}
	}

	return c.CompleteUpload(h, session.UploadID)
}
