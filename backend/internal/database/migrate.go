package database

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MigrationFile struct {
	Version int
	Name    string
	Path    string
}

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	slog.Info("checking database migrations...")

	// 1. Create schema_migrations tracking table if not exists
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// 2. Locate migrations directory
	searchDirs := []string{
		"db/migrations",
		"../db/migrations",
		"../../db/migrations",
		"/app/db/migrations",
	}

	var migDir string
	for _, dir := range searchDirs {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			migDir = dir
			break
		}
	}

	if migDir == "" {
		slog.Warn("no migration directory found, skipping auto migration")
		return nil
	}

	entries, err := os.ReadDir(migDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory %s: %w", migDir, err)
	}

	var upFiles []MigrationFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) < 2 {
			continue
		}
		version, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		upFiles = append(upFiles, MigrationFile{
			Version: version,
			Name:    entry.Name(),
			Path:    filepath.Join(migDir, entry.Name()),
		})
	}

	sort.Slice(upFiles, func(i, j int) bool {
		return upFiles[i].Version < upFiles[j].Version
	})

	for _, mig := range upFiles {
		var exists bool
		err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", mig.Version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to query migration status for %s: %w", mig.Name, err)
		}

		if exists {
			continue
		}

		slog.Info("applying migration", slog.Int("version", mig.Version), slog.String("name", mig.Name))
		sqlContent, err := os.ReadFile(mig.Path)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", mig.Path, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", mig.Name, err)
		}

		if _, err := tx.Exec(ctx, string(sqlContent)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to execute migration %s: %w", mig.Name, err)
		}

		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", mig.Version, mig.Name); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to record applied migration %s: %w", mig.Name, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration transaction %s: %w", mig.Name, err)
		}
		slog.Info("migration applied successfully", slog.Int("version", mig.Version), slog.String("name", mig.Name))
	}

	slog.Info("database migrations up to date")
	return nil
}
