package memory_test

// _test package is needed to avoid import cycle

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/fedebram/showtime/internal/booking"
	"github.com/fedebram/showtime/internal/booking/memory"
)

func TestListShowsInCreationOrder(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepository()

	first := booking.Show{ID: uuid.Must(uuid.NewV7()), MovieTitle: "The Big Lebowski"}
	second := booking.Show{ID: uuid.Must(uuid.NewV7()), MovieTitle: "The Flintstones"}

	// inserted in reverse.
	// but the result list but be in creation order
	for _, show := range []booking.Show{second, first} {
		if err := repo.InsertShow(ctx, show); err != nil {
			t.Fatal(err)
		}
	}

	shows, err := repo.ListShows(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(shows) != 2 || shows[0].ID != first.ID || shows[1].ID != second.ID {
		t.Fatalf("got %v, want %s then %s", shows, first.ID, second.ID)
	}
}

func TestGetShow(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepository()

	want := booking.Show{ID: uuid.Must(uuid.NewV7()), MovieTitle: "The Big Lebowski"}
	if err := repo.InsertShow(ctx, want); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetShow(ctx, want.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestGetShowNotFound(t *testing.T) {
	repo := memory.NewRepository()

	_, err := repo.GetShow(context.Background(), uuid.Must(uuid.NewV7()))
	if !errors.Is(err, booking.ErrShowNotFound) {
		t.Fatalf("got %v, want ErrShowNotFound", err)
	}
}

func TestGetBooking(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepository()

	want := booking.Booking{ID: uuid.Must(uuid.NewV7()), Status: booking.BookingPending}
	if err := repo.InsertBooking(ctx, want); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetBooking(ctx, want.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestGetBookingNotFound(t *testing.T) {
	repo := memory.NewRepository()

	_, err := repo.GetBooking(context.Background(), uuid.Must(uuid.NewV7()))
	if !errors.Is(err, booking.ErrBookingNotFound) {
		t.Fatalf("got %v, want ErrBookingNotFound", err)
	}
}

func TestListReservations(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepository()

	mine, other := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	first := booking.Reservation{ID: uuid.Must(uuid.NewV7()), BookingID: mine}
	second := booking.Reservation{ID: uuid.Must(uuid.NewV7()), BookingID: mine}
	notMine := booking.Reservation{ID: uuid.Must(uuid.NewV7()), BookingID: other}

	// inserted out of order, with another booking's reservation in between
	for _, res := range []booking.Reservation{second, notMine, first} {
		if err := repo.InsertReservation(ctx, res); err != nil {
			t.Fatal(err)
		}
	}

	got, err := repo.ListReservations(ctx, mine)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != first.ID || got[1].ID != second.ID {
		t.Fatalf("got %v, want %s then %s", got, first.ID, second.ID)
	}
}
