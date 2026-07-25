// Package sweeper periodically hard-deletes files and folders that have
// been sitting in the trash longer than a configured retention period.
package sweeper

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"avenue/backend/db"
	"avenue/backend/logger"
	"avenue/backend/sdk"
	"avenue/backend/shared"

	"github.com/spf13/afero"
)

const (
	retentionEnvKey      = "TRASH_RETENTION"
	defaultRetention     = "720h" // 30 days
	intervalEnvKey       = "TRASH_SWEEP_INTERVAL"
	defaultSweepInterval = "5m"
)

// Retention returns the configured trash retention period — how long an
// item sits in the trash before the sweeper permanently deletes it.
func Retention() time.Duration {
	return shared.GetEnvDuration(retentionEnvKey, defaultRetention)
}

// Start launches a background goroutine that sweeps the trash on a fixed
// interval, hard-deleting anything trashed for longer than the retention
// period. Both are configurable via env vars (Go duration strings, e.g.
// "5m", "720h"): TRASH_SWEEP_INTERVAL (default 5m) and TRASH_RETENTION
// (default 720h, i.e. 30 days).
func Start(fs afero.Fs) {
	interval := shared.GetEnvDuration(intervalEnvKey, defaultSweepInterval)
	retention := Retention()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		sweep(fs, retention)
		for range ticker.C {
			sweep(fs, retention)
		}
	}()
}

func sweep(fs afero.Fs, retention time.Duration) {
	cutoff := time.Now().Add(-retention)

	folders, err := db.ListExpiredTrashedFolders(cutoff)
	if err != nil {
		logger.Errorf("trash sweeper: list expired folders: %v", err)
	}
	for _, folder := range folders {
		files, err := db.PurgeFolder(folder.UUID, strconv.FormatInt(folder.OwnerID, 10))
		if err != nil {
			logger.Errorf("trash sweeper: purge folder %s: %v", folder.UUID, err)
			continue
		}
		for _, f := range files {
			removeBlob(fs, f)
			if err := db.UpdateUsage(f.CreatedBy, -f.FileSize); err != nil {
				logger.Errorf("trash sweeper: update usage for user %d: %v", f.CreatedBy, err)
			}
		}
	}

	files, err := db.ListExpiredTrashedFiles(cutoff)
	if err != nil {
		logger.Errorf("trash sweeper: list expired files: %v", err)
		return
	}
	for _, f := range files {
		purged, err := db.PurgeFileForUser(f.UUID, strconv.FormatInt(f.CreatedBy, 10))
		if err != nil {
			logger.Errorf("trash sweeper: purge file %s: %v", f.UUID, err)
			continue
		}
		removeBlob(fs, *purged)
		if err := db.UpdateUsage(purged.CreatedBy, -purged.FileSize); err != nil {
			logger.Errorf("trash sweeper: update usage for user %d: %v", purged.CreatedBy, err)
		}
	}
}

func removeBlob(fs afero.Fs, f sdk.File) {
	if err := fs.Remove(fmt.Sprintf("/%d/%s", f.CreatedBy, f.UUID)); err != nil && !errors.Is(err, afero.ErrFileNotFound) {
		logger.Errorf("trash sweeper: remove blob %s: %v", f.UUID, err)
	}
}
