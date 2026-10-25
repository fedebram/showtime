package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

func (s *Server) listShows(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	// waste space, but the json then is more readable
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(s.store.ListShows()); err != nil {
		log.Printf("encode shows: %v", err)
	}
}

func (s *Server) getShow(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		http.Error(w, "invalid show ID", http.StatusBadRequest)
		return
	}
	show, ok := s.store.GetShow(id)
	if !ok {
		http.Error(w, "show not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(show); err != nil {
		log.Printf("encode response: %v", err)
	}
}

// TODO: add validation for id (eg. non empty)

func (s *Server) listAvailableSeats(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		http.Error(w, "invalid show ID", http.StatusBadRequest)
		return
	}

	seats, err := s.store.AvailableSeats(id)
	if err != nil {
		http.Error(w, "show not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(seats); err != nil {
		log.Printf("encode available seats: %v", err)
	}
}
