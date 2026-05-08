package db

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

// DB encapsulates a split-connection pool for SQLite.
// SQLite in WAL mode allows multiple readers but only ONE concurrent writer.
// We split the pools to ensure mutations are serialized (MaxOpenConns=1) 
// while lookups remain highly concurrent.
type DB struct {
	Writer *sql.DB
	Reader *sql.DB
}

// InitDB initializes the SQLite connections with WAL mode and appropriate pool limits.
func InitDB(dbPath string) (*DB, error) {
	// DSN: WAL mode for concurrency, NORMAL sync for performance, and 5s busy timeout.
	dsn := dbPath + "?_journal=WAL&_synchronous=NORMAL&_busy_timeout=5000"

	writer, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	// WRITER: Set to 1 to prevent "database is locked" errors by serializing all writes.
	writer.SetMaxOpenConns(1)

	_, err = writer.Exec(schemaSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to execute schema: %w", err)
	}

	// Automatic Migrations
	if err := applyMigrations(writer); err != nil {
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	reader, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	// READER: Set to 10 to allow concurrent read-heavy operations (e.g. status checks).
	reader.SetMaxOpenConns(10)

	return &DB{
		Writer: writer,
		Reader: reader,
	}, nil
}

func applyMigrations(db *sql.DB) error {
	// 1. Add meta_error_message to jobs table if it doesn't exist
	var exists bool
	query := "SELECT count(*) FROM pragma_table_info('jobs') WHERE name='meta_error_message'"
	err := db.QueryRow(query).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		_, err = db.Exec("ALTER TABLE jobs ADD COLUMN meta_error_message TEXT")
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) Close() error {
	db.Writer.Close()
	return db.Reader.Close()
}
