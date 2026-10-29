package booking

import "context"

// Notifier delivers outbox messages outside the booking service.
// Delivery may be repeated: the receiver must ignore duplicates by ID.
type Notifier interface {
	Notify(ctx context.Context, m OutboxMessage) error
}
