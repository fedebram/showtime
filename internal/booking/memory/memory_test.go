package memory_test

// _test package is needed to avoid import cycle

import (
	"context"
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
