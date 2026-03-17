package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	"dclean/internal/domain"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

func New() (*Store, error) {
	dbPath := resolveDBPath()
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() {
	s.db.Close()
}

func (s *Store) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS scan_paths (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			path       TEXT    NOT NULL UNIQUE,
			label      TEXT    NOT NULL DEFAULT '',
			active     INTEGER NOT NULL DEFAULT 1,
			created_at TEXT    NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS deletion_history (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			path       TEXT    NOT NULL,
			category   TEXT    NOT NULL,
			size_bytes INTEGER NOT NULL,
			deleted_at TEXT    NOT NULL DEFAULT (datetime('now'))
		)`,
	}

	for _, migration := range migrations {
		if _, err := s.db.Exec(migration); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) AddPath(path, label string) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO scan_paths (path, label) VALUES (?, ?)`,
		path, label,
	)
	return err
}

func (s *Store) RemovePath(id int64) error {
	_, err := s.db.Exec(`DELETE FROM scan_paths WHERE id = ?`, id)
	return err
}

func (s *Store) TogglePath(id int64) error {
	_, err := s.db.Exec(`UPDATE scan_paths SET active = NOT active WHERE id = ?`, id)
	return err
}

func (s *Store) ListPaths() ([]domain.ScanPath, error) {
	return s.queryPaths(`SELECT id, path, label, active, created_at FROM scan_paths ORDER BY label, path`)
}

func (s *Store) ActivePaths() ([]domain.ScanPath, error) {
	return s.queryPaths(`SELECT id, path, label, active, created_at FROM scan_paths WHERE active = 1 ORDER BY label, path`)
}

func (s *Store) queryPaths(query string) ([]domain.ScanPath, error) {
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paths []domain.ScanPath
	for rows.Next() {
		var sp domain.ScanPath
		var createdStr string
		if err := rows.Scan(&sp.ID, &sp.Path, &sp.Label, &sp.Active, &createdStr); err != nil {
			continue
		}
		sp.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdStr)
		paths = append(paths, sp)
	}
	return paths, nil
}

func (s *Store) HasPaths() bool {
	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM scan_paths`).Scan(&count)
	return count > 0
}

func (s *Store) SeedDefaults() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	candidates := []struct{ path, label string }{
		{filepath.Join(home, "Documentos"), "Documentos"},
		{filepath.Join(home, "Documents"), "Documents"},
		{filepath.Join(home, "Projects"), "Projects"},
		{filepath.Join(home, "dev"), "dev"},
	}

	for _, c := range candidates {
		if info, err := os.Stat(c.path); err == nil && info.IsDir() {
			s.AddPath(c.path, c.label)
		}
	}

	return nil
}

func (s *Store) RecordDeletion(record domain.DeletionRecord) error {
	_, err := s.db.Exec(
		`INSERT INTO deletion_history (path, category, size_bytes) VALUES (?, ?, ?)`,
		record.Path, record.Category, record.SizeBytes,
	)
	return err
}

func (s *Store) DeletionSummaries() ([]domain.DeletionSummary, error) {
	rows, err := s.db.Query(`
		SELECT category, SUM(size_bytes), COUNT(*), MAX(deleted_at)
		FROM deletion_history
		GROUP BY category
		ORDER BY SUM(size_bytes) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []domain.DeletionSummary
	for rows.Next() {
		var ds domain.DeletionSummary
		if err := rows.Scan(&ds.Category, &ds.TotalSize, &ds.DirCount, &ds.LastDelete); err != nil {
			continue
		}
		summaries = append(summaries, ds)
	}
	return summaries, nil
}

func resolveDBPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "dclean", "dclean.db")
}
