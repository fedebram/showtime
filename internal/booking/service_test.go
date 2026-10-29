package booking_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fedebram/showtime/internal/booking"
	"github.com/fedebram/showtime/internal/booking/memory"
	"github.com/fedebram/showtime/internal/token"
)

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

var created = time.Date(2026, 10, 28, 18, 0, 0, 0, time.UTC)

// pendingBooking returns a booking created at created, holding its seats
// for HoldDuration.
func pendingBooking() booking.Booking {
	return booking.Booking{
		ID:        newID(),
		Status:    booking.BookingPending,
		CreatedAt: created,
		ExpiresAt: created.Add(booking.HoldDuration),
	}
}

// newRepo returns a memory repository already holding b and its reservations.
func newRepo(t *testing.T, b booking.Booking, reservations ...booking.Reservation) *memory.Repository {
	t.Helper()
	ctx := context.Background()
	repo := memory.NewRepository()
	err := repo.Update(ctx, func(tx booking.Tx) error {
		if err := tx.InsertBooking(ctx, b); err != nil {
			return err
		}
		for _, res := range reservations {
			if err := tx.InsertReservation(ctx, res); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

// clockAt returns a clock stopped at the given time.
func clockAt(at time.Time) func() time.Time {
	return func() time.Time { return at }
}

func TestAuthenticate(t *testing.T) {
	ctx := context.Background()

	secret, hash := token.GenerateToken()
	b := pendingBooking()
	b.TokenHash = hash
	svc := booking.NewService(newRepo(t, b), time.Now)

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
		_, err := svc.Authenticate(ctx, newID(), secret)
		if !errors.Is(err, booking.ErrBookingNotFound) {
			t.Fatalf("got %v, want ErrBookingNotFound", err)
		}
	})
}

func TestGetBooking(t *testing.T) {
	ctx := context.Background()

	b := pendingBooking()
	res := booking.Reservation{ID: newID(), BookingID: b.ID}
	repo := newRepo(t, b, res)

	// here comes in handy the clock function!!
	cases := []struct {
		name string
		now  time.Time
		want booking.BookingStatus
	}{
		{"during the hold", b.ExpiresAt.Add(-time.Second), booking.BookingPending},
		{"hold just over", b.ExpiresAt, booking.BookingExpired},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reservations, err := booking.NewService(repo, clockAt(tc.now)).GetBooking(ctx, b.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != tc.want {
				t.Fatalf("status: got %s, want %s", got.Status, tc.want)
			}
			if len(reservations) != 1 || reservations[0].ID != res.ID {
				t.Fatalf("reservations: got %v, want [%s]", reservations, res.ID)
			}
		})
	}
}

func TestCancelBooking(t *testing.T) {
	ctx := context.Background()
	duringHold := created.Add(time.Minute)
	afterHold := created.Add(booking.HoldDuration)

	cases := []struct {
		name    string
		status  booking.BookingStatus
		now     time.Time
		wantErr error
	}{
		{"pending", booking.BookingPending, duringHold, nil},
		{"confirmed", booking.BookingConfirmed, duringHold, nil},
		{"expired", booking.BookingPending, afterHold, booking.ErrBookingNotActive},
		{"already cancelled", booking.BookingCancelled, duringHold, booking.ErrBookingNotActive},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := pendingBooking()
			b.Status = tc.status
			repo := newRepo(t, b)
			svc := booking.NewService(repo, clockAt(tc.now))

			got, err := svc.CancelBooking(ctx, b.ID)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			if got.Status != booking.BookingCancelled {
				t.Fatalf("returned %s, want cancelled", got.Status)
			}

			// the change must be saved, not only returned
			stored, _, err := svc.GetBooking(ctx, b.ID)
			if err != nil {
				t.Fatal(err)
			}
			if stored.Status != booking.BookingCancelled {
				t.Fatalf("stored %s, want cancelled", stored.Status)
			}
		})
	}
}

func TestConfirmBooking(t *testing.T) {
	ctx := context.Background()
	duringHold := created.Add(time.Minute)
	afterHold := created.Add(booking.HoldDuration)

	cases := []struct {
		name            string
		status          booking.BookingStatus
		hasReservations bool
		now             time.Time
		wantErr         error
	}{
		{"pending with reservations", booking.BookingPending, true, duringHold, nil},
		{"pending but empty", booking.BookingPending, false, duringHold, booking.ErrBookingEmpty},
		{"expired", booking.BookingPending, true, afterHold, booking.ErrBookingNotPending},
		{"already confirmed", booking.BookingConfirmed, true, duringHold, booking.ErrBookingNotPending},
		{"cancelled", booking.BookingCancelled, true, duringHold, booking.ErrBookingNotPending},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := pendingBooking()
			b.Status = tc.status
			var reservations []booking.Reservation
			if tc.hasReservations {
				reservations = append(reservations, booking.Reservation{ID: newID(), BookingID: b.ID})
			}
			svc := booking.NewService(newRepo(t, b, reservations...), clockAt(tc.now))

			got, err := svc.ConfirmBooking(ctx, b.ID)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			if got.Status != booking.BookingConfirmed {
				t.Fatalf("returned %s, want confirmed", got.Status)
			}

			// the change must be saved, not only returned
			stored, _, err := svc.GetBooking(ctx, b.ID)
			if err != nil {
				t.Fatal(err)
			}
			if stored.Status != booking.BookingConfirmed {
				t.Fatalf("stored %s, want confirmed", stored.Status)
			}
		})
	}
}

func TestRemoveReservation(t *testing.T) {
	ctx := context.Background()
	duringHold := created.Add(time.Minute)
	afterHold := created.Add(booking.HoldDuration)

	cases := []struct {
		name         string
		status       booking.BookingStatus
		now          time.Time
		otherBooking bool // try to remove it through a different booking
		wantErr      error
	}{
		{"pending", booking.BookingPending, duringHold, false, nil},
		{"expired", booking.BookingPending, afterHold, false, booking.ErrBookingNotPending},
		{"confirmed", booking.BookingConfirmed, duringHold, false, booking.ErrBookingNotPending},
		{"not this booking's", booking.BookingPending, duringHold, true, booking.ErrReservationNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := pendingBooking()
			b.Status = tc.status
			res := booking.Reservation{ID: newID(), BookingID: b.ID}
			repo := newRepo(t, b, res)

			bookingID := b.ID
			if tc.otherBooking {
				other := pendingBooking()
				if err := repo.Update(ctx, func(tx booking.Tx) error { return tx.InsertBooking(ctx, other) }); err != nil {
					t.Fatal(err)
				}
				bookingID = other.ID
			}
			svc := booking.NewService(repo, clockAt(tc.now))

			err := svc.RemoveReservation(ctx, bookingID, res.ID)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}

			// the reservation is gone only if the call succeeded
			_, reservations, err := svc.GetBooking(ctx, b.ID)
			if err != nil {
				t.Fatal(err)
			}
			if wantLeft := 1; tc.wantErr == nil {
				wantLeft = 0
				if len(reservations) != wantLeft {
					t.Fatalf("got %d reservations, want %d", len(reservations), wantLeft)
				}
			} else if len(reservations) != wantLeft {
				t.Fatalf("got %d reservations, want %d", len(reservations), wantLeft)
			}
		})
	}
}

func TestSeatMap(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepository()
	now := created.Add(time.Minute)

	// a show in a room with 5 seats.
	room := booking.Room{ID: newID(), Name: "Room 1"}
	show := booking.Show{ID: newID(), RoomID: room.ID}
	a1 := booking.Seat{ID: newID(), RoomID: room.ID, Row: "A", Number: 1}
	a2 := booking.Seat{ID: newID(), RoomID: room.ID, Row: "A", Number: 2}
	a3 := booking.Seat{ID: newID(), RoomID: room.ID, Row: "A", Number: 3}
	a4 := booking.Seat{ID: newID(), RoomID: room.ID, Row: "A", Number: 4}
	b1 := booking.Seat{ID: newID(), RoomID: room.ID, Row: "B", Number: 1}

	// and one booking in each state, each holding one seat
	confirmed := pendingBooking()
	confirmed.Status = booking.BookingConfirmed
	inCart := pendingBooking() // its hold lasts until created + 10 minutes
	expired := pendingBooking()
	expired.ExpiresAt = now.Add(-time.Second) // its hold is already over
	cancelled := pendingBooking()
	cancelled.Status = booking.BookingCancelled

	reservation := func(b booking.Booking, seat booking.Seat) booking.Reservation {
		return booking.Reservation{ID: newID(), BookingID: b.ID, ShowID: show.ID, SeatIDs: []uuid.UUID{seat.ID}}
	}

	err := repo.Update(ctx, func(tx booking.Tx) error {
		// the room first: seats and the show point to it
		tx.InsertRoom(ctx, room)
		tx.InsertShow(ctx, show)
		// seats saved in a shuffled order
		tx.InsertSeat(ctx, b1)
		tx.InsertSeat(ctx, a3)
		tx.InsertSeat(ctx, a1)
		tx.InsertSeat(ctx, a4)
		tx.InsertSeat(ctx, a2)

		tx.InsertBooking(ctx, confirmed)
		tx.InsertBooking(ctx, inCart)
		tx.InsertBooking(ctx, expired)
		tx.InsertBooking(ctx, cancelled)
		tx.InsertReservation(ctx, reservation(confirmed, a1))
		tx.InsertReservation(ctx, reservation(inCart, a2))
		tx.InsertReservation(ctx, reservation(expired, a3))
		tx.InsertReservation(ctx, reservation(cancelled, a4))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// we ask for a seat map (all the seats of a show with their availability)
	got, err := booking.NewService(repo, clockAt(now)).SeatMap(ctx, show.ID)
	if err != nil {
		t.Fatal(err)
	}

	// only the seats of active bookings are held, in row and number order
	want := []struct {
		seat   booking.Seat
		status booking.SeatStatus
	}{
		{a1, booking.SeatHeld},      // confirmed
		{a2, booking.SeatHeld},      // in a cart that hasn't expired
		{a3, booking.SeatAvailable}, // the cart expired
		{a4, booking.SeatAvailable}, // the booking was cancelled
		{b1, booking.SeatAvailable}, // nobody booked it
	}
	if len(got) != len(want) {
		t.Fatalf("got %d seats, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].ID != w.seat.ID || got[i].Status != w.status {
			t.Errorf("position %d: got %s%d %s, want %s%d %s",
				i, got[i].Row, got[i].Number, got[i].Status, w.seat.Row, w.seat.Number, w.status)
		}
	}
}

func TestSeatMapShowNotFound(t *testing.T) {
	ctx := context.Background()
	svc := booking.NewService(memory.NewRepository(), time.Now)

	// we ask for the seat map of a show that doesn't exist
	_, err := svc.SeatMap(ctx, newID())

	// we get ErrShowNotFound
	if !errors.Is(err, booking.ErrShowNotFound) {
		t.Fatalf("got %v, want ErrShowNotFound", err)
	}
}

// newShow saves a show in a room with seats A1, A2, A3 and returns
// the service, the show and the seats in row order.
func newShow(t *testing.T) (*booking.Service, booking.Show, []booking.Seat) {
	t.Helper()
	ctx := context.Background()
	repo := memory.NewRepository()

	room := booking.Room{ID: newID(), Name: "Room 1"}
	show := booking.Show{ID: newID(), RoomID: room.ID}
	seats := []booking.Seat{
		{ID: newID(), RoomID: room.ID, Row: "A", Number: 1},
		{ID: newID(), RoomID: room.ID, Row: "A", Number: 2},
		{ID: newID(), RoomID: room.ID, Row: "A", Number: 3},
	}
	err := repo.Update(ctx, func(tx booking.Tx) error {
		// the room first: seats and the show point to it
		tx.InsertRoom(ctx, room)
		tx.InsertShow(ctx, show)
		for _, seat := range seats {
			tx.InsertSeat(ctx, seat)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return booking.NewService(repo, clockAt(created)), show, seats
}

func TestCreateBookingChoosingSeats(t *testing.T) {
	ctx := context.Background()
	svc, show, seats := newShow(t)

	// we ask for A2 and A3
	req := booking.ReservationRequest{ShowID: show.ID, SeatIDs: []uuid.UUID{seats[1].ID, seats[2].ID}}
	b, reservations, secret, err := svc.CreateBooking(ctx, []booking.ReservationRequest{req})
	if err != nil {
		t.Fatal(err)
	}

	// we get a pending booking with exactly those seats, and a token
	if b.Status != booking.BookingPending || secret == "" {
		t.Fatalf("got status %s and token %q, want pending and a token", b.Status, secret)
	}
	if len(reservations) != 1 || !slices.Equal(reservations[0].SeatIDs, req.SeatIDs) {
		t.Fatalf("got %v, want seats A2 A3", reservations)
	}
}

func TestCreateBookingByQuantity(t *testing.T) {
	ctx := context.Background()
	svc, show, seats := newShow(t)

	// we ask for any 2 seats
	req := booking.ReservationRequest{ShowID: show.ID, Quantity: 2}
	_, reservations, _, err := svc.CreateBooking(ctx, []booking.ReservationRequest{req})
	if err != nil {
		t.Fatal(err)
	}

	// we get the first two, next to each other: A1 A2
	want := []uuid.UUID{seats[0].ID, seats[1].ID}
	if !slices.Equal(reservations[0].SeatIDs, want) {
		t.Fatalf("got %v, want A1 A2", reservations[0].SeatIDs)
	}
}

func TestCreateBookingSeatTaken(t *testing.T) {
	ctx := context.Background()
	svc, show, seats := newShow(t)
	a1 := booking.ReservationRequest{ShowID: show.ID, SeatIDs: []uuid.UUID{seats[0].ID}}

	// someone already booked A1
	if _, _, _, err := svc.CreateBooking(ctx, []booking.ReservationRequest{a1}); err != nil {
		t.Fatal(err)
	}

	// we ask for A1 too
	_, _, _, err := svc.CreateBooking(ctx, []booking.ReservationRequest{a1})

	// it's refused
	if !errors.Is(err, booking.ErrSeatUnavailable) {
		t.Fatalf("got %v, want ErrSeatUnavailable", err)
	}
}

func TestCreateBookingNotEnoughSeats(t *testing.T) {
	ctx := context.Background()
	svc, show, _ := newShow(t)

	// we ask for 4 seats in a room of 3
	req := booking.ReservationRequest{ShowID: show.ID, Quantity: 4}
	_, _, _, err := svc.CreateBooking(ctx, []booking.ReservationRequest{req})

	// it's refused
	if !errors.Is(err, booking.ErrNotEnoughSeats) {
		t.Fatalf("got %v, want ErrNotEnoughSeats", err)
	}
}

func TestCreateBookingAllOrNothing(t *testing.T) {
	ctx := context.Background()
	svc, show, _ := newShow(t)

	// one show is fine but the other doesn't exist
	reqs := []booking.ReservationRequest{
		{ShowID: show.ID, Quantity: 3},
		{ShowID: newID(), Quantity: 1},
	}
	_, _, _, err := svc.CreateBooking(ctx, reqs)

	// the whole booking fails
	if !errors.Is(err, booking.ErrShowNotFound) {
		t.Fatalf("got %v, want ErrShowNotFound", err)
	}
	// and the first show's seats are still all free: nothing was kept
	seatMap, err := svc.SeatMap(ctx, show.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range seatMap {
		if s.Status != booking.SeatAvailable {
			t.Fatalf("seat %s%d is %s, want available", s.Row, s.Number, s.Status)
		}
	}
}

func TestCreateBookingInvalid(t *testing.T) {
	svc, _, _ := newShow(t)

	// we send no reservations
	_, _, _, err := svc.CreateBooking(context.Background(), nil)

	// it's an invalid request
	if !errors.Is(err, booking.ErrInvalid) {
		t.Fatalf("got %v, want ErrInvalid", err)
	}
}

func TestAddReservationChangesSeats(t *testing.T) {
	ctx := context.Background()
	svc, show, seats := newShow(t)

	// create a booking holding A1
	a1 := booking.ReservationRequest{ShowID: show.ID, SeatIDs: []uuid.UUID{seats[0].ID}}
	b, reservations, _, err := svc.CreateBooking(ctx, []booking.ReservationRequest{a1})
	if err != nil {
		t.Fatal(err)
	}

	// we remove the show and add it again with A2: changing seats
	if err := svc.RemoveReservation(ctx, b.ID, reservations[0].ID); err != nil {
		t.Fatal(err)
	}
	a2 := booking.ReservationRequest{ShowID: show.ID, SeatIDs: []uuid.UUID{seats[1].ID}}
	res, err := svc.AddReservation(ctx, b.ID, a2)
	if err != nil {
		t.Fatal(err)
	}

	// the new reservation holds A2
	if !slices.Equal(res.SeatIDs, a2.SeatIDs) {
		t.Fatalf("returned %v, want A2", res.SeatIDs)
	}

	// and the stored booking holds only A2
	_, stored, err := svc.GetBooking(ctx, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 || stored[0].ID != res.ID {
		t.Fatalf("stored %v, want only the new reservation %s", stored, res.ID)
	}
}

func TestAddReservationShowAlreadyInBooking(t *testing.T) {
	ctx := context.Background()
	svc, show, seats := newShow(t)

	// we create a booking holding A1
	a1 := booking.ReservationRequest{ShowID: show.ID, SeatIDs: []uuid.UUID{seats[0].ID}}
	b, _, _, err := svc.CreateBooking(ctx, []booking.ReservationRequest{a1})
	if err != nil {
		t.Fatal(err)
	}

	// we add the same show again
	more := booking.ReservationRequest{ShowID: show.ID, Quantity: 1}
	_, err = svc.AddReservation(ctx, b.ID, more)

	// it's refused: one reservation per show
	if !errors.Is(err, booking.ErrShowAlreadyBooked) {
		t.Fatalf("got %v, want ErrShowAlreadyBooked", err)
	}
}

func TestAddReservationNotPending(t *testing.T) {
	ctx := context.Background()
	svc, show, seats := newShow(t)

	// we create and confirm a booking
	a1 := booking.ReservationRequest{ShowID: show.ID, SeatIDs: []uuid.UUID{seats[0].ID}}
	b, _, _, err := svc.CreateBooking(ctx, []booking.ReservationRequest{a1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmBooking(ctx, b.ID); err != nil {
		t.Fatal(err)
	}

	// if we try to add a reservation to a confirmed booking
	_, err = svc.AddReservation(ctx, b.ID, booking.ReservationRequest{ShowID: show.ID, Quantity: 1})

	// it's refused: only a pending booking can change
	if !errors.Is(err, booking.ErrBookingNotPending) {
		t.Fatalf("got %v, want ErrBookingNotPending", err)
	}
}

func TestAddReservationInvalid(t *testing.T) {
	ctx := context.Background()
	svc, show, _ := newShow(t)

	// we create a reservation request without a quantity or a show ids.
	_, err := svc.AddReservation(ctx, newID(), booking.ReservationRequest{ShowID: show.ID})

	// it's an invalid request, refused before looking at any booking
	if !errors.Is(err, booking.ErrInvalid) {
		t.Fatalf("got %v, want ErrInvalid", err)
	}
}
