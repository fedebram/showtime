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

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// update runs fn inside repo.Update and fails the test on error.
func update(t *testing.T, repo *memory.Repository, fn func(tx booking.Tx) error) {
	t.Helper()
	if err := repo.Update(context.Background(), fn); err != nil {
		t.Fatal(err)
	}
}

func TestListShowsInCreationOrder(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepository()

	first := booking.Show{ID: newID(), MovieTitle: "The Big Lebowski"}
	second := booking.Show{ID: newID(), MovieTitle: "The Flintstones"}

	// inserted in reverse.
	// but the result list must be in creation order
	update(t, repo, func(tx booking.Tx) error {
		if err := tx.InsertShow(ctx, second); err != nil {
			return err
		}
		return tx.InsertShow(ctx, first)
	})

	var shows []booking.Show
	err := repo.View(ctx, func(tx booking.Tx) error {
		var err error
		shows, err = tx.ListShows(ctx)
		return err
	})
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

	want := booking.Show{ID: newID(), MovieTitle: "The Big Lebowski"}
	update(t, repo, func(tx booking.Tx) error { return tx.InsertShow(ctx, want) })

	var got booking.Show
	err := repo.View(ctx, func(tx booking.Tx) error {
		var err error
		got, err = tx.GetShow(ctx, want.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestGetShowNotFound(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepository()

	err := repo.View(ctx, func(tx booking.Tx) error {
		_, err := tx.GetShow(ctx, newID())
		return err
	})
	if !errors.Is(err, booking.ErrShowNotFound) {
		t.Fatalf("got %v, want ErrShowNotFound", err)
	}
}

func TestGetBooking(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepository()

	want := booking.Booking{ID: newID(), Status: booking.BookingPending}
	update(t, repo, func(tx booking.Tx) error { return tx.InsertBooking(ctx, want) })

	var got booking.Booking
	err := repo.View(ctx, func(tx booking.Tx) error {
		var err error
		got, err = tx.GetBooking(ctx, want.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestGetBookingNotFound(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepository()

	err := repo.View(ctx, func(tx booking.Tx) error {
		_, err := tx.GetBooking(ctx, newID())
		return err
	})
	if !errors.Is(err, booking.ErrBookingNotFound) {
		t.Fatalf("got %v, want ErrBookingNotFound", err)
	}
}

func TestUpdateBooking(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepository()

	b := booking.Booking{ID: newID(), Status: booking.BookingPending}
	update(t, repo, func(tx booking.Tx) error { return tx.InsertBooking(ctx, b) })

	b.Status = booking.BookingCancelled
	update(t, repo, func(tx booking.Tx) error { return tx.UpdateBooking(ctx, b) })

	var got booking.Booking
	err := repo.View(ctx, func(tx booking.Tx) error {
		var err error
		got, err = tx.GetBooking(ctx, b.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != booking.BookingCancelled {
		t.Fatalf("got %s, want cancelled", got.Status)
	}
}

func TestUpdateBookingNotFound(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepository()

	err := repo.Update(ctx, func(tx booking.Tx) error {
		return tx.UpdateBooking(ctx, booking.Booking{ID: newID()})
	})
	if !errors.Is(err, booking.ErrBookingNotFound) {
		t.Fatalf("got %v, want ErrBookingNotFound", err)
	}
}

func TestListReservations(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepository()

	mine, other := newID(), newID()
	first := booking.Reservation{ID: newID(), BookingID: mine}
	second := booking.Reservation{ID: newID(), BookingID: mine}
	notMine := booking.Reservation{ID: newID(), BookingID: other}

	// inserted out of order, with another booking's reservation in between
	update(t, repo, func(tx booking.Tx) error {
		for _, res := range []booking.Reservation{second, notMine, first} {
			if err := tx.InsertReservation(ctx, res); err != nil {
				return err
			}
		}
		return nil
	})

	var got []booking.Reservation
	err := repo.View(ctx, func(tx booking.Tx) error {
		var err error
		got, err = tx.ListReservations(ctx, mine)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != first.ID || got[1].ID != second.ID {
		t.Fatalf("got %v, want %s then %s", got, first.ID, second.ID)
	}
}

func TestUpdateRollsBack(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepository()

	b := booking.Booking{ID: newID()}
	failure := errors.New("something went wrong")

	err := repo.Update(ctx, func(tx booking.Tx) error {
		if err := tx.InsertBooking(ctx, b); err != nil {
			return err
		}
		// fail after the insert. the insert must be undone.
		// ROLLBACK!!
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("got %v, want the failure", err)
	}

	err = repo.View(ctx, func(tx booking.Tx) error {
		_, err := tx.GetBooking(ctx, b.ID)
		return err
	})
	if !errors.Is(err, booking.ErrBookingNotFound) {
		t.Fatalf("insert was not undone: %v", err)
	}
}
