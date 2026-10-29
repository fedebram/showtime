package sqlite

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"
)

// the schema is copied into the binary at build time.
//
//go:embed schema.sql
var schema string

// we make a write pool where there is only one open connection. So there is no chance that sqlite responds with busy timeout
// also we enforce the single writer of sqlite at app level!
// concurrent reads on sqlite are possible (with wal mode) so we use multiple open connection (default to 4)
// This way of working with sqlite is pretty standard and is inspired by pocketbase codebase and the way it handles sqlite.
//
// see: https://github.com/pocketbase/pocketbase though i don't remember where I have read about that in the codebase.

// Repository stores everything in SQLite, with two pools: one connection for
// writes (Update) and several for reads (View).
type Repository struct {
	read  *sql.DB
	write *sql.DB
}

// Open opens the database at path, creating it if needed,
// and makes sure the tables exist.
func Open(path string) (*Repository, error) {
	// basic pragmas for sqlite. no tuning of page cache and so on.
	// it is out of the scope of the exercise

	write, err := sql.Open("sqlite", "file:"+path+
		"?_pragma=journal_mode(WAL)"+
		"&_pragma=foreign_keys(1)"+
		"&_pragma=busy_timeout(5000)"+
		"&_pragma=synchronous(FULL)"+
		"&_txlock=immediate")
	if err != nil {
		return nil, err
	}

	write.SetMaxOpenConns(1)
	write.SetMaxIdleConns(1)

	if _, err := write.Exec(schema); err != nil {
		write.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	read, err := sql.Open("sqlite", "file:"+path+
		"?_pragma=foreign_keys(1)"+
		"&_pragma=busy_timeout(5000)"+
		"&_pragma=query_only(1)")
	if err != nil {
		write.Close()
		return nil, err
	}
	read.SetMaxOpenConns(4)
	read.SetMaxIdleConns(4)

	if err := read.Ping(); err != nil {
		read.Close()
		write.Close()
		return nil, err
	}
	return &Repository{read: read, write: write}, nil
}

// Close closes both pools.
func (r *Repository) Close() error {
	return errors.Join(r.read.Close(), r.write.Close())
}
