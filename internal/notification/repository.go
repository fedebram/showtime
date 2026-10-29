package notification

import (
	"context"

	"github.com/google/uuid"
)

// Repository stores the sent messages.
type Repository interface {
	// Seen reports whether a message with this ID is already stored.
	Seen(ctx context.Context, id uuid.UUID) (bool, error)
	// Insert saves m. If a message with the same ID is already stored
	// it does nothing.
	Insert(ctx context.Context, m Message) error
	// List returns the newest messages first, at most limit.
	List(ctx context.Context, limit int) ([]Message, error)
}
