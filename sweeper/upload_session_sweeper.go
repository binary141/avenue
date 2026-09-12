// Upload session sweeping: periodically deletes chunked-upload sessions
// that stopped receiving chunks (a client that closed the tab, crashed, or
// gave up on a resume) along with their partial blobs, so they don't sit on
// disk and in the DB forever.
package sweeper

import (
	"errors"
	"time"

	"avenue/backend/db"
	"avenue/backend/logger"
	"avenue/backend/shared"

	"github.com/spf13/afero"
)

const (
	uploadSessionRetentionEnvKey  = "UPLOAD_SESSION_RETENTION"
	defaultUploadSessionRetention = "48h"
	uploadSessionIntervalEnvKey   = "UPLOAD_SESSION_SWEEP_INTERVAL"
	defaultUploadSessionInterval  = "1h"
)

// StartUploadSessionSweeper launches a background goroutine that purges
// upload sessions which haven't received a chunk in longer than
// UPLOAD_SESSION_RETENTION (default 48h), checking every
// UPLOAD_SESSION_SWEEP_INTERVAL (default 1h).
func StartUploadSessionSweeper(fs afero.Fs) {
	interval := shared.GetEnvDuration(uploadSessionIntervalEnvKey, defaultUploadSessionInterval)
	retention := shared.GetEnvDuration(uploadSessionRetentionEnvKey, defaultUploadSessionRetention)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		sweepUploadSessions(fs, retention)
		for range ticker.C {
			sweepUploadSessions(fs, retention)
		}
	}()
}

func sweepUploadSessions(fs afero.Fs, retention time.Duration) {
	cutoff := time.Now().Add(-retention)

	sessions, err := db.ListExpiredUploadSessions(cutoff)
	if err != nil {
		logger.Errorf("upload session sweeper: list expired sessions: %v", err)
		return
	}

	for _, session := range sessions {
		if err := fs.Remove(shared.BlobPath(session.ID)); err != nil && !errors.Is(err, afero.ErrFileNotFound) {
			logger.Errorf("upload session sweeper: remove blob %s: %v", session.ID, err)
			continue
		}
		if err := db.DeleteUploadSession(session.ID, session.UserID); err != nil {
			logger.Errorf("upload session sweeper: delete session %s: %v", session.ID, err)
		}
	}

	if len(sessions) > 0 {
		logger.Infof("upload session sweeper: removed %d abandoned upload session(s)", len(sessions))
	}
}
