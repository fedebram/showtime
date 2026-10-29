package server

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/fedebram/showtime/internal/booking"
	"github.com/fedebram/showtime/internal/httpjson"
)

// pretty basic middleware where we hydrate the context key with the booking

type bookingKey struct{}

func (s *Server) requireBookingToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			httpjson.WriteErrors(w, http.StatusBadRequest, "invalid booking ID")
			return
		}

		secret, ok := bearerToken(r)
		if !ok {
			httpjson.WriteErrors(w, http.StatusUnauthorized, "missing token")
			return
		}

		// wrong token and missing booking give the same 404.
		// this way someone can't enumerate bookings.
		b, err := s.bookings.Authenticate(r.Context(), id, secret)
		if err != nil {
			writeError(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), bookingKey{}, b)
		next(w, r.WithContext(ctx))
	}
}

func bookingFrom(ctx context.Context) booking.Booking {
	// it panics if we don't set the key!! which is nice
	return ctx.Value(bookingKey{}).(booking.Booking)
}
