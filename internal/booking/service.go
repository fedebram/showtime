package booking

import (
	"context"
	"fmt"
	"time"

	"github.com/fedebram/showtime/internal/token"
	"github.com/google/uuid"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

// NewService returns a Service. now is the clock dependency injected.
func NewService(repo Repository, now func() time.Time) *Service {
	return &Service{repo: repo, now: now}
}

func (s *Service) ListShows(ctx context.Context) ([]Show, error) {
	var shows []Show
	err := s.repo.View(ctx, func(tx Tx) error {
		var err error
		shows, err = tx.ListShows(ctx)
		return err
	})
	return shows, err
}

func (s *Service) GetShow(ctx context.Context, id uuid.UUID) (Show, error) {
	var show Show
	err := s.repo.View(ctx, func(tx Tx) error {
		var err error
		show, err = tx.GetShow(ctx, id)
		return err
	})
	return show, err
}

type SeatStatus string

const (
	SeatAvailable SeatStatus = "available"
	SeatHeld      SeatStatus = "held"
)

// SeatAvailability is a seat of a show and whether it can be booked.
// It is computed on every SeatMap call, never stored.
type SeatAvailability struct {
	Seat
	Status SeatStatus `json:"status"`
}

// SeatMap returns every seat of the show's room, each available or held.
func (s *Service) SeatMap(ctx context.Context, showID uuid.UUID) ([]SeatAvailability, error) {
	var seatMap []SeatAvailability
	err := s.repo.View(ctx, func(tx Tx) error {
		show, err := tx.GetShow(ctx, showID)
		if err != nil {
			return err
		}
		seats, err := tx.ListSeats(ctx, show.RoomID)
		if err != nil {
			return err
		}
		taken, err := s.takenSeats(ctx, tx, showID)
		if err != nil {
			return err
		}

		seatMap = make([]SeatAvailability, 0, len(seats))
		for _, seat := range seats {
			status := SeatAvailable
			if taken[seat.ID] {
				status = SeatHeld
			}
			seatMap = append(seatMap, SeatAvailability{Seat: seat, Status: status})
		}
		return nil
	})
	return seatMap, err
}

// takenSeats returns the seats of a show held by active bookings:
// confirmed, or pending and not expired.
// it takes tx so it runs inside the caller's transaction.
func (s *Service) takenSeats(ctx context.Context, tx Tx, showID uuid.UUID) (map[uuid.UUID]bool, error) {
	// the only downside of this is that list show reservations
	// returns all the reservations! even the ones expired. For a show it is not a problem,
	// but if there will be many reservations we can make a GC or filter inside the repo...
	reservations, err := tx.ListShowReservations(ctx, showID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	taken := make(map[uuid.UUID]bool)
	for _, r := range reservations {
		// expired bookings and cancelled bookings don't hold their seats.
		if !r.Booking.HoldsSeats(now) {
			continue
		}
		for _, id := range r.SeatIDs {
			taken[id] = true
		}
	}
	return taken, nil
}

// Authenticate returns the booking if secret is its token.
// A missing booking and a wrong token both give ErrBookingNotFound.
func (s *Service) Authenticate(ctx context.Context, id uuid.UUID, secret string) (Booking, error) {
	var b Booking
	err := s.repo.View(ctx, func(tx Tx) error {
		var err error
		b, err = tx.GetBooking(ctx, id)
		return err
	})
	if err != nil {
		return Booking{}, err
	}
	if !token.Matches(secret, b.TokenHash) {
		return Booking{}, fmt.Errorf("%w: %s", ErrBookingNotFound, id)
	}
	return b, nil
}

// GetBooking returns a booking and its reservations.
// A pending booking past its hold is returned as expired.
func (s *Service) GetBooking(ctx context.Context, id uuid.UUID) (Booking, []Reservation, error) {
	var b Booking
	var reservations []Reservation
	// read transaction for get booking and list reservations
	err := s.repo.View(ctx, func(tx Tx) error {
		var err error
		if b, err = tx.GetBooking(ctx, id); err != nil {
			return err
		}
		reservations, err = tx.ListReservations(ctx, id)
		return err
	})
	if err != nil {
		return Booking{}, nil, err
	}
	b.Status = b.CurrentStatus(s.now())
	return b, reservations, nil
}

// CancelBooking cancels a pending or confirmed booking and returns it.
func (s *Service) CancelBooking(ctx context.Context, id uuid.UUID) (Booking, error) {
	var b Booking
	err := s.repo.Update(ctx, func(tx Tx) error {
		var err error
		if b, err = tx.GetBooking(ctx, id); err != nil {
			return err
		}
		if err := b.Cancel(s.now()); err != nil {
			return err
		}
		return tx.UpdateBooking(ctx, b)
	})
	if err != nil {
		return Booking{}, err
	}
	return b, nil
}

// ConfirmBooking confirms a pending booking that has reservations and returns it.
func (s *Service) ConfirmBooking(ctx context.Context, id uuid.UUID) (Booking, error) {
	var b Booking
	err := s.repo.Update(ctx, func(tx Tx) error {
		var err error
		if b, err = tx.GetBooking(ctx, id); err != nil {
			return err
		}
		reservations, err := tx.ListReservations(ctx, id)
		if err != nil {
			return err
		}
		if err := b.Confirm(s.now(), len(reservations)); err != nil {
			return err
		}
		return tx.UpdateBooking(ctx, b)
	})
	if err != nil {
		return Booking{}, err
	}
	return b, nil
}

// RemoveReservation removes a reservation from a pending booking.
// Its seats are free again immediately.
func (s *Service) RemoveReservation(ctx context.Context, bookingID, reservationID uuid.UUID) error {
	return s.repo.Update(ctx, func(tx Tx) error {
		b, err := tx.GetBooking(ctx, bookingID)
		if err != nil {
			return err
		}
		if b.CurrentStatus(s.now()) != BookingPending {
			return ErrBookingNotPending
		}
		return tx.DeleteReservation(ctx, bookingID, reservationID)
	})
}

// ReservationRequest asks for seats of one show: either specific seat_ids,
// or a quantity of seats the system picks.
type ReservationRequest struct {
	ShowID   uuid.UUID   `json:"show_id"`
	SeatIDs  []uuid.UUID `json:"seat_ids"`
	Quantity int         `json:"quantity"`
}

// CreateBooking opens a pending booking holding the requested seats, all or
// nothing. It returns the booking, its reservations and the secret token,
// which is given out only here.
func (s *Service) CreateBooking(ctx context.Context, reqs []ReservationRequest) (Booking, []Reservation, string, error) {
	if err := ValidateReservations(reqs); err != nil {
		return Booking{}, nil, "", err
	}

	secret, hash := token.GenerateToken()
	now := s.now()
	b := Booking{
		ID:        uuid.Must(uuid.NewV7()),
		Status:    BookingPending,
		CreatedAt: now,
		ExpiresAt: now.Add(HoldDuration),
		TokenHash: hash,
	}

	var reservations []Reservation
	err := s.repo.Update(ctx, func(tx Tx) error {
		if err := tx.InsertBooking(ctx, b); err != nil {
			return err
		}
		for _, req := range reqs {
			res, err := s.reserve(ctx, tx, b.ID, req)
			if err != nil {
				return err
			}
			reservations = append(reservations, res)
		}
		return nil
	})
	if err != nil {
		return Booking{}, nil, "", err
	}
	return b, reservations, secret, nil
}

// AddReservation adds a show to a pending booking and returns the new
// reservation. A booking has one reservation per show: to change the seats
// of a show, remove its reservation and add it again.
func (s *Service) AddReservation(ctx context.Context, bookingID uuid.UUID, req ReservationRequest) (Reservation, error) {
	if err := req.Validate(); err != nil {
		return Reservation{}, err
	}

	var res Reservation
	err := s.repo.Update(ctx, func(tx Tx) error {
		b, err := tx.GetBooking(ctx, bookingID)
		if err != nil {
			return err
		}
		if b.CurrentStatus(s.now()) != BookingPending {
			return ErrBookingNotPending
		}

		existing, err := tx.ListReservations(ctx, bookingID)
		if err != nil {
			return err
		}
		for _, e := range existing {
			if e.ShowID == req.ShowID {
				return fmt.Errorf("%w: %s", ErrShowAlreadyBooked, req.ShowID)
			}
		}

		res, err = s.reserve(ctx, tx, bookingID, req)
		return err
	})
	if err != nil {
		return Reservation{}, err
	}
	return res, nil
}

// reserve picks the seats asked by req among the free ones and saves the
// reservation. It takes tx so it runs inside the caller's transaction.
func (s *Service) reserve(ctx context.Context, tx Tx, bookingID uuid.UUID, req ReservationRequest) (Reservation, error) {
	show, err := tx.GetShow(ctx, req.ShowID)
	if err != nil {
		return Reservation{}, err
	}
	seats, err := tx.ListSeats(ctx, show.RoomID)
	if err != nil {
		return Reservation{}, err
	}
	taken, err := s.takenSeats(ctx, tx, show.ID)
	if err != nil {
		return Reservation{}, err
	}

	free := make([]Seat, 0, len(seats))
	for _, seat := range seats {
		if !taken[seat.ID] {
			free = append(free, seat)
		}
	}
	seatIDs, err := assignSeats(free, req)
	if err != nil {
		return Reservation{}, err
	}

	res := Reservation{
		ID:        uuid.Must(uuid.NewV7()),
		BookingID: bookingID,
		ShowID:    show.ID,
		SeatIDs:   seatIDs,
	}
	return res, tx.InsertReservation(ctx, res)
}

func assignSeats(available []Seat, req ReservationRequest) ([]uuid.UUID, error) {
	// we assign seats from the quantity or from the seatIDs slice.

	if req.Quantity > 0 {
		if len(available) < req.Quantity {
			return nil, ErrNotEnoughSeats
		}
		seatIDs := make([]uuid.UUID, 0, req.Quantity)
		// from index 0 through quantity-1
		for _, seat := range available[:req.Quantity] {
			seatIDs = append(seatIDs, seat.ID)
		}
		return seatIDs, nil
	}

	free := make(map[uuid.UUID]bool, len(available))
	for _, seat := range available {
		free[seat.ID] = true
	}
	seatIDs := make([]uuid.UUID, 0, len(req.SeatIDs))
	for _, id := range req.SeatIDs {
		if !free[id] {
			return nil, fmt.Errorf("%w: %s", ErrSeatUnavailable, id)
		}
		seatIDs = append(seatIDs, id)
	}
	return seatIDs, nil
}
