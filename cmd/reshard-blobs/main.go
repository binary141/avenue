// Command reshard-blobs is a one-off maintenance tool that migrates file
// blobs from the old flat "/{createdBy}/{uuid}" storage layout to the new
// sharded "/{uuid[0:2]}/{uuid[2:4]}/{uuid}" layout (see shared.BlobPath),
// and backfills the checksum column for any file that doesn't have one yet.
//
// It's idempotent and safe to re-run: a file whose blob already lives at its
// new sharded path is left alone (aside from backfilling its checksum if
// missing), and a file with a checksum already set is skipped entirely.
//
// Stop the server before running this against a live UPLOAD_DIR — it moves
// blobs on disk that the running app assumes are still at their old path.
//
// Usage:
//
//	go run ./cmd/reshard-blobs [-dry-run]
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"path"

	"avenue/backend/db"
	"avenue/backend/shared"

	"github.com/spf13/afero"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "log what would change without touching disk or the database")
	flag.Parse()

	if err := db.Connect(); err != nil {
		log.Fatalf("db connect: %v", err)
	}

	fs := afero.NewBasePathFs(afero.NewOsFs(), shared.GetEnv("UPLOAD_DIR", "./avenuectl/temp/"))

	refs, err := db.ListAllFileBlobRefs()
	if err != nil {
		log.Fatalf("list files: %v", err)
	}

	var migrated, checksummed, skipped, failed int
	for _, ref := range refs {
		oldPath := fmt.Sprintf("/%d/%s", ref.CreatedBy, ref.UUID)
		newPath := shared.BlobPath(ref.UUID)

		newExists, err := afero.Exists(fs, newPath)
		if err != nil {
			log.Printf("file %s: check new path: %v", ref.UUID, err)
			failed++
			continue
		}

		if !newExists {
			oldExists, err := afero.Exists(fs, oldPath)
			if err != nil {
				log.Printf("file %s: check old path: %v", ref.UUID, err)
				failed++
				continue
			}
			if !oldExists {
				log.Printf("file %s: blob missing at both %s and %s, skipping", ref.UUID, oldPath, newPath)
				skipped++
				continue
			}

			checksum, err := moveBlob(fs, oldPath, newPath, *dryRun)
			if err != nil {
				log.Printf("file %s: move blob: %v", ref.UUID, err)
				failed++
				continue
			}
			migrated++

			if ref.Checksum == "" {
				if !*dryRun {
					if err := db.SetFileChecksum(ref.UUID, checksum); err != nil {
						log.Printf("file %s: set checksum: %v", ref.UUID, err)
						failed++
						continue
					}
				}
				checksummed++
			}
			continue
		}

		// Blob is already at its sharded path (e.g. a previous partial run).
		// Just backfill the checksum if it's missing.
		if ref.Checksum == "" {
			checksum, err := hashBlob(fs, newPath)
			if err != nil {
				log.Printf("file %s: hash existing blob: %v", ref.UUID, err)
				failed++
				continue
			}
			if !*dryRun {
				if err := db.SetFileChecksum(ref.UUID, checksum); err != nil {
					log.Printf("file %s: set checksum: %v", ref.UUID, err)
					failed++
					continue
				}
			}
			checksummed++
			continue
		}

		skipped++
	}

	prefix := ""
	if *dryRun {
		prefix = "[dry run] "
	}
	log.Printf("%sdone: %d blobs moved, %d checksums backfilled, %d already up to date, %d failed",
		prefix, migrated, checksummed, skipped, failed)
}

// moveBlob copies oldPath to newPath while hashing its contents, removes
// oldPath on success, and returns the hex-encoded SHA-256 checksum.
func moveBlob(fs afero.Fs, oldPath, newPath string, dryRun bool) (string, error) {
	src, err := fs.Open(oldPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = src.Close() }()

	hasher := sha256.New()

	if dryRun {
		if _, err := io.Copy(hasher, src); err != nil {
			return "", err
		}
		return hex.EncodeToString(hasher.Sum(nil)), nil
	}

	if err := fs.MkdirAll(path.Dir(newPath), 0o755); err != nil {
		return "", err
	}

	dst, err := fs.Create(newPath)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(io.MultiWriter(dst, hasher), src); err != nil {
		_ = dst.Close()
		_ = fs.Remove(newPath)
		return "", err
	}
	if err := dst.Close(); err != nil {
		return "", err
	}

	if err := fs.Remove(oldPath); err != nil && !errors.Is(err, afero.ErrFileNotFound) {
		return "", fmt.Errorf("copied to %s but failed to remove old blob %s: %w", newPath, oldPath, err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// hashBlob computes the SHA-256 checksum of an existing blob without moving it.
func hashBlob(fs afero.Fs, path string) (string, error) {
	f, err := fs.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
