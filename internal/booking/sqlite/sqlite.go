package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

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

// unfortunately sqlite doesn't have a built in time. so we need to format ourselves.

// timeFormat is how times are stored: always UTC, always with nanoseconds.
// Every stored time has the same length, so sorting the text sorts by time.
const timeFormat = "2006-01-02T15:04:05.000000000Z"

func formatTime(t time.Time) string {
	return t.UTC().Format(timeFormat)
}

func parseTime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}

func (t *txRepo) InsertRoom(ctx context.Context, r booking.Room) error {
	_, err := t.tx.ExecContext(ctx,
		`INSERT INTO rooms (id, name) VALUES (?, ?)`,
		r.ID, r.Name)
	return err
}

func (t *txRepo) InsertSeat(ctx context.Context, s booking.Seat) error {
	_, err := t.tx.ExecContext(ctx,
		`INSERT INTO seats (id, room_id, seat_row, seat_number) VALUES (?, ?, ?, ?)`,
		s.ID, s.RoomID, s.Row, s.Number)
	return err
}

func (t *txRepo) ListSeats(ctx context.Context, roomID uuid.UUID) ([]booking.Seat, error) {
	// like the memory repo, we order the seats
	// by row and by number!

	rows, err := t.tx.QueryContext(ctx,
		`SELECT id, room_id, seat_row, seat_number
		 FROM seats
		 WHERE room_id = ?
		 ORDER BY seat_row, seat_number`,
		roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seats := make([]booking.Seat, 0)
	for rows.Next() {
		var s booking.Seat
		if err := rows.Scan(&s.ID, &s.RoomID, &s.Row, &s.Number); err != nil {
			return nil, err
		}
		seats = append(seats, s)
	}
	return seats, rows.Err()
}

func (t *txRepo) ListShows(ctx context.Context) ([]booking.Show, error) {
	// v7 uuids written as text sort by creation time, like the memory repo
	rows, err := t.tx.QueryContext(ctx,
		`SELECT id, movie_title, room_id, starts_at FROM shows ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shows := make([]booking.Show, 0)
	for rows.Next() {
		var s booking.Show
		// starts_at is text in the database: scan it as a string, then parse it
		var startsAt string
		if err := rows.Scan(&s.ID, &s.MovieTitle, &s.RoomID, &startsAt); err != nil {
			return nil, err
		}
		if s.StartsAt, err = parseTime(startsAt); err != nil {
			return nil, err
		}
		shows = append(shows, s)
	}
	return shows, rows.Err()
}

func (t *txRepo) InsertShow(ctx context.Context, s booking.Show) error {
	_, err := t.tx.ExecContext(ctx,
		`INSERT INTO shows (id, movie_title, room_id, starts_at) VALUES (?, ?, ?, ?)`,
		s.ID, s.MovieTitle, s.RoomID, formatTime(s.StartsAt))
	return err
}

func (t *txRepo) GetShow(ctx context.Context, id uuid.UUID) (booking.Show, error) {
	var s booking.Show
	var startsAt string
	err := t.tx.QueryRowContext(ctx,
		`SELECT id, movie_title, room_id, starts_at FROM shows WHERE id = ?`,
		id).Scan(&s.ID, &s.MovieTitle, &s.RoomID, &startsAt)
	// no row is the sql version of a missing map key
	if errors.Is(err, sql.ErrNoRows) {
		return booking.Show{}, fmt.Errorf("%w: %s", booking.ErrShowNotFound, id)
	}
	if err != nil {
		return booking.Show{}, err
	}

	if s.StartsAt, err = parseTime(startsAt); err != nil {
		return booking.Show{}, err
	}
	return s, nil
}

func (t *txRepo) GetBooking(ctx context.Context, id uuid.UUID) (booking.Booking, error) {
	var b booking.Booking
	// status and the times are text in the database: scan them as strings, then convert
	var status, createdAt, expiresAt string
	err := t.tx.QueryRowContext(ctx,
		`SELECT id, status, created_at, expires_at, token_hash FROM bookings WHERE id = ?`,
		id).Scan(&b.ID, &status, &createdAt, &expiresAt, &b.TokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return booking.Booking{}, fmt.Errorf("%w: %s", booking.ErrBookingNotFound, id)
	}
	if err != nil {
		return booking.Booking{}, err
	}

	b.Status = booking.BookingStatus(status)
	if b.CreatedAt, err = parseTime(createdAt); err != nil {
		return booking.Booking{}, err
	}
	if b.ExpiresAt, err = parseTime(expiresAt); err != nil {
		return booking.Booking{}, err
	}
	return b, nil
}

func (t *txRepo) InsertBooking(ctx context.Context, b booking.Booking) error {
	_, err := t.tx.ExecContext(ctx,
		`INSERT INTO bookings (id, status, created_at, expires_at, token_hash)
		 VALUES (?, ?, ?, ?, ?)`,
		b.ID, string(b.Status), formatTime(b.CreatedAt), formatTime(b.ExpiresAt), b.TokenHash)
	return err
}

func (t *txRepo) UpdateBooking(ctx context.Context, b booking.Booking) error {
	res, err := t.tx.ExecContext(ctx,
		`UPDATE bookings
		 SET status = ?, created_at = ?, expires_at = ?, token_hash = ?
		 WHERE id = ?`,
		string(b.Status), formatTime(b.CreatedAt), formatTime(b.ExpiresAt), b.TokenHash, b.ID)
	if err != nil {
		return err
	}
	// the update ran, but changed nothing. there is no such booking.
	// SQL UPDATE that matches no row still succeeds!! 0 rows changed
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%w: %s", booking.ErrBookingNotFound, b.ID)
	}
	return nil
}

func (t *txRepo) ListReservations(ctx context.Context, bookingID uuid.UUID) ([]booking.Reservation, error) {
	// we return one row per seat.
	// left join because this way we enforce the contract test that might generate a reservation without seats.
	// but in reality the app creates reservations always with seats (seat ids)
	rows, err := t.tx.QueryContext(ctx,
		`SELECT r.id, r.booking_id, r.show_id, rs.seat_id
		 FROM reservations r
		 LEFT JOIN reservation_seats rs ON rs.reservation_id = r.id
		 WHERE r.booking_id = ?
		 ORDER BY r.id, rs.rowid`,
		bookingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reservations := make([]booking.Reservation, 0)
	for rows.Next() {
		var res booking.Reservation
		// NULL when the reservation has no seats
		var seat uuid.NullUUID
		if err := rows.Scan(&res.ID, &res.BookingID, &res.ShowID, &seat); err != nil {
			return nil, err
		}
		// a new reservation starts at the first row with a new id
		if n := len(reservations); n == 0 || reservations[n-1].ID != res.ID {
			reservations = append(reservations, res)
		}
		// we place the seat id on the reservation
		if seat.Valid {
			last := &reservations[len(reservations)-1]
			last.SeatIDs = append(last.SeatIDs, seat.UUID)
		}
	}
	return reservations, rows.Err()
}

func (t *txRepo) InsertReservation(ctx context.Context, r booking.Reservation) error {
	_, err := t.tx.ExecContext(ctx,
		`INSERT INTO reservations (id, booking_id, show_id) VALUES (?, ?, ?)`,
		r.ID, r.BookingID, r.ShowID)
	if err != nil {
		return err
	}
	// the SeatIDs list: one row per seat
	for _, seatID := range r.SeatIDs {
		_, err := t.tx.ExecContext(ctx,
			`INSERT INTO reservation_seats (reservation_id, seat_id) VALUES (?, ?)`,
			r.ID, seatID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *txRepo) DeleteReservation(ctx context.Context, bookingID, reservationID uuid.UUID) error {
	// booking_id in the WHERE: you can't delete someone else's reservation.
	// ON DELETE CASCADE removes its seat links.
	res, err := t.tx.ExecContext(ctx,
		`DELETE FROM reservations WHERE id = ? AND booking_id = ?`,
		reservationID, bookingID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%w: %s", booking.ErrReservationNotFound, reservationID)
	}
	return nil
}

func (t *txRepo) ListShowReservations(ctx context.Context, showID uuid.UUID) ([]booking.ShowReservation, error) {
	// like ListReservations, with the booking of each reservation joined in
	rows, err := t.tx.QueryContext(ctx,
		`SELECT r.id, r.booking_id, r.show_id, rs.seat_id,
		        b.id, b.status, b.created_at, b.expires_at, b.token_hash
		 FROM reservations r
		 JOIN bookings b ON b.id = r.booking_id
		 LEFT JOIN reservation_seats rs ON rs.reservation_id = r.id
		 WHERE r.show_id = ?
		 ORDER BY r.id, rs.rowid`,
		showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]booking.ShowReservation, 0)
	for rows.Next() {
		var sr booking.ShowReservation
		// NULL when the reservation has no seats
		var seat uuid.NullUUID
		// the booking's status and times are text: scan them as strings, then convert
		var status, createdAt, expiresAt string
		err := rows.Scan(&sr.ID, &sr.BookingID, &sr.ShowID, &seat,
			&sr.Booking.ID, &status, &createdAt, &expiresAt, &sr.Booking.TokenHash)
		if err != nil {
			return nil, err
		}

		// a new reservation starts at the first row with a new id
		if n := len(result); n == 0 || result[n-1].ID != sr.ID {
			sr.Booking.Status = booking.BookingStatus(status)
			if sr.Booking.CreatedAt, err = parseTime(createdAt); err != nil {
				return nil, err
			}
			if sr.Booking.ExpiresAt, err = parseTime(expiresAt); err != nil {
				return nil, err
			}
			result = append(result, sr)
		}
		// we place the seat id on the reservation
		if seat.Valid {
			last := &result[len(result)-1]
			last.SeatIDs = append(last.SeatIDs, seat.UUID)
		}
	}
	return result, rows.Err()
}
