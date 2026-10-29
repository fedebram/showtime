package booking

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Relay publishes the unsent outbox messages, oldest first.
type Relay struct {
	repo      Repository
	publisher Publisher
	now       func() time.Time
	interval  time.Duration
}

func NewRelay(repo Repository, publisher Publisher, now func() time.Time) *Relay {
	return &Relay{repo: repo, publisher: publisher, now: now, interval: time.Second}
}

// Run publishes the outbox every interval until ctx is cancelled.
func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.Deliver(ctx); err != nil {
				log.Printf("relay: %v", err)
			}
		}
	}
}

// Deliver publishes the unsent messages once. It stops at the first failure:
// the rest stays in the outbox for the next try.
func (r *Relay) Deliver(ctx context.Context) error {
	var msgs []OutboxMessage
	err := r.repo.View(ctx, func(tx Tx) error {
		var err error
		msgs, err = tx.ListUnsentOutbox(ctx, 100)
		return err
	})
	if err != nil {
		return err
	}

	for _, m := range msgs {
		if err := r.publisher.Publish(ctx, m); err != nil {
			return fmt.Errorf("notify %s: %w", m.ID, err)
		}
		// a crash before this line publishes m again.
		// the receiver must ignore the repeat
		err := r.repo.Update(ctx, func(tx Tx) error {
			return tx.MarkOutboxSent(ctx, m.ID, r.now())
		})
		if err != nil {
			return err
		}
	}
	return nil
}
