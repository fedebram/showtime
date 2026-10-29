package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/fedebram/showtime/internal/notification"
)

// compile time check that Repository satisfies the notification interface.
var _ notification.Repository = (*Repository)(nil)

// the schema is copied into the binary at build time.
//
//go:embed schema.sql
var schema string

// Repository stores the messages in SQLite.
type Repository struct {
	db *sql.DB
}

// Open opens the database at path, creating it if needed,
// and makes sure the table exists.
func Open(path string) (*Repository, error) {
	db, err := sql.Open("sqlite", "file:"+path+
		"?_pragma=journal_mode(WAL)"+
		"&_pragma=busy_timeout(5000)"+
		"&_pragma=synchronous(FULL)")
	if err != nil {
		return nil, err
	}
	// one pool for this service is enough!
	// we have already showed a nice and more production ready setup on the booking service
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}
	return &Repository{db: db}, nil
}

// Close closes the database.
func (r *Repository) Close() error {
	return r.db.Close()
}

// maybe these functions might be shared with sqlite booking service...

// timeFormat is how times are stored: always UTC, always with nanoseconds.
// Every stored time has the same length, so sorting the text sorts by time.
const timeFormat = "2006-01-02T15:04:05.000000000Z"

func (r *Repository) Seen(ctx context.Context, id uuid.UUID) (bool, error) {
	var seen bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM notifications WHERE id = ?)`, id).Scan(&seen)
	return seen, err
}

func (r *Repository) Insert(ctx context.Context, m notification.Message) error {
	// a repeat hits the primary key: DO NOTHING keeps the first one, no error
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO notifications (id, kind, booking_id, sent_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT (id) DO NOTHING`,
		m.ID, m.Kind, m.BookingID, m.SentAt.UTC().Format(timeFormat))
	return err
}

func (r *Repository) List(ctx context.Context, limit int) ([]notification.Message, error) {
	// rowid grows with each insert: rowid DESC is newest first,
	// in the same order as the memory repository
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, kind, booking_id, sent_at FROM notifications
		 ORDER BY rowid DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	msgs := make([]notification.Message, 0)
	for rows.Next() {
		var m notification.Message
		// sent_at is text in the database: scan it as a string, then parse it
		var sentAt string
		if err := rows.Scan(&m.ID, &m.Kind, &m.BookingID, &sentAt); err != nil {
			return nil, err
		}
		if m.SentAt, err = time.Parse(timeFormat, sentAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
