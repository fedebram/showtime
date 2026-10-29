package server

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/fedebram/showtime/internal/booking"
)

func (s *Server) listShows(w http.ResponseWriter, r *http.Request) {
	shows, err := s.bookings.ListShows(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, shows)
}

func (s *Server) getShow(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeErrors(w, http.StatusBadRequest, "invalid show ID")
		return
	}
	show, err := s.bookings.GetShow(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, show)
}

type SeatMapResponse struct {
	ShowID    uuid.UUID                  `json:"show_id"`
	Total     int                        `json:"total"`
	Available int                        `json:"available"`
	Seats     []booking.SeatAvailability `json:"seats"`
}

func (s *Server) seatMap(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeErrors(w, http.StatusBadRequest, "invalid show ID")
		return
	}
	seats, err := s.bookings.SeatMap(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	available := 0
	for _, seat := range seats {
		if seat.Status == booking.SeatAvailable {
			available++
		}
	}
	writeJSON(w, http.StatusOK, SeatMapResponse{
		ShowID:    id,
		Total:     len(seats),
		Available: available,
		Seats:     seats,
	})
}

type CreateBookingRequest struct {
	Reservations []booking.ReservationRequest `json:"reservations"`
}

type CreateBookingResponse struct {
	Booking      booking.Booking       `json:"booking"`
	Reservations []booking.Reservation `json:"reservations"`
	Token        string                `json:"token"`
}

type BookingResponse struct {
	Booking      booking.Booking       `json:"booking"`
	Reservations []booking.Reservation `json:"reservations"`
}

func (s *Server) createBooking(w http.ResponseWriter, r *http.Request) {
	var req CreateBookingRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	b, reservations, secret, err := s.bookings.CreateBooking(r.Context(), req.Reservations)
	if err != nil {
		writeError(w, err)
		return
	}

	// the secret is returned only here. the client sends it back as a bearer, we store only its hash
	writeJSON(w, http.StatusCreated, CreateBookingResponse{
		Booking:      b,
		Reservations: reservations,
		Token:        secret,
	})
}

func (s *Server) getBooking(w http.ResponseWriter, r *http.Request) {
	b, reservations, err := s.bookings.GetBooking(r.Context(), bookingFrom(r.Context()).ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, BookingResponse{Booking: b, Reservations: reservations})
}

func (s *Server) confirmBooking(w http.ResponseWriter, r *http.Request) {
	confirmed, err := s.bookings.ConfirmBooking(r.Context(), bookingFrom(r.Context()).ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, confirmed)
}

func (s *Server) cancelBooking(w http.ResponseWriter, r *http.Request) {
	cancelled, err := s.bookings.CancelBooking(r.Context(), bookingFrom(r.Context()).ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cancelled)
}

func (s *Server) addReservation(w http.ResponseWriter, r *http.Request) {
	var req booking.ReservationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	// only what was added: the client keeps adding shows,
	// and reviews the full booking with GET /bookings/{id}
	res, err := s.bookings.AddReservation(r.Context(), bookingFrom(r.Context()).ID, req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func (s *Server) removeReservation(w http.ResponseWriter, r *http.Request) {
	b := bookingFrom(r.Context())

	rid, err := uuid.Parse(r.PathValue("rid"))
	if err != nil {
		writeErrors(w, http.StatusBadRequest, "invalid reservation ID")
		return
	}
	if err := s.bookings.RemoveReservation(r.Context(), b.ID, rid); err != nil {
		writeError(w, err)
		return
	}
	// nothing to return: the client reviews the booking with GET /bookings/{id}
	w.WriteHeader(http.StatusNoContent)
}
