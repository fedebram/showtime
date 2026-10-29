package seed

import (
	"context"
	"time"

	"github.com/fedebram/showtime/internal/booking"
	"github.com/google/uuid"
)

// Seed fills repo with demo data: two rooms of six seats and one show in each.
// The shows start tomorrow, at 18:00 and 21:00 in now's time zone.
func Seed(ctx context.Context, repo booking.Repository, now time.Time) error {
	room1, room2 := newID(), newID()

	tomorrow := now.AddDate(0, 0, 1)
	at := func(hour int) time.Time {
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), hour, 0, 0, 0, now.Location())
	}
	shows := []booking.Show{
		{ID: newID(), MovieTitle: "The Big Lebowski", RoomID: room1, StartsAt: at(18)},
		{ID: newID(), MovieTitle: "The Flintstones", RoomID: room2, StartsAt: at(21)},
	}

	return repo.Update(ctx, func(tx booking.Tx) error {
		// rows A and B with 3 seats each, in both rooms
		for _, room := range []uuid.UUID{room1, room2} {
			for _, row := range []string{"A", "B"} {
				for number := 1; number <= 3; number++ {
					seat := booking.Seat{ID: newID(), RoomID: room, Row: row, Number: number}
					if err := tx.InsertSeat(ctx, seat); err != nil {
						return err
					}
				}
			}
		}
		for _, show := range shows {
			if err := tx.InsertShow(ctx, show); err != nil {
				return err
			}
		}
		return nil
	})
}

// newID returns a time ordered uuid (v7).
func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }
