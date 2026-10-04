package shared

import (
	"os"

	"github.com/spf13/afero"
)

// BlobPath returns the on-disk path for a file blob, sharded two levels
// deep by the first four characters of its UUID (e.g. "ab/cd/abcd1234-...").
// Flat directories holding hundreds of thousands of files degrade badly on
// most local filesystems, so blobs are spread across a fixed set of shard
// directories instead of one directory per user.
func BlobPath(uuid string) string {
	return "/" + uuid[0:2] + "/" + uuid[2:4] + "/" + uuid
}

// CheckBlobRootWritable verifies that the blob root can be written to by
// creating and removing a probe file at its top level. Run at startup so a
// misconfigured or read-only UPLOAD_DIR fails immediately instead of on the
// first upload.
func CheckBlobRootWritable(fs afero.Fs) error {
	const probe = ".avenue-write-check"
	f, err := fs.OpenFile(probe, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return fs.Remove(probe)
}

// EnsureBlobDir creates the shard directory that BlobPath(uuid) lives in, if
// it doesn't already exist.
func EnsureBlobDir(fs afero.Fs, uuid string) error {
	dir := "/" + uuid[0:2] + "/" + uuid[2:4]
	exists, err := afero.DirExists(fs, dir)
	if err != nil {
		return err
	}
	if !exists {
		return fs.MkdirAll(dir, os.ModePerm)
	}
	return nil
}
