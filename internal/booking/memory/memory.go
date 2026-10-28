package memory

import (
	"bytes"
	"context"
	"slices"
	"sync"

	"github.com/google/uuid"

	"github.com/fedebram/showtime/internal/booking"
)

// compile time check that memory repository satisfies the booking repository interface.
var _ booking.Repository = (*Repository)(nil)

type Repository struct {
	mu    sync.Mutex
	shows map[uuid.UUID]booking.Show
}

func NewRepository() *Repository {
	return &Repository{
		shows: make(map[uuid.UUID]booking.Show),
	}
}

func (r *Repository) ListShows(ctx context.Context) ([]booking.Show, error) {
	// in memory repository implementation doesn't need context

	r.mu.Lock()
	defer r.mu.Unlock()

	shows := make([]booking.Show, 0, len(r.shows))
	for _, show := range r.shows {
		shows = append(shows, show)
	}
	// for show we use v7 uuids that are time ordered, so this is creation order
	slices.SortFunc(shows, func(a, b booking.Show) int {
		return bytes.Compare(a.ID[:], b.ID[:])
	})
	return shows, nil
}

func (r *Repository) InsertShow(ctx context.Context, s booking.Show) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.shows[s.ID] = s
	return nil
}
