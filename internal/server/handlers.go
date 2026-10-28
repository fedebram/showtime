package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/fedebram/showtime/internal/booking"
	"github.com/fedebram/showtime/internal/token"
)

// TODO: return error as json bodies. No text.

func (s *Server) listShows(w http.ResponseWriter, r *http.Request) {
	shows := s.store.ListShows()
	writeJSON(w, http.StatusOK, shows)
}

func (s *Server) getShow(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
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
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
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

	// if the body is empty the validation doesn't return an error.
	// it is good. when creating the booking, we are free to specify or not the reservations.
	if err := booking.ValidateReservations(req.Reservations); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	secret, hash := token.GenerateToken()

	b, reservations, err := s.store.CreateBooking(hash, req.Reservations)
	if err != nil {
		writeBookingError(w, err)
		return
	}

	// the secret is returned only here. the client sends it back as a bearer, we store only its hash
	writeJSON(w, http.StatusCreated, CreateBookingResponse{
		Booking:      b,
		Reservations: reservations,
		Token:        secret,
	})
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

func (s *Server) getBooking(w http.ResponseWriter, r *http.Request) {
	b := bookingFrom(r.Context())

	b.Status = b.CurrentStatus(time.Now())

	writeJSON(w, http.StatusOK, BookingResponse{
		Booking:      b,
		Reservations: s.store.ListReservations(b.ID),
	})
}

func (s *Server) confirmBooking(w http.ResponseWriter, r *http.Request) {
	b := bookingFrom(r.Context())

	confirmed, err := s.store.ConfirmBooking(b.ID)
	if err != nil {
		writeBookingError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, confirmed)
}

func (s *Server) cancelBooking(w http.ResponseWriter, r *http.Request) {
	b := bookingFrom(r.Context())

	if err := s.store.CancelBooking(b.ID); err != nil {
		writeBookingError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) removeReservation(w http.ResponseWriter, r *http.Request) {
	b := bookingFrom(r.Context())

	rid, err := uuid.Parse(r.PathValue("rid"))
	if err != nil {
		http.Error(w, "invalid reservation ID", http.StatusBadRequest)
		return
	}

	if err := s.store.RemoveReservation(b.ID, rid); err != nil {
		writeBookingError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
