package memory

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/google/uuid"

	"github.com/fedebram/showtime/internal/booking"
)

// compile time check that memory repository satisfies the booking repository interface.
var _ booking.Repository = (*Repository)(nil)

type Repository struct {
	mu           sync.Mutex
	shows        map[uuid.UUID]booking.Show
	bookings     map[uuid.UUID]booking.Booking
	reservations map[uuid.UUID]booking.Reservation
}

func NewRepository() *Repository {
	return &Repository{
		shows:        make(map[uuid.UUID]booking.Show),
		bookings:     make(map[uuid.UUID]booking.Booking),
		reservations: make(map[uuid.UUID]booking.Reservation),
	}
}

func (r *Repository) ListShows(ctx context.Context) ([]booking.Show, error) {
	// in memory repository implementation doesn't need context

	r.mu.Lock()
	defer r.mu.Unlock()

	shows := make([]booking.Show, 0, len(r.shows))
	for _, show := range r.shows {
		shows = append(shows, show)
	}
	// for show we use v7 uuids that are time ordered, so this is creation order
	slices.SortFunc(shows, func(a, b booking.Show) int {
		return bytes.Compare(a.ID[:], b.ID[:])
	})
	return shows, nil
}

func (r *Repository) InsertShow(ctx context.Context, s booking.Show) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.shows[s.ID] = s
	return nil
}

func (r *Repository) GetShow(ctx context.Context, id uuid.UUID) (booking.Show, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	show, ok := r.shows[id]
	if !ok {
		return booking.Show{}, fmt.Errorf("%w: %s", booking.ErrShowNotFound, id)
	}
	return show, nil
}

func (r *Repository) GetBooking(ctx context.Context, id uuid.UUID) (booking.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, ok := r.bookings[id]
	if !ok {
		return booking.Booking{}, fmt.Errorf("%w: %s", booking.ErrBookingNotFound, id)
	}
	return b, nil
}

func (r *Repository) InsertBooking(ctx context.Context, b booking.Booking) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.bookings[b.ID] = b
	return nil
}

func (r *Repository) ListReservations(ctx context.Context, bookingID uuid.UUID) ([]booking.Reservation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	reservations := make([]booking.Reservation, 0)
	for _, res := range r.reservations {
		if res.BookingID == bookingID {
			reservations = append(reservations, res)
		}
	}
	slices.SortFunc(reservations, func(a, b booking.Reservation) int {
		return bytes.Compare(a.ID[:], b.ID[:])
	})
	return reservations, nil
}

func (r *Repository) InsertReservation(ctx context.Context, res booking.Reservation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reservations[res.ID] = res
	return nil
}
