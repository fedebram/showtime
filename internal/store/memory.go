package store

import (
	"sort"

	"github.com/fedebram/showtime/internal/booking"
)

type MemoryStore struct {
	shows map[int]booking.Show
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		shows: map[int]booking.Show{
			1: {ID: 1, MovieTitle: "The Big Lebowski", RoomID: 1, StartsAt: "2026-10-24T18:00:00+02:00"},
			2: {ID: 2, MovieTitle: "The Flintstones", RoomID: 1, StartsAt: "2026-10-24T21:00:00+02:00"},
		},
	}
}

func (m *MemoryStore) ListShows() []booking.Show {
	shows := make([]booking.Show, 0, len(m.shows))
	for _, show := range m.shows {
		shows = append(shows, show)
	}
	sort.Slice(shows, func(i, j int) bool {
		return shows[i].ID < shows[j].ID
	})
	return shows
}

func (m *MemoryStore) GetShow(id int) (booking.Show, bool) {
	show, ok := m.shows[id]
	return show, ok
}
