package booking

import "context"

// Publisher sends outbox messages out of the booking service.
// The relay may publish a message more than once: the receiver
// must ignore repeats by message ID.
type Publisher interface {
	Publish(ctx context.Context, m OutboxMessage) error
}
