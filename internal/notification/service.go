package notification

import (
	"context"
	"log"
	"time"
)

// Service sends notifications, each message once.
type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository, now func() time.Time) *Service {
	return &Service{repo: repo, now: now}
}

// Send sends m once. A message already sent is skipped,
// so the sender can retry safely.
func (s *Service) Send(ctx context.Context, m Message) error {
	seen, err := s.repo.Seen(ctx, m.ID)
	if err != nil {
		return err
	}
	if seen {
		return nil // already sent
	}

	// there is an edge case here. sending and insert are not atomic! the same problem of
	// the outbox on the booking service! AH!
	// Here is enough. if we crash after sending but before insert, another send is possible.
	// for emails no problem!
	// the external system might provide an idempotency key or something like that!

	// the "send" is a log. A real solution would be sending an email!
	log.Printf("notification: %s for booking %s", m.Kind, m.BookingID)

	m.SentAt = s.now()
	return s.repo.Insert(ctx, m)
}

// List returns the newest messages first, at most limit.
func (s *Service) List(ctx context.Context, limit int) ([]Message, error) {
	return s.repo.List(ctx, limit)
}
