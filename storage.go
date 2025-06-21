package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Storage defines the interface for persisting updates
type Storage interface {
	SaveUpdate(ctx context.Context, update Update) error
	GetLatestUpdateBySource(ctx context.Context, sourceID string) (*Update, error)
	GetUpdateByHash(ctx context.Context, hash string) (*Update, bool, error)
	GetUpdatesByDateRange(ctx context.Context, startDate, endDate time.Time) ([]Update, error)
	GetDailyStats(ctx context.Context, date time.Time) (map[string]interface{}, error)
	GetSourceStats(ctx context.Context, date time.Time) (map[string]map[string]interface{}, error)
	Close() error
}

// SQLiteStorage implements Storage using SQLite
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Printf("[STORAGE] Database initialized at %s", dbPath)
	return &SQLiteStorage{db: db}, nil
}

// initSchema creates the necessary tables
func initSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS updates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source_id TEXT NOT NULL,
		url TEXT NOT NULL,
		fetched_at TIMESTAMP NOT NULL,
		hash TEXT UNIQUE,
		status_code INTEGER NOT NULL,
		success BOOLEAN NOT NULL,
		retry_count INTEGER NOT NULL,
		error_detail TEXT,
		body_size INTEGER DEFAULT 0,
		title TEXT,
		summary TEXT,
		content_type TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_updates_source_id ON updates(source_id);
	CREATE INDEX IF NOT EXISTS idx_updates_hash ON updates(hash);
	CREATE INDEX IF NOT EXISTS idx_updates_fetched_at ON updates(fetched_at);
	CREATE INDEX IF NOT EXISTS idx_updates_date ON updates(date(fetched_at));
	`

	_, err := db.Exec(schema)
	return err
}

// SaveUpdate stores an update in the database
func (s *SQLiteStorage) SaveUpdate(ctx context.Context, update Update) error {
	// Check if we already have this hash
	if update.Hash != "" {
		_, exists, err := s.GetUpdateByHash(ctx, update.Hash)
		if err != nil {
			return fmt.Errorf("failed to check existing hash: %w", err)
		}

		if exists {
			// Update already exists, skip insertion
			log.Printf("[STORAGE] Skipping duplicate update for hash %s (source: %s)",
				update.Hash[:8], update.SourceID)
			return nil
		}
	}

	query := `
	INSERT INTO updates 
	(source_id, url, fetched_at, hash, status_code, success, retry_count, error_detail, body_size, title, summary, content_type)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	bodySize := len(update.Body)
	_, err := s.db.ExecContext(
		ctx,
		query,
		update.SourceID,
		update.URL,
		update.FetchedAt.Format(time.RFC3339),
		update.Hash,
		update.StatusCode,
		update.Success,
		update.RetryCount,
		update.ErrorDetail,
		bodySize,
		update.Title,
		update.Summary,
		update.ContentType,
	)

	if err != nil {
		return fmt.Errorf("failed to save update: %w", err)
	}

	return nil
}

// GetLatestUpdateBySource retrieves the latest successful update for a given source
func (s *SQLiteStorage) GetLatestUpdateBySource(ctx context.Context, sourceID string) (*Update, error) {
	query := `
	SELECT source_id, url, fetched_at, COALESCE(hash, '') as hash, status_code, success, retry_count, 
	       COALESCE(error_detail, '') as error_detail
	FROM updates
	WHERE source_id = ? AND success = 1
	ORDER BY fetched_at DESC
	LIMIT 1
	`

	row := s.db.QueryRowContext(ctx, query, sourceID)

	var update Update
	var fetchedAt string

	err := row.Scan(
		&update.SourceID,
		&update.URL,
		&fetchedAt,
		&update.Hash,
		&update.StatusCode,
		&update.Success,
		&update.RetryCount,
		&update.ErrorDetail,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get latest update: %w", err)
	}

	update.FetchedAt, err = time.Parse(time.RFC3339, fetchedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse fetched_at: %w", err)
	}

	return &update, nil
}

// GetUpdateByHash retrieves an update by its hash
func (s *SQLiteStorage) GetUpdateByHash(ctx context.Context, hash string) (*Update, bool, error) {
	if hash == "" {
		return nil, false, nil
	}

	query := `
	SELECT source_id, url, fetched_at, hash, status_code, success, retry_count, 
	       COALESCE(error_detail, '') as error_detail
	FROM updates
	WHERE hash = ?
	LIMIT 1
	`

	row := s.db.QueryRowContext(ctx, query, hash)

	var update Update
	var fetchedAt string

	err := row.Scan(
		&update.SourceID,
		&update.URL,
		&fetchedAt,
		&update.Hash,
		&update.StatusCode,
		&update.Success,
		&update.RetryCount,
		&update.ErrorDetail,
	)

	if err == sql.ErrNoRows {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, fmt.Errorf("failed to get update by hash: %w", err)
	}

	update.FetchedAt, err = time.Parse(time.RFC3339, fetchedAt)
	if err != nil {
		return nil, true, fmt.Errorf("failed to parse fetched_at: %w", err)
	}

	return &update, true, nil
}

// GetUpdatesByDateRange retrieves all updates within a date range
func (s *SQLiteStorage) GetUpdatesByDateRange(ctx context.Context, startDate, endDate time.Time) ([]Update, error) {
	query := `
	SELECT source_id, url, fetched_at, COALESCE(hash, '') as hash, status_code, success, retry_count, 
	       COALESCE(error_detail, '') as error_detail, COALESCE(title, '') as title, 
	       COALESCE(summary, '') as summary, COALESCE(content_type, '') as content_type
	FROM updates
	WHERE date(fetched_at) >= date(?) AND date(fetched_at) <= date(?)
	ORDER BY fetched_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to query updates: %w", err)
	}
	defer rows.Close()

	var updates []Update
	for rows.Next() {
		var update Update
		var fetchedAt string

		err := rows.Scan(
			&update.SourceID,
			&update.URL,
			&fetchedAt,
			&update.Hash,
			&update.StatusCode,
			&update.Success,
			&update.RetryCount,
			&update.ErrorDetail,
			&update.Title,
			&update.Summary,
			&update.ContentType,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan update: %w", err)
		}

		update.FetchedAt, err = time.Parse(time.RFC3339, fetchedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse fetched_at: %w", err)
		}

		updates = append(updates, update)
	}

	return updates, nil
}

// GetDailyStats retrieves daily statistics for reporting
func (s *SQLiteStorage) GetDailyStats(ctx context.Context, date time.Time) (map[string]interface{}, error) {
	query := `
	SELECT 
		COUNT(*) as total_updates,
		COUNT(CASE WHEN success = 1 THEN 1 END) as successful_updates,
		COUNT(CASE WHEN success = 0 THEN 1 END) as failed_updates,
		COUNT(DISTINCT source_id) as unique_sources,
		SUM(body_size) as total_size
	FROM updates
	WHERE date(fetched_at) = date(?)
	`

	row := s.db.QueryRowContext(ctx, query, date.Format("2006-01-02"))

	var totalUpdates, successfulUpdates, failedUpdates, uniqueSources int
	var totalSize sql.NullInt64

	err := row.Scan(&totalUpdates, &successfulUpdates, &failedUpdates, &uniqueSources, &totalSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily stats: %w", err)
	}

	stats := map[string]interface{}{
		"date":               date.Format("2006-01-02"),
		"total_updates":      totalUpdates,
		"successful_updates": successfulUpdates,
		"failed_updates":     failedUpdates,
		"unique_sources":     uniqueSources,
		"total_size":         totalSize.Int64,
	}

	return stats, nil
}

// GetSourceStats retrieves statistics by source for a given date
func (s *SQLiteStorage) GetSourceStats(ctx context.Context, date time.Time) (map[string]map[string]interface{}, error) {
	query := `
	SELECT 
		source_id,
		COUNT(*) as total_updates,
		COUNT(CASE WHEN success = 1 THEN 1 END) as successful_updates,
		COUNT(CASE WHEN success = 0 THEN 1 END) as failed_updates,
		SUM(body_size) as total_size
	FROM updates
	WHERE date(fetched_at) = date(?)
	GROUP BY source_id
	ORDER BY total_updates DESC
	`

	rows, err := s.db.QueryContext(ctx, query, date.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to query source stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]map[string]interface{})
	for rows.Next() {
		var sourceID string
		var totalUpdates, successfulUpdates, failedUpdates int
		var totalSize sql.NullInt64

		err := rows.Scan(&sourceID, &totalUpdates, &successfulUpdates, &failedUpdates, &totalSize)
		if err != nil {
			return nil, fmt.Errorf("failed to scan source stats: %w", err)
		}

		stats[sourceID] = map[string]interface{}{
			"total_updates":      totalUpdates,
			"successful_updates": successfulUpdates,
			"failed_updates":     failedUpdates,
			"total_size":         totalSize.Int64,
		}
	}

	return stats, nil
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
