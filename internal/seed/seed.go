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
	room1 := booking.Room{ID: newID(), Name: "Room 1"}
	room2 := booking.Room{ID: newID(), Name: "Room 2"}

	tomorrow := now.AddDate(0, 0, 1)
	at := func(hour int) time.Time {
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), hour, 0, 0, 0, now.Location())
	}
	shows := []booking.Show{
		{ID: newID(), MovieTitle: "The Big Lebowski", RoomID: room1.ID, StartsAt: at(18)},
		{ID: newID(), MovieTitle: "The Flintstones", RoomID: room2.ID, StartsAt: at(21)},
	}

	return repo.Update(ctx, func(tx booking.Tx) error {
		// rooms first: seats and shows point to them
		for _, room := range []booking.Room{room1, room2} {
			if err := tx.InsertRoom(ctx, room); err != nil {
				return err
			}
			// rows A and B with 3 seats each
			for _, row := range []string{"A", "B"} {
				for number := 1; number <= 3; number++ {
					seat := booking.Seat{ID: newID(), RoomID: room.ID, Row: row, Number: number}
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
