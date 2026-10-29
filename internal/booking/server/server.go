package server

import (
	"net/http"

	"github.com/fedebram/showtime/internal/booking"
)

type Server struct {
	bookings *booking.Service
	mux      *http.ServeMux
}

func NewServer(bookings *booking.Service) *Server {
	s := &Server{bookings: bookings, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /shows", s.listShows)
	s.mux.HandleFunc("GET /shows/{id}", s.getShow)
	s.mux.HandleFunc("GET /shows/{id}/seats", s.seatMap)

	s.mux.HandleFunc("POST /bookings", s.createBooking)

	// restish convention.
	// these routes need the booking token to be accessed.
	s.mux.HandleFunc("GET /bookings/{id}", s.requireBookingToken(s.getBooking))
	// nothing is deleted: cancelling changes the status, so it's a POST
	s.mux.HandleFunc("POST /bookings/{id}/cancel", s.requireBookingToken(s.cancelBooking))
	s.mux.HandleFunc("POST /bookings/{id}/confirm", s.requireBookingToken(s.confirmBooking))
	s.mux.HandleFunc("POST /bookings/{id}/reservations", s.requireBookingToken(s.addReservation))
	s.mux.HandleFunc("DELETE /bookings/{id}/reservations/{rid}", s.requireBookingToken(s.removeReservation))

	return s
}

// we need serve http to satisfy the handler interface

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
