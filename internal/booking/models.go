package booking

import (
	"time"

	"github.com/google/uuid"
)

// right now we place everything inside booking. It is fine like that

// Room represents a screening room in the cinema.
type Room struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// Show represents a movie playing in a room at a scheduled time.
type Show struct {
	ID         uuid.UUID `json:"id"`
	MovieTitle string    `json:"movie_title"`
	RoomID     uuid.UUID `json:"room_id"`
	StartsAt   string    `json:"starts_at"`
}

// Seat represents a physical seat in a room.
type Seat struct {
	ID     uuid.UUID `json:"id"`
	RoomID uuid.UUID `json:"room_id"`
	Row    string    `json:"row"`
	Number int       `json:"number"`
}

// Reservation records seats booked for a show.
type Reservation struct {
	ID        uuid.UUID   `json:"id"`
	BookingID uuid.UUID   `json:"booking_id"`
	ShowID    uuid.UUID   `json:"show_id"`
	SeatIDs   []uuid.UUID `json:"seat_ids"`
}

// Booking groups reservations and tracks their status.
type Booking struct {
	ID        uuid.UUID     `json:"id"`
	Status    BookingStatus `json:"status"` // lazy: use CurrentStatus to read it
	CreatedAt time.Time     `json:"created_at"`
	ExpiresAt time.Time     `json:"expires_at"`
	TokenHash string        `json:"-"`
}

func (b Booking) HoldsSeats(now time.Time) bool {
	return b.Status == BookingConfirmed ||
		(b.Status == BookingPending && now.Before(b.ExpiresAt))
}

// CurrentStatus returns expired if the booking is pending and its time is over.
// otherwise the stored status.
func (b Booking) CurrentStatus(now time.Time) BookingStatus {
	if b.Status == BookingPending && !now.Before(b.ExpiresAt) {
		return BookingExpired
	}
	return b.Status
}

// Cancel moves a pending or confirmed booking to cancelled.
// Expired and already cancelled bookings return ErrBookingNotActive.
func (b *Booking) Cancel(now time.Time) error {
	switch b.CurrentStatus(now) {
	case BookingPending, BookingConfirmed:
		b.Status = BookingCancelled
		return nil
	default:
		return ErrBookingNotActive
	}
}

// Confirm moves a pending booking with at least one reservation to confirmed.
func (b *Booking) Confirm(now time.Time, reservations int) error {
	if b.CurrentStatus(now) != BookingPending {
		return ErrBookingNotPending
	}
	if reservations == 0 {
		return ErrBookingEmpty
	}
	b.Status = BookingConfirmed
	return nil
}

type BookingStatus string

const (
	BookingPending   BookingStatus = "pending"
	BookingConfirmed BookingStatus = "confirmed"
	BookingExpired   BookingStatus = "expired"
	BookingCancelled BookingStatus = "cancelled"
)
