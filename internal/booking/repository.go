package booking

import (
	"context"

	"github.com/google/uuid"
)

// Repository gives access to the stored data.
type Repository interface {
	// View runs fn with read-only access.
	View(ctx context.Context, fn func(tx Tx) error) error
	// Update runs fn as one transaction.
	// If the fn returns an error the update(s) are rolled back.
	Update(ctx context.Context, fn func(tx Tx) error) error
}

// Tx reads and writes data inside View or Update.
// Don't use it after fn returns.
type Tx interface {
	// ListShows returns every show, in creation order.
	ListShows(ctx context.Context) ([]Show, error)
	// InsertShow saves a new show. show ID must already be set.
	InsertShow(ctx context.Context, s Show) error
	// GetShow returns the show with the given id.
	// If there is no such show, the error wraps ErrShowNotFound.
	GetShow(ctx context.Context, id uuid.UUID) (Show, error)

	// GetBooking returns the booking with the given id.
	// If there is no such booking, the error wraps ErrBookingNotFound.
	GetBooking(ctx context.Context, id uuid.UUID) (Booking, error)
	// InsertBooking saves a new booking. booking ID must already be set.
	InsertBooking(ctx context.Context, b Booking) error
	// UpdateBooking replaces a stored booking.
	// If there is no such booking, the error wraps ErrBookingNotFound.
	UpdateBooking(ctx context.Context, b Booking) error

	// ListReservations returns the reservations of a booking, in creation order.
	ListReservations(ctx context.Context, bookingID uuid.UUID) ([]Reservation, error)
	// InsertReservation saves a new reservation. reservation ID must already be set.
	InsertReservation(ctx context.Context, r Reservation) error
}
