package server

import (
	"errors"
	"log"
	"net/http"

	"github.com/fedebram/showtime/internal/booking"
	"github.com/fedebram/showtime/internal/httpjson"
)

// writeError maps the booking errors to http status codes.
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, booking.ErrInvalid):
		// one message per problem
		httpjson.WriteErrors(w, http.StatusBadRequest, booking.Problems(err)...)
	case errors.Is(err, booking.ErrShowNotFound),
		errors.Is(err, booking.ErrBookingNotFound),
		errors.Is(err, booking.ErrReservationNotFound):
		httpjson.WriteErrors(w, http.StatusNotFound, err.Error())
	case errors.Is(err, booking.ErrSeatUnavailable),
		errors.Is(err, booking.ErrNotEnoughSeats),
		errors.Is(err, booking.ErrBookingNotPending),
		errors.Is(err, booking.ErrBookingNotActive),
		errors.Is(err, booking.ErrBookingEmpty),
		errors.Is(err, booking.ErrShowAlreadyBooked):
		httpjson.WriteErrors(w, http.StatusConflict, err.Error())
	default:
		// never show internal details to the client
		log.Printf("internal error: %v", err)
		httpjson.WriteErrors(w, http.StatusInternalServerError, "internal error")
	}
}
