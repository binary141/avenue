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
