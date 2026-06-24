package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"
)

func ApplyMigrations(ctx context.Context, database *gorm.DB, migrationsDir string) error {
	if database == nil {
		return fmt.Errorf("database is required")
	}

	paths, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(paths)

	if len(paths) == 0 {
		return fmt.Errorf("no migrations found in %s", migrationsDir)
	}

	return database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			CREATE TABLE IF NOT EXISTS schema_migrations (
				version TEXT PRIMARY KEY,
				applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
			)
		`).Error; err != nil {
			return err
		}

		for _, path := range paths {
			version := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			applied, err := migrationApplied(tx, version)
			if err != nil {
				return err
			}
			if applied {
				continue
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read migration %s: %w", path, err)
			}

			if err := tx.Exec(string(content)).Error; err != nil {
				return fmt.Errorf("apply migration %s: %w", version, err)
			}

			if err := tx.Exec(`
				INSERT INTO schema_migrations (version)
				VALUES (?)
			`, version).Error; err != nil {
				return fmt.Errorf("record migration %s: %w", version, err)
			}
		}

		return nil
	})
}

func migrationApplied(tx *gorm.DB, version string) (bool, error) {
	var count int64
	if err := tx.Raw(`
		SELECT COUNT(*)
		FROM schema_migrations
		WHERE version = ?
	`, version).Scan(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}
