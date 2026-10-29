package notification_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fedebram/showtime/internal/notification"
	"github.com/fedebram/showtime/internal/notification/memory"
)

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func TestSendTwiceStoresOnce(t *testing.T) {
	ctx := context.Background()
	svc := notification.NewService(memory.NewRepository(), time.Now)
	m := notification.Message{ID: newID(), Kind: "booking_confirmed", BookingID: newID()}

	// the relay retried: same message twice
	for range 2 {
		if err := svc.Send(ctx, m); err != nil {
			t.Fatal(err)
		}
	}

	got, err := svc.List(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != m.ID {
		t.Fatalf("got %+v, want only %s", got, m.ID)
	}
}
