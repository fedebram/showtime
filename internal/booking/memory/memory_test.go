package memory_test

// _test package is needed to avoid import cycle

import (
	"context"
	"errors"
	"slices"
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

func TestDeleteReservation(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepository()

	mine, other := newID(), newID()
	res := booking.Reservation{ID: newID(), BookingID: mine}
	update(t, repo, func(tx booking.Tx) error { return tx.InsertReservation(ctx, res) })

	// you can't delete a reservation that is from another booking (different uuid)
	t.Run("another booking's", func(t *testing.T) {
		err := repo.Update(ctx, func(tx booking.Tx) error {
			return tx.DeleteReservation(ctx, other, res.ID)
		})
		if !errors.Is(err, booking.ErrReservationNotFound) {
			t.Fatalf("got %v, want ErrReservationNotFound", err)
		}
	})

	t.Run("its own", func(t *testing.T) {
		update(t, repo, func(tx booking.Tx) error { return tx.DeleteReservation(ctx, mine, res.ID) })

		var got []booking.Reservation
		err := repo.View(ctx, func(tx booking.Tx) error {
			var err error
			got, err = tx.ListReservations(ctx, mine)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Fatalf("got %v, want no reservations", got)
		}
	})

	t.Run("already deleted", func(t *testing.T) {
		err := repo.Update(ctx, func(tx booking.Tx) error {
			return tx.DeleteReservation(ctx, mine, res.ID)
		})
		if !errors.Is(err, booking.ErrReservationNotFound) {
			t.Fatalf("got %v, want ErrReservationNotFound", err)
		}
	})
}

func TestListSeatsSortedByRowThenNumber(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepository()

	// four seats of the same room
	room := newID()
	a1 := booking.Seat{ID: newID(), RoomID: room, Row: "A", Number: 1}
	a2 := booking.Seat{ID: newID(), RoomID: room, Row: "A", Number: 2}
	a10 := booking.Seat{ID: newID(), RoomID: room, Row: "A", Number: 10}
	b1 := booking.Seat{ID: newID(), RoomID: room, Row: "B", Number: 1}

	// we save the seats in a shuffled order
	err := repo.Update(ctx, func(tx booking.Tx) error {
		tx.InsertSeat(ctx, b1)
		tx.InsertSeat(ctx, a10)
		tx.InsertSeat(ctx, a1)
		tx.InsertSeat(ctx, a2)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// list the seats of the room
	var got []booking.Seat
	err = repo.View(ctx, func(tx booking.Tx) error {
		got, err = tx.ListSeats(ctx, room)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	// they must come back sorted: A1, A2, A10, B1
	want := []booking.Seat{a1, a2, a10, b1}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestListShowReservations(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepository()

	// two bookings with a reservation for a show, one confirmed and one
	// cancelled, and a reservation for another show
	show, otherShow := newID(), newID()
	confirmed := booking.Booking{ID: newID(), Status: booking.BookingConfirmed}
	cancelled := booking.Booking{ID: newID(), Status: booking.BookingCancelled}
	fromConfirmed := booking.Reservation{ID: newID(), BookingID: confirmed.ID, ShowID: show}
	fromCancelled := booking.Reservation{ID: newID(), BookingID: cancelled.ID, ShowID: show}
	otherShowRes := booking.Reservation{ID: newID(), BookingID: confirmed.ID, ShowID: otherShow}

	err := repo.Update(ctx, func(tx booking.Tx) error {
		tx.InsertBooking(ctx, confirmed)
		tx.InsertBooking(ctx, cancelled)
		tx.InsertReservation(ctx, fromConfirmed)
		tx.InsertReservation(ctx, fromCancelled)
		tx.InsertReservation(ctx, otherShowRes)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// we list the reservations of the first show
	var got []booking.ShowReservation
	err = repo.View(ctx, func(tx booking.Tx) error {
		got, err = tx.ListShowReservations(ctx, show)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	// the reservations inserted must match
	statusOf := make(map[uuid.UUID]booking.BookingStatus)
	for _, r := range got {
		if r.ShowID != show {
			t.Fatalf("got a reservation of show %s, want only %s", r.ShowID, show)
		}
		statusOf[r.ID] = r.Booking.Status
	}
	if len(got) != 2 ||
		statusOf[fromConfirmed.ID] != booking.BookingConfirmed ||
		statusOf[fromCancelled.ID] != booking.BookingCancelled {
		t.Fatalf("got %v, want the confirmed and the cancelled reservation", got)
	}
}
