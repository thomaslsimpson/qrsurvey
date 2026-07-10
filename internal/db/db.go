// Package db owns the SQLite connection and all SQL used by the app.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

const pragmas = "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"

// buildDSN accepts either a plain filesystem path ("data/qrsurvey.db") or an
// already-composed sqlite "file:" DSN (e.g. a test using a named in-memory
// database: "file:foo?mode=memory&cache=shared"), and appends this
// package's required pragmas either way.
func buildDSN(dbPath string) string {
	if !strings.HasPrefix(dbPath, "file:") {
		dbPath = "file:" + dbPath
	}
	sep := "?"
	if strings.Contains(dbPath, "?") {
		sep = "&"
	}
	return dbPath + sep + pragmas
}

// Open opens (creating parent directories and the file if needed) a SQLite
// database configured for a single-writer web app: WAL journal mode so
// reads and the occasional write don't block each other, foreign key
// enforcement on (SQLite disables it by default), and a busy timeout so
// concurrent writers retry instead of failing immediately.
func Open(ctx context.Context, dbPath string) (*DB, error) {
	if !strings.Contains(dbPath, "mode=memory") && dbPath != ":memory:" {
		if dir := filepath.Dir(dbPath); dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create db directory: %w", err)
			}
		}
	}

	conn, err := sql.Open("sqlite", buildDSN(dbPath))
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite serializes writes regardless; capping open conns avoids
	// SQLITE_BUSY storms under concurrent requests.
	conn.SetMaxOpenConns(8)

	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	return &DB{conn: conn}, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

// Conn exposes the underlying *sql.DB for packages (queries/) that live
// alongside this one but are split into per-entity files.
func (d *DB) Conn() *sql.DB {
	return d.conn
}
