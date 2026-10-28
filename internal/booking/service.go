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
	return s.repo.ListShows(ctx)
}

func (s *Service) GetShow(ctx context.Context, id uuid.UUID) (Show, error) {
	return s.repo.GetShow(ctx, id)
}

// Authenticate returns the booking if secret is its token.
// A missing booking and a wrong token both give ErrBookingNotFound.
func (s *Service) Authenticate(ctx context.Context, id uuid.UUID, secret string) (Booking, error) {
	b, err := s.repo.GetBooking(ctx, id)
	if err != nil {
		return Booking{}, err
	}
	if !token.Matches(secret, b.TokenHash) {
		return Booking{}, fmt.Errorf("%w: %s", ErrBookingNotFound, id)
	}
	return b, nil
}
