package db

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

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

func (db *DB) Close() error {
	db.Writer.Close()
	return db.Reader.Close()
}
