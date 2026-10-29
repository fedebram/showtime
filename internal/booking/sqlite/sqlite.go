package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/fedebram/showtime/internal/booking"
)

// compile time checks that the sqlite types satisfy the booking interfaces.
var (
	_ booking.Repository = (*Repository)(nil)
	_ booking.Tx         = (*txRepo)(nil)
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

	// pragma synchronous full is not so much needed.
	// but is nice to fsync each write.

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

// View runs fn in a read transaction on the read pool.
func (r *Repository) View(ctx context.Context, fn func(tx booking.Tx) error) error {
	tx, err := r.read.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// a read keeps nothing: always roll back at the end
	// commit would work the same
	defer tx.Rollback()
	return fn(&txRepo{tx: tx})
}

// Update runs fn in a write transaction on the write connection:
// committed if fn returns nil, rolled back if it returns an error.
func (r *Repository) Update(ctx context.Context, fn func(tx booking.Tx) error) error {
	tx, err := r.write.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// undoes everything if fn fails; does nothing after a successful Commit
	defer tx.Rollback()

	if err := fn(&txRepo{tx: tx}); err != nil {
		return err
	}
	return tx.Commit()
}

// txRepo does the real work with SQL, inside one transaction.
// It is only reachable through View and Update.
type txRepo struct {
	tx *sql.Tx
}

// errNotImplemented marks the Tx methods still to write.
var errNotImplemented = errors.New("not implemented")

func (t *txRepo) InsertSeat(ctx context.Context, s booking.Seat) error {
	return errNotImplemented
}

func (t *txRepo) ListSeats(ctx context.Context, roomID uuid.UUID) ([]booking.Seat, error) {
	return nil, errNotImplemented
}

func (t *txRepo) ListShows(ctx context.Context) ([]booking.Show, error) {
	return nil, errNotImplemented
}

func (t *txRepo) InsertShow(ctx context.Context, s booking.Show) error {
	return errNotImplemented
}

func (t *txRepo) GetShow(ctx context.Context, id uuid.UUID) (booking.Show, error) {
	return booking.Show{}, errNotImplemented
}

func (t *txRepo) GetBooking(ctx context.Context, id uuid.UUID) (booking.Booking, error) {
	return booking.Booking{}, errNotImplemented
}

func (t *txRepo) InsertBooking(ctx context.Context, b booking.Booking) error {
	return errNotImplemented
}

func (t *txRepo) UpdateBooking(ctx context.Context, b booking.Booking) error {
	return errNotImplemented
}

func (t *txRepo) ListReservations(ctx context.Context, bookingID uuid.UUID) ([]booking.Reservation, error) {
	return nil, errNotImplemented
}

func (t *txRepo) InsertReservation(ctx context.Context, r booking.Reservation) error {
	return errNotImplemented
}

func (t *txRepo) DeleteReservation(ctx context.Context, bookingID, reservationID uuid.UUID) error {
	return errNotImplemented
}

func (t *txRepo) ListShowReservations(ctx context.Context, showID uuid.UUID) ([]booking.ShowReservation, error) {
	return nil, errNotImplemented
}
