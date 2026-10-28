package booking

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Validate reports every problem of a single reservation request.
// The error wraps ErrInvalid.
func (r ReservationRequest) Validate() error {
	return invalid(r.problems(""))
}

// problems returns every problem of r, each message starting with prefix
// to say where r is in the request (like "reservations[1]: ").
func (r ReservationRequest) problems(prefix string) []error {
	var errs []error
	if r.ShowID == uuid.Nil {
		errs = append(errs, fmt.Errorf("%sshow_id is required", prefix))
	}

	// nil slice or also empty slice length is 0
	hasSeats := len(r.SeatIDs) > 0
	hasQuantity := r.Quantity > 0

	if r.Quantity < 0 {
		errs = append(errs, fmt.Errorf("%squantity cannot be negative", prefix))
	} else if hasSeats == hasQuantity {
		// require exactly one of seat_ids or a positive quantity
		errs = append(errs, fmt.Errorf("%sspecify either seat_ids or a positive quantity", prefix))
	}

	seen := make(map[uuid.UUID]bool)
	for _, id := range r.SeatIDs {
		switch {
		case id == uuid.Nil:
			errs = append(errs, fmt.Errorf("%sseat IDs cannot be nil", prefix))
		case seen[id]:
			errs = append(errs, fmt.Errorf("%sduplicate seat ID: %s", prefix, id))
		}
		seen[id] = true
	}
	return errs
}

// ValidateReservations reports every problem of the reservations of a new
// booking. A booking needs at least one reservation, and one per show.
// The error wraps ErrInvalid.
func ValidateReservations(reqs []ReservationRequest) error {
	// we reject an empty list
	if len(reqs) == 0 {
		return invalid([]error{errors.New("at least one reservation is required")})
	}

	var errs []error
	// we enforce 1 reservation -> 1 show
	// you can't make multiple reservation requests for the same show.
	seenShows := make(map[uuid.UUID]bool)
	for i, r := range reqs {
		prefix := fmt.Sprintf("reservations[%d]: ", i)
		errs = append(errs, r.problems(prefix)...)
		if r.ShowID != uuid.Nil && seenShows[r.ShowID] {
			errs = append(errs, fmt.Errorf("%sduplicate show_id: %s", prefix, r.ShowID))
		}
		seenShows[r.ShowID] = true
	}
	return invalid(errs)
}

// invalid joins the problems into one error wrapping ErrInvalid,
// or returns nil if there are none.
func invalid(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrInvalid, errors.Join(errs...))
}
