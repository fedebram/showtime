package booking_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fedebram/showtime/internal/booking"
	"github.com/fedebram/showtime/internal/booking/memory"
	"github.com/fedebram/showtime/internal/token"
)

func TestAuthenticate(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepository()

	secret, hash := token.GenerateToken()
	b := booking.Booking{ID: uuid.Must(uuid.NewV7()), Status: booking.BookingPending, TokenHash: hash}
	if err := repo.InsertBooking(ctx, b); err != nil {
		t.Fatal(err)
	}
	svc := booking.NewService(repo, time.Now)

	t.Run("right token", func(t *testing.T) {
		got, err := svc.Authenticate(ctx, b.ID, secret)
		if err != nil {
			t.Fatal(err)
		}
		if got.ID != b.ID {
			t.Fatalf("got booking %s, want %s", got.ID, b.ID)
		}
	})

	t.Run("wrong token", func(t *testing.T) {
		_, err := svc.Authenticate(ctx, b.ID, "wrong")
		if !errors.Is(err, booking.ErrBookingNotFound) {
			t.Fatalf("got %v, want ErrBookingNotFound", err)
		}
	})

	t.Run("missing booking", func(t *testing.T) {
		_, err := svc.Authenticate(ctx, uuid.Must(uuid.NewV7()), secret)
		if !errors.Is(err, booking.ErrBookingNotFound) {
			t.Fatalf("got %v, want ErrBookingNotFound", err)
		}
	})
}
