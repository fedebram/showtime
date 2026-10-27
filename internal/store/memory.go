package store

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/fedebram/showtime/internal/booking"
)

var ErrShowNotFound = errors.New("show not found")

type MemoryStore struct {
	mu           sync.Mutex
	shows        map[int]booking.Show
	rooms        map[int]booking.Room
	seats        map[int]booking.Seat
	reservations map[int]booking.Reservation
	bookings     map[int]booking.Booking

	// last used IDs, incremented on every insert
	// TODO: move to uuid
	lastBookingID     int
	lastReservationID int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		shows: map[int]booking.Show{
			1: {ID: 1, MovieTitle: "The Big Lebowski", RoomID: 1, StartsAt: "2026-10-24T18:00:00+02:00"},
			2: {ID: 2, MovieTitle: "The Flintstones", RoomID: 2, StartsAt: "2026-10-24T21:00:00+02:00"},
		},
		rooms: map[int]booking.Room{
			1: {ID: 1, Name: "Room 1"},
			2: {ID: 2, Name: "Room 2"},
		},
		seats: map[int]booking.Seat{
			// Room 1 seats.
			1: {ID: 1, RoomID: 1, Row: "A", Number: 1},
			2: {ID: 2, RoomID: 1, Row: "A", Number: 2},
			3: {ID: 3, RoomID: 1, Row: "A", Number: 3},
			4: {ID: 4, RoomID: 1, Row: "B", Number: 1},
			5: {ID: 5, RoomID: 1, Row: "B", Number: 2},
			6: {ID: 6, RoomID: 1, Row: "B", Number: 3},

			// Room 2 seats.
			7:  {ID: 7, RoomID: 2, Row: "A", Number: 1},
			8:  {ID: 8, RoomID: 2, Row: "A", Number: 2},
			9:  {ID: 9, RoomID: 2, Row: "A", Number: 3},
			10: {ID: 10, RoomID: 2, Row: "B", Number: 1},
			11: {ID: 11, RoomID: 2, Row: "B", Number: 2},
			12: {ID: 12, RoomID: 2, Row: "B", Number: 3},
		},
		reservations: make(map[int]booking.Reservation),
		bookings:     make(map[int]booking.Booking),
	}
}

func (m *MemoryStore) ListShows() []booking.Show {
	m.mu.Lock()
	shows := make([]booking.Show, 0, len(m.shows))
	for _, show := range m.shows {
		shows = append(shows, show)
	}
	m.mu.Unlock()

	sort.Slice(shows, func(i, j int) bool {
		return shows[i].ID < shows[j].ID
	})
	return shows
}

func (m *MemoryStore) GetShow(id int) (booking.Show, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	show, ok := m.shows[id]
	return show, ok
}

func (m *MemoryStore) GetBooking(id int) (booking.Booking, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	booking, ok := m.bookings[id]
	return booking, ok
}

// AvailableSeats is mostly a business logic function.
// The store was intended as an abastraction for the implementation of multiple dbs.
// Doesn't make sense now to layering a repository to place the business logic there.

func (m *MemoryStore) AvailableSeats(showID int) ([]booking.Seat, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.availableSeats(showID)
}

// availableSeats requires the caller to hold the store mutex.
func (m *MemoryStore) availableSeats(showID int) ([]booking.Seat, error) {
	show, ok := m.shows[showID]
	if !ok {
		return nil, fmt.Errorf("%w: %d", ErrShowNotFound, showID)
	}

	now := time.Now()
	// seatID -> reserved or not (bool)
	reserved := make(map[int]bool)
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

	// determinist ordering
	sort.Slice(available, func(i, j int) bool {
		return available[i].ID < available[j].ID
	})

	return available, nil
}

func (m *MemoryStore) CreateBooking(tokenHash string, reqs []booking.ReservationRequest) (booking.Booking, []booking.Reservation, error) {
	if err := booking.ValidateReservations(reqs); err != nil {
		return booking.Booking{}, nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	reservations := make([]booking.Reservation, 0, len(reqs))
	for _, req := range reqs {
		available, err := m.availableSeats(req.ShowID)
		if err != nil {
			return booking.Booking{}, nil, err
		}

		seatIDs, err := booking.AssignSeats(available, req)
		if err != nil {
			return booking.Booking{}, nil, err
		}
		reservations = append(reservations, booking.Reservation{ShowID: req.ShowID, SeatIDs: seatIDs})
	}

	// save to the store
	now := time.Now()
	m.lastBookingID++
	b := booking.Booking{
		ID:        m.lastBookingID,
		Status:    booking.BookingPending,
		CreatedAt: now,
		ExpiresAt: now.Add(booking.HoldDuration),
		TokenHash: tokenHash,
	}
	m.bookings[b.ID] = b

	for i := range reservations {
		m.lastReservationID++
		reservations[i].ID = m.lastReservationID
		reservations[i].BookingID = b.ID
		m.reservations[reservations[i].ID] = reservations[i]
	}

	return b, reservations, nil
}
