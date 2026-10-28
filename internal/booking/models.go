package booking

import (
	"fmt"
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

type BookingStatus string

const (
	BookingPending   BookingStatus = "pending"
	BookingConfirmed BookingStatus = "confirmed"
	BookingExpired   BookingStatus = "expired"
	BookingCancelled BookingStatus = "cancelled"
)

// a reservation request is more like a public api contract.

type ReservationRequest struct {
	ShowID   uuid.UUID   `json:"show_id"`
	SeatIDs  []uuid.UUID `json:"seat_ids"`
	Quantity int         `json:"quantity"`
}

// TODO: return multiple validation fields errors. map?

func (r *ReservationRequest) Validate() error {
	if r.ShowID == uuid.Nil {
		return fmt.Errorf("show_id is required")
	}
	if r.Quantity < 0 {
		return fmt.Errorf("quantity cannot be negative")
	}

	// nil slice or also empty slice length is 0
	hasSeats := len(r.SeatIDs) > 0
	hasQuantity := r.Quantity > 0

	// require exactly one of seat_ids or a positive quantity
	if hasSeats == hasQuantity {
		return fmt.Errorf("specify either seat_ids or a positive quantity")
	}

	seen := make(map[uuid.UUID]bool)
	for _, id := range r.SeatIDs {
		if id == uuid.Nil {
			return fmt.Errorf("seat IDs cannot be nil")
		}
		if seen[id] {
			return fmt.Errorf("duplicate seat ID: %s", id)
		}
		seen[id] = true
	}

	return nil
}

func ValidateReservations(reqs []ReservationRequest) error {
	// we enforce 1 reservation -> 1 show
	// you can't make multiple reservation requests for the same show.

	// an empty list is valid.

	seenShows := make(map[uuid.UUID]bool)
	for _, reservation := range reqs {
		if err := reservation.Validate(); err != nil {
			return err
		}
		if seenShows[reservation.ShowID] {
			return fmt.Errorf("duplicate show_id: %s", reservation.ShowID)
		}
		seenShows[reservation.ShowID] = true
	}
	return nil
}
