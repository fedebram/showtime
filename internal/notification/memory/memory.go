package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/fedebram/showtime/internal/notification"
)

// compile time check that Repository satisfies the notification interface.
var _ notification.Repository = (*Repository)(nil)

// Repository keeps the messages in memory.
type Repository struct {
	mu   sync.Mutex
	msgs []notification.Message
	seen map[uuid.UUID]bool
}

func NewRepository() *Repository {
	return &Repository{seen: make(map[uuid.UUID]bool)}
}

func (r *Repository) Seen(ctx context.Context, id uuid.UUID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.seen[id], nil
}

func (r *Repository) Insert(ctx context.Context, m notification.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.seen[m.ID] {
		return nil
	}
	r.seen[m.ID] = true
	r.msgs = append(r.msgs, m)
	return nil
}

func (r *Repository) List(ctx context.Context, limit int) ([]notification.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := min(limit, len(r.msgs))
	out := make([]notification.Message, 0, n)
	// msgs is oldest first: walk it backwards
	for i := len(r.msgs) - 1; i >= len(r.msgs)-n; i-- {
		out = append(out, r.msgs[i])
	}
	return out, nil
}
