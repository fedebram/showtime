package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/fedebram/showtime/internal/booking"
	"github.com/fedebram/showtime/internal/token"
)

// TODO: return error as json bodies. No text.

func (s *Server) listShows(w http.ResponseWriter, r *http.Request) {
	shows := s.store.ListShows()
	writeJSON(w, http.StatusOK, shows)
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
	writeJSON(w, http.StatusOK, show)
}

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

	writeJSON(w, http.StatusOK, seats)
}

type CreateBookingRequest struct {
	Reservations []booking.ReservationRequest `json:"reservations"`
}

func (s *Server) createBooking(w http.ResponseWriter, r *http.Request) {
	var req CreateBookingRequest

	const maxRequestBodyBytes = 1_000_000 // 1 MB
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodyBytes))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// we reject additional json
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		http.Error(w, "expected a single JSON object", http.StatusBadRequest)
		return
	}

	for _, reservation := range req.Reservations {
		if err := reservation.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	secret, hash := token.GenerateToken()
}
