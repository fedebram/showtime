package booking

import (
	"context"
	"time"

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
	// InsertRoom saves a new room. room ID must already be set.
	InsertRoom(ctx context.Context, r Room) error

	// InsertSeat saves a new seat. seat ID must already be set.
	InsertSeat(ctx context.Context, s Seat) error
	// ListSeats returns the seats of a room, sorted by row and then by number.
	ListSeats(ctx context.Context, roomID uuid.UUID) ([]Seat, error)

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
	// DeleteReservation removes a reservation of a booking. If the reservation
	// doesn't exist or belongs to another booking, the error wraps ErrReservationNotFound.
	DeleteReservation(ctx context.Context, bookingID, reservationID uuid.UUID) error
	// ListShowReservations returns every reservation of a show together with
	// its booking, whatever the booking's status.
	ListShowReservations(ctx context.Context, showID uuid.UUID) ([]ShowReservation, error)

	// InsertOutbox saves a new outbox message. message ID must already be set.
	InsertOutbox(ctx context.Context, m OutboxMessage) error
	// ListUnsentOutbox returns at most limit unsent messages, oldest first.
	ListUnsentOutbox(ctx context.Context, limit int) ([]OutboxMessage, error)
	// MarkOutboxSent records that a message was delivered at the given time.
	// Marking it again keeps the first time.
	// If there is no such message, the error wraps ErrOutboxNotFound.
	MarkOutboxSent(ctx context.Context, id uuid.UUID, at time.Time) error
}

// ShowReservation is a reservation of a show together with its booking.
type ShowReservation struct {
	Reservation
	Booking Booking
}
