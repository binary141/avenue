package db

import (
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"avenue/backend/logger"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func RunMigrations() error {
	if err := ensureMigrationsTable(); err != nil {
		return fmt.Errorf("migrations: ensure table: %w", err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("migrations: read dir: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	for _, filename := range names {
		name := strings.TrimSuffix(filename, ".sql")

		applied, err := isMigrationApplied(name)
		if err != nil {
			return fmt.Errorf("migrations: check %s: %w", name, err)
		}
		if applied {
			continue
		}

		path := "migrations/" + filename
		sql, err := migrationFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("migrations: read %s: %w", path, err)
		}

		if err := applyMigration(name, string(sql)); err != nil {
			return fmt.Errorf("migrations: apply %s: %w", name, err)
		}

		logger.Infof("migration applied: %s", name)
	}

	return nil
}

func ensureMigrationsTable() error {
	_, err := DB.Exec(`CREATE TABLE IF NOT EXISTS migrations (
		name       VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL DEFAULT now()
	)`)
	return err
}

func isMigrationApplied(name string) (bool, error) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM migrations WHERE name = $1", name).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func applyMigration(name, body string) error {
	if _, err := DB.Exec(body); err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	if _, err := DB.Exec("INSERT INTO migrations (name) VALUES ($1)", name); err != nil {
		return fmt.Errorf("record: %w", err)
	}

	return nil
}
