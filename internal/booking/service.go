package booking

import (
	"context"
	"time"

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
