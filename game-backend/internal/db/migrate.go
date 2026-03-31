package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("migration pool is nil")
	}

	migrationsDir, err := findMigrationsDir()
	if err != nil {
		return err
	}

	files, err := migrationFiles(migrationsDir)
	if err != nil {
		return err
	}

	for _, fileName := range files {
		migrationPath := filepath.Join(migrationsDir, fileName)
		contents, err := os.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", migrationPath, err)
		}

		if _, err := pool.Exec(ctx, string(contents)); err != nil {
			return fmt.Errorf("execute migration %s: %w", fileName, err)
		}
	}

	return nil
}

func findMigrationsDir() (string, error) {
	candidates := []string{
		"game-backend/migrations",
		"../migrations",
		"migrations",
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("migrations directory not found")
}

func migrationFiles(migrationsDir string) ([]string, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("read migrations directory %s: %w", migrationsDir, err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".sql") {
			files = append(files, entry.Name())
		}
	}

	sort.Strings(files)
	return files, nil
}
