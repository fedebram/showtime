package store

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/fedebram/showtime/internal/booking"
)

var ErrShowNotFound = errors.New("show not found")

type MemoryStore struct {
	mu           sync.Mutex
	shows        map[uuid.UUID]booking.Show
	rooms        map[uuid.UUID]booking.Room
	seats        map[uuid.UUID]booking.Seat
	reservations map[uuid.UUID]booking.Reservation
	bookings     map[uuid.UUID]booking.Booking
}

func NewMemoryStore() *MemoryStore {
	m := &MemoryStore{
		shows:        make(map[uuid.UUID]booking.Show),
		rooms:        make(map[uuid.UUID]booking.Room),
		seats:        make(map[uuid.UUID]booking.Seat),
		reservations: make(map[uuid.UUID]booking.Reservation),
		bookings:     make(map[uuid.UUID]booking.Booking),
	}

	room1 := m.seedRoom("Room 1")
	room2 := m.seedRoom("Room 2")
	m.seedShow("The Big Lebowski", room1, "2026-10-24T18:00:00+02:00")
	m.seedShow("The Flintstones", room2, "2026-10-24T21:00:00+02:00")

	return m
}

// seedRoom adds a room with rows A and B of 3 seats each and returns its ID.
func (m *MemoryStore) seedRoom(name string) uuid.UUID {
	room := booking.Room{ID: newID(), Name: name}
	m.rooms[room.ID] = room

	for _, row := range []string{"A", "B"} {
		for number := 1; number <= 3; number++ {
			seat := booking.Seat{ID: newID(), RoomID: room.ID, Row: row, Number: number}
			m.seats[seat.ID] = seat
		}
	}
	return room.ID
}

func (m *MemoryStore) seedShow(title string, roomID uuid.UUID, startsAt string) {
	show := booking.Show{ID: newID(), MovieTitle: title, RoomID: roomID, StartsAt: startsAt}
	m.shows[show.ID] = show
}

// newID returns a time ordered uuid (v7).
// it fails only if the random source fails.
func newID() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}

// lessID reports whether a sorts before b.
func lessID(a, b uuid.UUID) bool {
	return bytes.Compare(a[:], b[:]) < 0
}

func (m *MemoryStore) ListShows() []booking.Show {
	m.mu.Lock()
	shows := make([]booking.Show, 0, len(m.shows))
	for _, show := range m.shows {
		shows = append(shows, show)
	}
	m.mu.Unlock()

	sort.Slice(shows, func(i, j int) bool {
		return lessID(shows[i].ID, shows[j].ID)
	})
	return shows
}

func (m *MemoryStore) GetShow(id uuid.UUID) (booking.Show, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	show, ok := m.shows[id]
	return show, ok
}

func (m *MemoryStore) GetBooking(id uuid.UUID) (booking.Booking, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	booking, ok := m.bookings[id]
	return booking, ok
}

// AvailableSeats is mostly a business logic function.
// The store was intended as an abastraction for the implementation of multiple dbs.
// Doesn't make sense now to layering a repository to place the business logic there.

func (m *MemoryStore) AvailableSeats(showID uuid.UUID) ([]booking.Seat, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.availableSeats(showID)
}

// availableSeats requires the caller to hold the store mutex.
func (m *MemoryStore) availableSeats(showID uuid.UUID) ([]booking.Seat, error) {
	show, ok := m.shows[showID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrShowNotFound, showID)
	}

	now := time.Now()
	// seatID -> reserved or not (bool)
	reserved := make(map[uuid.UUID]bool)
	for _, reservation := range m.reservations {
		if reservation.ShowID != showID {
			continue
		}
		// if the booking is expired or cancelled
		// we don't count the seats as reserved!
		if !m.bookings[reservation.BookingID].HoldsSeats(now) {
			continue
		}
		for _, seatID := range reservation.SeatIDs {
			reserved[seatID] = true
		}
	}

	// from all the seats on the room, we retain the non reserved ones.
	// this way we have the available seats for the show.

	available := make([]booking.Seat, 0)
	for _, seat := range m.seats {
		if seat.RoomID == show.RoomID && !reserved[seat.ID] {
			available = append(available, seat)
		}
	}

	// deterministic ordering, by row and then by number
	sort.Slice(available, func(i, j int) bool {
		if available[i].Row != available[j].Row {
			return available[i].Row < available[j].Row
		}
		return available[i].Number < available[j].Number
	})

	return available, nil
}

func (m *MemoryStore) CreateBooking(tokenHash string, reqs []booking.ReservationRequest) (booking.Booking, []booking.Reservation, error) {
	if err := booking.ValidateReservations(reqs); err != nil {
		return booking.Booking{}, nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	reservations, err := m.assignReservations(reqs)
	if err != nil {
		return booking.Booking{}, nil, err
	}

	// save to the store
	now := time.Now()
	b := booking.Booking{
		ID:        newID(),
		Status:    booking.BookingPending,
		CreatedAt: now,
		ExpiresAt: now.Add(booking.HoldDuration),
		TokenHash: tokenHash,
	}
	m.bookings[b.ID] = b

	m.saveReservations(b.ID, reservations)

	return b, reservations, nil
}

func (m *MemoryStore) assignReservations(reqs []booking.ReservationRequest) ([]booking.Reservation, error) {
	reservations := make([]booking.Reservation, 0, len(reqs))
	for _, req := range reqs {
		available, err := m.availableSeats(req.ShowID)
		if err != nil {
			return nil, err
		}
		seatIDs, err := booking.AssignSeats(available, req)
		if err != nil {
			return nil, err
		}
		reservations = append(reservations, booking.Reservation{ShowID: req.ShowID, SeatIDs: seatIDs})
	}
	return reservations, nil
}

func (m *MemoryStore) saveReservations(bookingID uuid.UUID, reservations []booking.Reservation) {
	for i := range reservations {
		reservations[i].ID = newID()
		reservations[i].BookingID = bookingID
		m.reservations[reservations[i].ID] = reservations[i]
	}
}

func (m *MemoryStore) ListReservations(bookingID uuid.UUID) []booking.Reservation {
	m.mu.Lock()
	defer m.mu.Unlock()

	// one to many, foreign key like
	reservations := make([]booking.Reservation, 0)
	for _, reservation := range m.reservations {
		if reservation.BookingID == bookingID {
			reservations = append(reservations, reservation)
		}
	}

	// v7 uuids are time ordered, so this is creation order
	sort.Slice(reservations, func(i, j int) bool {
		return lessID(reservations[i].ID, reservations[j].ID)
	})
	return reservations
}

// RemoveReservation deletes a reservation from a pending booking.
func (m *MemoryStore) RemoveReservation(bookingID, reservationID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	b, ok := m.bookings[bookingID]
	if !ok {
		return fmt.Errorf("booking %s not found", bookingID)
	}
	if b.CurrentStatus(time.Now()) != booking.BookingPending {
		return booking.ErrBookingNotPending
	}

	// it must belong to this booking, otherwise you could delete someone else's
	reservation, ok := m.reservations[reservationID]
	if !ok || reservation.BookingID != bookingID {
		return booking.ErrReservationNotFound
	}

	delete(m.reservations, reservationID)
	return nil
}

// CancelBooking moves a pending or confirmed booking to cancelled.
func (m *MemoryStore) CancelBooking(id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	b, ok := m.bookings[id]
	if !ok {
		return fmt.Errorf("booking %s not found", id)
	}
	switch b.CurrentStatus(time.Now()) {
	case booking.BookingPending, booking.BookingConfirmed:
	default:
		return booking.ErrBookingNotActive
	}

	b.Status = booking.BookingCancelled
	m.bookings[id] = b
	return nil
}

// ConfirmBooking moves a pending booking to confirmed. seats stay held.
func (m *MemoryStore) ConfirmBooking(id uuid.UUID) (booking.Booking, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	b, ok := m.bookings[id]
	if !ok {
		return booking.Booking{}, fmt.Errorf("booking %s not found", id)
	}
	if b.CurrentStatus(time.Now()) != booking.BookingPending {
		return booking.Booking{}, booking.ErrBookingNotPending
	}

	// an empty booking can't be confirmed
	hasReservations := false
	for _, reservation := range m.reservations {
		if reservation.BookingID == id {
			hasReservations = true
			break
		}
	}
	if !hasReservations {
		return booking.Booking{}, booking.ErrBookingEmpty
	}

	b.Status = booking.BookingConfirmed
	m.bookings[id] = b
	return b, nil
}
