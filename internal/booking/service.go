package booking

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListShows(ctx context.Context) ([]Show, error) {
	return s.repo.ListShows(ctx)
}

func (s *Service) GetShow(ctx context.Context, id uuid.UUID) (Show, error) {
	return s.repo.GetShow(ctx, id)
}
