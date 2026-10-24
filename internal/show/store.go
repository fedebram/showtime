package show

import "sort"

// show id -> show

type Store struct {
	shows map[int]Show
}

func NewStore() *Store {
	return &Store{
		shows: map[int]Show{
			1: {ID: 1, MovieTitle: "The Big Lebowski", RoomID: 1, StartsAt: "2026-10-24T18:00:00+02:00"},
			2: {ID: 2, MovieTitle: "The Flintstones", RoomID: 1, StartsAt: "2026-10-24T21:00:00+02:00"},
		},
	}
}

func (s *Store) List() []Show {
	shows := make([]Show, 0, len(s.shows))
	for _, show := range s.shows {
		shows = append(shows, show)
	}
	// we need to sort because iterating over a map doesn't guarantee an order.
	// this way the list is consistent
	sort.Slice(shows, func(i, j int) bool {
		return shows[i].ID < shows[j].ID
	})
	return shows
}

func (s *Store) Get(id int) (Show, bool) {
	show, ok := s.shows[id]
	return show, ok
}
