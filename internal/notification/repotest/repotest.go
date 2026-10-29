// Package repotest is the contract every notification.Repository must satisfy.
// Each repository package must create a test calling repotest.Run
package repotest

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fedebram/showtime/internal/notification"
)

// Run runs every contract test, each one on a fresh, empty repository from newRepo.
func Run(t *testing.T, newRepo func(t *testing.T) notification.Repository) {
	tests := []struct {
		name string
		fn   func(t *testing.T, repo notification.Repository)
	}{
		{"Seen", testSeen},
		{"InsertTwiceKeepsFirst", testInsertTwiceKeepsFirst},
		{"ListNewestFirst", testListNewestFirst},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.fn(t, newRepo(t))
		})
	}
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

var sent = time.Date(2026, 10, 28, 18, 0, 0, 0, time.UTC)

// newMessage returns a message sent at the given time.
func newMessage(at time.Time) notification.Message {
	return notification.Message{ID: newID(), Kind: "booking_confirmed", BookingID: newID(), SentAt: at}
}

// insert saves m and fails the test on error.
func insert(t *testing.T, repo notification.Repository, m notification.Message) {
	t.Helper()
	if err := repo.Insert(context.Background(), m); err != nil {
		t.Fatal(err)
	}
}

func testSeen(t *testing.T, repo notification.Repository) {
	ctx := context.Background()
	m := newMessage(sent)

	seen, err := repo.Seen(ctx, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if seen {
		t.Fatal("seen before insert, want not seen")
	}

	insert(t, repo, m)
	seen, err = repo.Seen(ctx, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !seen {
		t.Fatal("not seen after insert, want seen")
	}
}

func testInsertTwiceKeepsFirst(t *testing.T, repo notification.Repository) {
	m := newMessage(sent)
	insert(t, repo, m)

	// the same ID again, sent later: no error, and the first one stays
	again := m
	again.SentAt = sent.Add(time.Minute)
	insert(t, repo, again)

	got, err := repo.List(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d messages, want 1", len(got))
	}
	if !got[0].SentAt.Equal(m.SentAt) {
		t.Fatalf("sent at %v, want the first %v", got[0].SentAt, m.SentAt)
	}
}

func testListNewestFirst(t *testing.T, repo notification.Repository) {
	first := newMessage(sent)
	second := newMessage(sent.Add(time.Minute))
	third := newMessage(sent.Add(2 * time.Minute))
	for _, m := range []notification.Message{first, second, third} {
		insert(t, repo, m)
	}

	// at most 2: the newest two, newest first
	got, err := repo.List(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != third.ID || got[1].ID != second.ID {
		t.Fatalf("got %+v, want third then second", got)
	}
	// every field comes back
	if got[0].Kind != third.Kind || got[0].BookingID != third.BookingID || !got[0].SentAt.Equal(third.SentAt) {
		t.Fatalf("got %+v, want %+v", got[0], third)
	}
}
