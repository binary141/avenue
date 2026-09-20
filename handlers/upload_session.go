package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"avenue/backend/db"
	"avenue/backend/logger"
	"avenue/backend/sdk"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
	"github.com/spf13/afero"
)

type initUploadRequest struct {
	Name   string `json:"name" binding:"required"`
	Size   int64  `json:"size" binding:"required"`
	Parent string `json:"parent"`
}

type initUploadResponse struct {
	UploadID       string          `json:"uploadId"`
	BytesReceived  int64           `json:"bytesReceived"`
	ReceivedRanges []db.ChunkRange `json:"receivedRanges,omitempty"`
}

// parseContentRange parses a "bytes start-end/total" Content-Range header
// value, as sent by the chunked upload client for each chunk it PUTs.
func parseContentRange(header string) (start, end, total int64, err error) {
	const prefix = "bytes "
	if !strings.HasPrefix(header, prefix) {
		return 0, 0, 0, fmt.Errorf("missing %q prefix", prefix)
	}
	n, err := fmt.Sscanf(header[len(prefix):], "%d-%d/%d", &start, &end, &total)
	if err != nil || n != 3 {
		return 0, 0, 0, fmt.Errorf("malformed Content-Range %q", header)
	}
	if start < 0 || end < start || total <= 0 {
		return 0, 0, 0, fmt.Errorf("invalid range values in %q", header)
	}
	return start, end, total, nil
}

// InitUpload starts a new chunked upload. It performs the same quota/size
// checks the single-shot upload does, up front, since the client is about to
// commit to sending the whole file.
func (s *Server) InitUpload(c *gin.Context) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not get user id", err)
		return
	}
	userIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return
	}

	var req initUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond(c, http.StatusBadRequest, "invalid request", err)
		return
	}
	if req.Size <= 0 {
		respond(c, http.StatusBadRequest, "size must be greater than zero", nil)
		return
	}

	// Uploading into a folder shared with the caller stores the file in the
	// folder owner's drive, so it's the owner's quota that applies. This also
	// checks the caller may upload to req.Parent at all.
	ownerID, ok := resolveUploadOwner(c, userID, req.Parent)
	if !ok {
		return
	}

	owner, err := db.GetUserByID(ownerID)
	if err != nil {
		logger.Errorf("error getting user: %s", err.Error())
		respond(c, http.StatusInternalServerError, "", err)
		return
	}

	envMaxFileSize := shared.GetEnvInt64("MAX_FILE_BYTE_SIZE", shared.DEFAULTMAXFILESIZE)
	var totalUsed int64
	if owner.Quota != 0 {
		totalUsed, err = db.GetUserUsage(ownerID)
		if err != nil {
			logger.Errorf("error getting user quota: %s", err.Error())
			respond(c, http.StatusInternalServerError, "", err)
			return
		}
	}

	maxFileSize, overQuota := computeUploadLimit(owner.Quota, totalUsed, envMaxFileSize)
	if overQuota {
		msg := "max quota reached. Please delete files to be able to upload files"
		if ownerID != userIDInt {
			msg = "the storage quota of the drive this folder belongs to is full"
		}
		respond(c, http.StatusUnprocessableEntity, "", errors.New(msg))
		return
	}
	if req.Size > maxFileSize {
		respond(c, http.StatusRequestEntityTooLarge, "", errors.New("file too large"))
		return
	}

	filename := filepath.Base(req.Name)
	extension := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))

	session, err := db.CreateUploadSession(userIDInt, filename, extension, req.Parent, req.Size)
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not create upload session", err)
		return
	}

	if err := shared.EnsureBlobDir(s.fs, session.ID); err != nil {
		respond(c, http.StatusInternalServerError, "could not ensure dir exists", err)
		return
	}
	// Pre-allocate the destination so chunks (which may arrive with gaps
	// resent out of order after a retry) can be written with WriteAt.
	dst, err := s.fs.Create(shared.BlobPath(session.ID))
	if err != nil {
		_ = db.DeleteUploadSession(session.ID, userIDInt)
		respond(c, http.StatusInternalServerError, "could not create file", err)
		return
	}
	if err := dst.Close(); err != nil {
		logger.Errorf("close new upload blob: %v", err)
	}

	c.JSON(http.StatusCreated, initUploadResponse{UploadID: session.ID, BytesReceived: 0})
}

// UploadStatus reports how many bytes of an upload session the server has
// already received, so a client that lost its in-memory state (e.g. a page
// reload) can resume from the right offset instead of restarting.
func (s *Server) UploadStatus(c *gin.Context) {
	userIDInt, session, ok := s.loadUploadSession(c)
	if !ok {
		return
	}
	ranges, err := db.ListUploadSessionChunks(session.ID, userIDInt)
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not load upload progress", err)
		return
	}
	c.JSON(http.StatusOK, initUploadResponse{UploadID: session.ID, BytesReceived: session.BytesReceived, ReceivedRanges: ranges})
}

// loadUploadSession fetches the upload session named by the :uploadID param,
// scoped to the requesting user, writing a 404 and returning ok=false if it
// doesn't exist (already completed, expired and swept, or belongs to someone
// else).
func (s *Server) loadUploadSession(c *gin.Context) (int64, *db.UploadSession, bool) {
	userID, err := shared.GetUserIDFromContext(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not get user id", err)
		return 0, nil, false
	}
	userIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		respond(c, http.StatusInternalServerError, "", err)
		return 0, nil, false
	}

	session, err := db.GetUploadSession(c.Param("uploadID"), userIDInt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond(c, http.StatusNotFound, "upload session not found", nil)
			return 0, nil, false
		}
		respond(c, http.StatusInternalServerError, "could not load upload session", err)
		return 0, nil, false
	}
	return userIDInt, session, true
}

// chunkChecksumHeader carries a per-chunk integrity check, formatted as
// "sha256:<hex digest>" of the chunk's exact bytes. It lets UploadChunk catch
// a corrupted chunk immediately (so the client can cheaply resend just that
// chunk) instead of only finding out via CompleteUpload's whole-file hash
// after every chunk has already landed.
const chunkChecksumHeader = "X-Chunk-Checksum"

// parseChunkChecksum extracts the hex sha256 digest from a chunkChecksumHeader
// value.
func parseChunkChecksum(header string) (string, error) {
	const prefix = "sha256:"
	if !strings.HasPrefix(header, prefix) {
		return "", fmt.Errorf("missing or malformed %s header", chunkChecksumHeader)
	}
	sum := strings.TrimPrefix(header, prefix)
	if len(sum) != hex.EncodedLen(sha256.Size) {
		return "", fmt.Errorf("invalid checksum length in %s header", chunkChecksumHeader)
	}
	if _, err := hex.DecodeString(sum); err != nil {
		return "", fmt.Errorf("invalid checksum encoding in %s header", chunkChecksumHeader)
	}
	return sum, nil
}

// UploadChunk appends one chunk to an in-progress upload. The chunk's byte
// range is given by the Content-Range header (bytes start-end/total, as sent
// by the client). Chunks may arrive out of order and several may be in
// flight at once (the client sends a handful of chunks per file in
// parallel) — the destination file is written with WriteAt at the chunk's
// own offset, and db.RecordUploadChunk idempotently tracks which byte
// ranges have landed so a retried chunk can't be double-counted.
func (s *Server) UploadChunk(c *gin.Context) {
	userIDInt, session, ok := s.loadUploadSession(c)
	if !ok {
		return
	}

	start, end, total, err := parseContentRange(c.GetHeader("Content-Range"))
	if err != nil {
		respond(c, http.StatusBadRequest, "invalid Content-Range header", err)
		return
	}
	if total != session.TotalSize {
		respond(c, http.StatusBadRequest, "Content-Range total does not match upload size", nil)
		return
	}
	chunkLen := end - start + 1
	if end >= total || chunkLen <= 0 {
		respond(c, http.StatusBadRequest, "chunk out of range", nil)
		return
	}

	wantChecksum, err := parseChunkChecksum(c.GetHeader(chunkChecksumHeader))
	if err != nil {
		respond(c, http.StatusBadRequest, "invalid chunk checksum header", err)
		return
	}

	body := http.MaxBytesReader(c.Writer, c.Request.Body, chunkLen)

	dst, err := s.fs.OpenFile(shared.BlobPath(session.ID), os.O_WRONLY, 0o644)
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not open upload destination", err)
		return
	}
	hasher := sha256.New()
	if written, err := io.Copy(io.NewOffsetWriter(dst, start), io.TeeReader(body, hasher)); err != nil || written != chunkLen {
		_ = dst.Close()
		if err == nil {
			err = io.ErrUnexpectedEOF
		}
		respond(c, http.StatusBadRequest, "could not read chunk body", err)
		return
	}
	if err := dst.Close(); err != nil {
		logger.Errorf("close upload chunk dest: %v", err)
	}

	if gotChecksum := hex.EncodeToString(hasher.Sum(nil)); gotChecksum != wantChecksum {
		respond(c, http.StatusBadRequest, "chunk checksum mismatch", nil)
		return
	}

	newBytesReceived, err := db.RecordUploadChunk(session.ID, userIDInt, start, end+1)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond(c, http.StatusNotFound, "upload session not found", nil)
			return
		}
		if errors.Is(err, db.ErrOverlappingChunk) {
			respond(c, http.StatusBadRequest, "chunk overlaps a different previously received range", nil)
			return
		}
		respond(c, http.StatusInternalServerError, "could not record chunk progress", err)
		return
	}

	c.JSON(http.StatusOK, initUploadResponse{UploadID: session.ID, BytesReceived: newBytesReceived})
}

// CompleteUpload finalizes an upload session once every byte has arrived:
// it hashes the assembled blob, detects its content type, creates the file
// record (reusing the session's id as the file's UUID, since the blob is
// already sitting at that path), and drops the now-finished session row.
func (s *Server) CompleteUpload(c *gin.Context) {
	userIDInt, session, ok := s.loadUploadSession(c)
	if !ok {
		return
	}
	userID := strconv.FormatInt(userIDInt, 10)

	if session.BytesReceived != session.TotalSize {
		respond(c, http.StatusConflict, "upload is not complete", nil)
		return
	}

	// Re-check access rather than trusting what InitUpload saw: the share may
	// have been revoked while the upload was in flight.
	ownerID, ok := resolveUploadOwner(c, userID, session.Parent)
	if !ok {
		return
	}

	blobPath := shared.BlobPath(session.ID)
	src, err := s.fs.Open(blobPath)
	if err != nil {
		respond(c, http.StatusInternalServerError, "could not open uploaded blob", err)
		return
	}
	defer src.Close()

	// Force the assembled blob to disk before we treat it as durable — chunk
	// writes may still be sitting in the OS page cache, and a crash before
	// they're flushed would silently lose bytes in a file we've already told
	// the client is complete.
	if err := src.Sync(); err != nil {
		respond(c, http.StatusInternalServerError, "could not sync uploaded blob", err)
		return
	}

	sniff := make([]byte, 512)
	n, err := io.ReadAtLeast(src, sniff, 1)
	if err != nil && err != io.ErrUnexpectedEOF {
		respond(c, http.StatusInternalServerError, "could not read uploaded blob", err)
		return
	}
	contentType := http.DetectContentType(sniff[:n])

	hasher := sha256.New()
	hasher.Write(sniff[:n])
	if _, err := io.Copy(hasher, src); err != nil {
		respond(c, http.StatusInternalServerError, "could not checksum uploaded blob", err)
		return
	}
	checksum := hex.EncodeToString(hasher.Sum(nil))

	fileRecord := &sdk.File{
		UUID:      session.ID,
		Name:      session.Name,
		Extension: session.Extension,
		MimeType:  contentType,
		FileSize:  session.TotalSize,
		Parent:    session.Parent,
		CreatedBy: ownerID,
	}
	if _, err := db.CreateFile(fileRecord); err != nil {
		respond(c, http.StatusInternalServerError, "could not create file record", err)
		return
	}
	// CreateFile may have deduplicated the name against existing siblings;
	// it also doesn't take a checksum, so set the final name + checksum here.
	if err := db.UpdateFile(*fileRecord, userID); err != nil {
		respond(c, http.StatusInternalServerError, "could not finalize file record", err)
		return
	}
	fileRecord.Checksum = checksum

	// CreateFile already applied session.TotalSize to the user's usage.

	if err := db.DeleteUploadSession(session.ID, userIDInt); err != nil {
		logger.Errorf("delete completed upload session %s: %v", session.ID, err)
	}

	c.JSON(http.StatusCreated, fileRecord)
}

// CancelUpload aborts an in-progress upload session: its blob and DB row are
// removed so it doesn't linger as orphaned state or count toward quota (it
// never did, since usage is only applied on CompleteUpload).
func (s *Server) CancelUpload(c *gin.Context) {
	userIDInt, session, ok := s.loadUploadSession(c)
	if !ok {
		return
	}

	if err := s.fs.Remove(shared.BlobPath(session.ID)); err != nil && !errors.Is(err, afero.ErrFileNotFound) {
		logger.Errorf("cancel upload: remove blob %s: %v", session.ID, err)
	}
	if err := db.DeleteUploadSession(session.ID, userIDInt); err != nil {
		respond(c, http.StatusInternalServerError, "could not cancel upload", err)
		return
	}

	c.Status(http.StatusNoContent)
}
