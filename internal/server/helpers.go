package server

import (
	"errors"
	"log"
	"net/http"

	"github.com/fedebram/showtime/internal/booking"
	"github.com/fedebram/showtime/internal/store"
)

// writeBookingError maps the store errors to http status codes.
func writeBookingError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrShowNotFound),
		errors.Is(err, booking.ErrReservationNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, booking.ErrSeatUnavailable),
		errors.Is(err, booking.ErrNotEnoughSeats),
		errors.Is(err, booking.ErrBookingNotPending),
		errors.Is(err, booking.ErrBookingNotActive),
		errors.Is(err, booking.ErrBookingEmpty):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		log.Printf("booking: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
