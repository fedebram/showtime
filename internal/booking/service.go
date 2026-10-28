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
