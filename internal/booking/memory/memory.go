package memory

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/fedebram/showtime/internal/booking"
)

// compile time checks that the memory types satisfy the booking interfaces.
var (
	_ booking.Repository = (*Repository)(nil)
	_ booking.Tx         = (*txRepo)(nil)
)

// Repository keeps everything in memory.
// Concurrent reads (View).
// One writer (Update) at the time.
type Repository struct {
	mu   sync.RWMutex
	data txRepo
}

func NewRepository() *Repository {
	return &Repository{
		data: txRepo{
			shows:        make(map[uuid.UUID]booking.Show),
			bookings:     make(map[uuid.UUID]booking.Booking),
			reservations: make(map[uuid.UUID]booking.Reservation),
			seats:        make(map[uuid.UUID]booking.Seat),
		},
	}
}

// in memory repository implementation doesn't need context

// View and Update functions take inspiration from the famous BoltDB of Ben Johnson
// and its maintained fork of etcd bbolt.
// see: https://github.com/etcd-io/bbolt
// of course our implementation is just an in memory kv store but the abstraction is just nice.

// View and Update takes a callback function. The real operations happen inside the function via
// the tx object. So multiple repository calls are grouped as one transaction and rolled back together.

func (r *Repository) View(ctx context.Context, fn func(tx booking.Tx) error) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return fn(&r.data)
}

func (r *Repository) Update(ctx context.Context, fn func(tx booking.Tx) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	backup := r.data.clone()
	if err := fn(&r.data); err != nil {
		// undo everything fn did. rollback
		r.data = backup
		return err
	}
	return nil
}

// txRepo holds the data and does the real work. It never locks:
// it is only reachable through View and Update, which hold the mutex.
type txRepo struct {
	shows        map[uuid.UUID]booking.Show
	bookings     map[uuid.UUID]booking.Booking
	reservations map[uuid.UUID]booking.Reservation
	seats        map[uuid.UUID]booking.Seat
}

// clone copies the maps, so a failed Update can be undone.
// like a real rollback transaction.
// clone the entire repo is fine.
func (t *txRepo) clone() txRepo {
	return txRepo{
		shows:        maps.Clone(t.shows),
		bookings:     maps.Clone(t.bookings),
		reservations: maps.Clone(t.reservations),
		seats:        maps.Clone(t.seats),
	}
}

func (t *txRepo) InsertSeat(ctx context.Context, s booking.Seat) error {
	t.seats[s.ID] = s
	return nil
}

func (t *txRepo) ListSeats(ctx context.Context, roomID uuid.UUID) ([]booking.Seat, error) {
	seats := make([]booking.Seat, 0)
	for _, seat := range t.seats {
		if seat.RoomID == roomID {
			seats = append(seats, seat)
		}
	}
	// by row and then by number: A1, A2, A3, B1...
	slices.SortFunc(seats, func(a, b booking.Seat) int {
		if a.Row != b.Row {
			return strings.Compare(a.Row, b.Row)
		}
		return a.Number - b.Number
	})
	return seats, nil
}

func (t *txRepo) ListShows(ctx context.Context) ([]booking.Show, error) {
	shows := make([]booking.Show, 0, len(t.shows))
	for _, show := range t.shows {
		shows = append(shows, show)
	}
	// for show we use v7 uuids that are time ordered, so this is creation order
	slices.SortFunc(shows, func(a, b booking.Show) int {
		return bytes.Compare(a.ID[:], b.ID[:])
	})
	return shows, nil
}

func (t *txRepo) InsertShow(ctx context.Context, s booking.Show) error {
	t.shows[s.ID] = s
	return nil
}

func (t *txRepo) GetShow(ctx context.Context, id uuid.UUID) (booking.Show, error) {
	show, ok := t.shows[id]
	if !ok {
		return booking.Show{}, fmt.Errorf("%w: %s", booking.ErrShowNotFound, id)
	}
	return show, nil
}

func (t *txRepo) GetBooking(ctx context.Context, id uuid.UUID) (booking.Booking, error) {
	b, ok := t.bookings[id]
	if !ok {
		return booking.Booking{}, fmt.Errorf("%w: %s", booking.ErrBookingNotFound, id)
	}
	return b, nil
}

func (t *txRepo) InsertBooking(ctx context.Context, b booking.Booking) error {
	t.bookings[b.ID] = b
	return nil
}

func (t *txRepo) UpdateBooking(ctx context.Context, b booking.Booking) error {
	if _, ok := t.bookings[b.ID]; !ok {
		return fmt.Errorf("%w: %s", booking.ErrBookingNotFound, b.ID)
	}
	t.bookings[b.ID] = b
	return nil
}

func (t *txRepo) ListReservations(ctx context.Context, bookingID uuid.UUID) ([]booking.Reservation, error) {
	reservations := make([]booking.Reservation, 0)
	for _, res := range t.reservations {
		if res.BookingID == bookingID {
			reservations = append(reservations, res)
		}
	}
	slices.SortFunc(reservations, func(a, b booking.Reservation) int {
		return bytes.Compare(a.ID[:], b.ID[:])
	})
	return reservations, nil
}

func (t *txRepo) InsertReservation(ctx context.Context, res booking.Reservation) error {
	t.reservations[res.ID] = res
	return nil
}

func (t *txRepo) DeleteReservation(ctx context.Context, bookingID, reservationID uuid.UUID) error {
	// it must belong to this booking, otherwise you could delete someone else's
	res, ok := t.reservations[reservationID]
	if !ok || res.BookingID != bookingID {
		return fmt.Errorf("%w: %s", booking.ErrReservationNotFound, reservationID)
	}
	delete(t.reservations, reservationID)
	return nil
}

func (t *txRepo) ListShowReservations(ctx context.Context, showID uuid.UUID) ([]booking.ShowReservation, error) {
	result := make([]booking.ShowReservation, 0)
	for _, res := range t.reservations {
		if res.ShowID == showID {
			result = append(result, booking.ShowReservation{
				Reservation: res,
				Booking:     t.bookings[res.BookingID],
			})
		}
	}
	return result, nil
}
