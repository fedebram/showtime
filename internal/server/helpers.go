package server

import (
	"errors"
	"log"
	"net/http"

	"github.com/fedebram/showtime/internal/booking"
)

// errorResponse is the body of every error response.
type errorResponse struct {
	Errors []string `json:"errors"`
}

// writeErrors answers status with the given messages.
func writeErrors(w http.ResponseWriter, status int, msgs ...string) {
	writeJSON(w, status, errorResponse{Errors: msgs})
}

// writeError maps the booking errors to http status codes.
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, booking.ErrInvalid):
		// one message per problem
		writeErrors(w, http.StatusBadRequest, booking.Problems(err)...)
	case errors.Is(err, booking.ErrShowNotFound),
		errors.Is(err, booking.ErrBookingNotFound),
		errors.Is(err, booking.ErrReservationNotFound):
		writeErrors(w, http.StatusNotFound, err.Error())
	case errors.Is(err, booking.ErrSeatUnavailable),
		errors.Is(err, booking.ErrNotEnoughSeats),
		errors.Is(err, booking.ErrBookingNotPending),
		errors.Is(err, booking.ErrBookingNotActive),
		errors.Is(err, booking.ErrBookingEmpty),
		errors.Is(err, booking.ErrShowAlreadyBooked):
		writeErrors(w, http.StatusConflict, err.Error())
	default:
		// never show internal details to the client
		log.Printf("internal error: %v", err)
		writeErrors(w, http.StatusInternalServerError, "internal error")
	}
}
