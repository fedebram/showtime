package show

import (
	"encoding/json"
	"log"
	"net/http"
)

func List(w http.ResponseWriter, r *http.Request) {
	shows := []Show{
		{ID: 1, MovieTitle: "The Big Lebowski", RoomID: 1, StartsAt: "2026-10-24T18:00:00+02:00"},
		{ID: 2, MovieTitle: "The Flintstones", RoomID: 1, StartsAt: "2026-10-24T21:00:00+02:00"},
	}

	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	// waste space, but the json then is more readable
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(shows); err != nil {
		log.Printf("encode shows: %v", err)
	}
}
