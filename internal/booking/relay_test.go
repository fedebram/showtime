package booking_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fedebram/showtime/internal/booking"
	"github.com/fedebram/showtime/internal/booking/memory"
	"github.com/google/uuid"
)

// fakePublisher records what it publishes. Messages in fail are refused,
// like a receiver that is down.
type fakePublisher struct {
	fail map[uuid.UUID]bool      // IDs to refuse
	got  []booking.OutboxMessage // what was published, in order
}

func (f *fakePublisher) Publish(ctx context.Context, m booking.OutboxMessage) error {
	if f.fail[m.ID] {
		return errors.New("receiver down")
	}
	f.got = append(f.got, m)
	return nil
}

// newOutbox returns a repository holding three confirmed bookings, each
// with its booking_confirmed message waiting in the outbox, oldest first.
func newOutbox(t *testing.T) (booking.Repository, []booking.OutboxMessage) {
	t.Helper()
	ctx := context.Background()
	repo := memory.NewRepository()

	var msgs []booking.OutboxMessage
	for range 3 {
		b := pendingBooking()
		b.Status = booking.BookingConfirmed
		m := booking.OutboxMessage{ID: newID(), Kind: booking.OutboxBookingConfirmed, BookingID: b.ID}
		err := repo.Update(ctx, func(tx booking.Tx) error {
			return errors.Join(
				tx.InsertBooking(ctx, b),
				tx.InsertOutbox(ctx, m),
			)
		})
		if err != nil {
			t.Fatal(err)
		}
		msgs = append(msgs, m)
	}
	return repo, msgs
}

func TestRelayDelivers(t *testing.T) {
	ctx := context.Background()
	repo, msgs := newOutbox(t)
	publisher := &fakePublisher{}
	relay := booking.NewRelay(repo, publisher, clockAt(created))

	if err := relay.Deliver(ctx); err != nil {
		t.Fatal(err)
	}

	// every message arrived, oldest first
	if len(publisher.got) != 3 {
		t.Fatalf("delivered %d messages, want 3", len(publisher.got))
	}
	for i := range msgs {
		if publisher.got[i].ID != msgs[i].ID {
			t.Fatalf("message %d: got %s, want %s", i, publisher.got[i].ID, msgs[i].ID)
		}
	}
	// and nothing is left to send
	if left := unsentOutbox(t, repo); len(left) != 0 {
		t.Fatalf("outbox: %d unsent, want 0", len(left))
	}

	// a second run sends nothing again
	if err := relay.Deliver(ctx); err != nil {
		t.Fatal(err)
	}
	if len(publisher.got) != 3 {
		t.Fatalf("after a second run: delivered %d messages, want still 3", len(publisher.got))
	}
}

func TestRelayRetriesAfterFailure(t *testing.T) {
	ctx := context.Background()
	repo, msgs := newOutbox(t)
	// the receiver refuses the second message
	publisher := &fakePublisher{fail: map[uuid.UUID]bool{msgs[1].ID: true}}
	relay := booking.NewRelay(repo, publisher, clockAt(created))

	// the relay publishes the first, fails on the second and stops there
	if err := relay.Deliver(ctx); err == nil {
		t.Fatal("got no error, want the publish failure")
	}
	if len(publisher.got) != 1 {
		t.Fatalf("delivered %d messages, want 1", len(publisher.got))
	}
	if left := unsentOutbox(t, repo); len(left) != 2 {
		t.Fatalf("outbox: %d unsent, want 2", len(left))
	}

	// the receiver is back: the next run publishes the other two
	publisher.fail = nil
	if err := relay.Deliver(ctx); err != nil {
		t.Fatal(err)
	}
	if len(publisher.got) != 3 {
		t.Fatalf("delivered %d messages, want 3", len(publisher.got))
	}
	if publisher.got[1].ID != msgs[1].ID || publisher.got[2].ID != msgs[2].ID {
		t.Fatal("the retried messages arrived out of order")
	}
	if left := unsentOutbox(t, repo); len(left) != 0 {
		t.Fatalf("outbox: %d unsent, want 0", len(left))
	}
}
