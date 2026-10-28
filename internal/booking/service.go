package booking

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListShows(ctx context.Context) ([]Show, error) {
	// no business logic

	return s.repo.ListShows(ctx)
}
