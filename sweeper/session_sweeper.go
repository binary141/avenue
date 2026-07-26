// Session sweeping: periodically deletes expired/invalid rows from the
// sessions table so it doesn't grow unbounded.
package sweeper

import (
	"time"

	"avenue/backend/db"
	"avenue/backend/logger"
	"avenue/backend/shared"
)

const (
	sessionSweepIntervalEnvKey  = "SESSION_SWEEP_INTERVAL"
	defaultSessionSweepInterval = "1h"
)

// StartSessionSweeper launches a background goroutine that deletes expired
// or invalidated sessions on a fixed interval, configurable via the
// SESSION_SWEEP_INTERVAL env var (a Go duration string, e.g. "1h"; default
// "1h").
func StartSessionSweeper() {
	interval := shared.GetEnvDuration(sessionSweepIntervalEnvKey, defaultSessionSweepInterval)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		sweepSessions()
		for range ticker.C {
			sweepSessions()
		}
	}()
}

func sweepSessions() {
	removed, err := db.DeleteExpiredSessions()
	if err != nil {
		logger.Errorf("session sweeper: delete expired sessions: %v", err)
		return
	}
	if removed > 0 {
		logger.Infof("session sweeper: removed %d expired session(s)", removed)
	}
}
