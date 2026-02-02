package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

type DB struct {
	conn *sql.DB
}

type SyncRecord struct {
	ID        int
	Image     string
	Status    string
	Bytes     int64
	Duration  float64
	Timestamp time.Time
}

func NewDB() (*DB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dbPath := filepath.Join(home, ".registry-mirror.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := initSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return &DB{conn: db}, nil
}

func initSchema(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS syncs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		image TEXT NOT NULL,
		status TEXT NOT NULL,
		bytes INTEGER DEFAULT 0,
		duration REAL DEFAULT 0,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_image ON syncs(image);
	`
	_, err := db.Exec(query)
	return err
}

func (db *DB) RecordSync(image, status string, bytes int64, duration float64) error {
	query := `INSERT INTO syncs (image, status, bytes, duration, timestamp) VALUES (?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, image, status, bytes, duration, time.Now())
	return err
}

func (db *DB) GetRecentSyncs(limit int) ([]SyncRecord, error) {
	query := `SELECT id, image, status, bytes, duration, timestamp FROM syncs ORDER BY timestamp DESC LIMIT ?`
	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []SyncRecord
	for rows.Next() {
		var rec SyncRecord
		if err := rows.Scan(&rec.ID, &rec.Image, &rec.Status, &rec.Bytes, &rec.Duration, &rec.Timestamp); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}

func (db *DB) GetLatestSync(image string) (*SyncRecord, error) {
	query := `SELECT id, image, status, bytes, duration, timestamp FROM syncs WHERE image = ? ORDER BY timestamp DESC LIMIT 1`
	row := db.conn.QueryRow(query, image)

	var rec SyncRecord
	if err := row.Scan(&rec.ID, &rec.Image, &rec.Status, &rec.Bytes, &rec.Duration, &rec.Timestamp); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}
