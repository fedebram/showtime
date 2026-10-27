package server

import (
	"net/http"

	"github.com/fedebram/showtime/internal/store"
)

type Server struct {
	store *store.MemoryStore
	mux   *http.ServeMux
}

func NewServer(memory *store.MemoryStore) *Server {
	s := &Server{store: memory, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /shows", s.listShows)
	s.mux.HandleFunc("GET /shows/{id}", s.getShow)
	s.mux.HandleFunc("GET /shows/{id}/seats", s.listAvailableSeats)

	s.mux.HandleFunc("POST /bookings", s.createBooking)

	// restish convention.
	// these routes need the booking token to be accessed.
	s.mux.HandleFunc("GET /bookings/{id}", s.requireBookingToken(s.getBooking))
	s.mux.HandleFunc("DELETE /bookings/{id}", s.requireBookingToken(s.cancelBooking))
	s.mux.HandleFunc("POST /bookings/{id}/confirm", s.requireBookingToken(s.confirmBooking))
	s.mux.HandleFunc("POST /bookings/{id}/reservations", s.requireBookingToken(notImplemented))
	s.mux.HandleFunc("DELETE /bookings/{id}/reservations/{rid}", s.requireBookingToken(s.removeReservation))

	return s
}

// we need serve http to satisfy the handler interface

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// notImplemented is a placeholder for routes we haven't written yet.
func notImplemented(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
