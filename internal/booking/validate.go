package booking

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Validate reports every problem of a single reservation request.
// The error wraps ErrInvalid.
func (r ReservationRequest) Validate() error {
	return invalid(r.problems(""))
}

// problems returns every problem of r, each message starting with prefix
// to say where r is in the request (like "reservations[1]: ").
func (r ReservationRequest) problems(prefix string) []string {
	var problems []string
	if r.ShowID == uuid.Nil {
		problems = append(problems, prefix+"show_id is required")
	}

	// nil slice or also empty slice length is 0
	hasSeats := len(r.SeatIDs) > 0
	hasQuantity := r.Quantity > 0

	if r.Quantity < 0 {
		problems = append(problems, prefix+"quantity cannot be negative")
	} else if hasSeats == hasQuantity {
		// require exactly one of seat_ids or a positive quantity
		problems = append(problems, prefix+"specify either seat_ids or a positive quantity")
	}

	seen := make(map[uuid.UUID]bool)
	for _, id := range r.SeatIDs {
		switch {
		case id == uuid.Nil:
			problems = append(problems, prefix+"seat IDs cannot be nil")
		case seen[id]:
			problems = append(problems, fmt.Sprintf("%sduplicate seat ID: %s", prefix, id))
		}
		seen[id] = true
	}
	return problems
}

// ValidateReservations reports every problem of the reservations of a new
// booking. A booking needs at least one reservation, and one per show.
// The error wraps ErrInvalid.
func ValidateReservations(reqs []ReservationRequest) error {
	// we reject an empty list
	if len(reqs) == 0 {
		return invalid([]string{"at least one reservation is required"})
	}

	var problems []string
	// we enforce 1 reservation -> 1 show
	// you can't make multiple reservation requests for the same show.
	seenShows := make(map[uuid.UUID]bool)
	for i, r := range reqs {
		prefix := fmt.Sprintf("reservations[%d]: ", i)
		problems = append(problems, r.problems(prefix)...)
		if r.ShowID != uuid.Nil && seenShows[r.ShowID] {
			problems = append(problems, fmt.Sprintf("%sduplicate show_id: %s", prefix, r.ShowID))
		}
		seenShows[r.ShowID] = true
	}
	return invalid(problems)
}

// validationError holds every problem of a request. It wraps ErrInvalid.
type validationError struct {
	problems []string
}

func (e validationError) Error() string {
	return "invalid request: " + strings.Join(e.problems, "; ")
}

// Unwrap makes errors.Is(err, ErrInvalid) true.
func (e validationError) Unwrap() error { return ErrInvalid }

// invalid returns the problems as one validation error,
// or nil if there are none.
func invalid(problems []string) error {
	if len(problems) == 0 {
		return nil
	}
	return validationError{problems: problems}
}

// Problems returns the problems of a validation error, or nil if err isn't one.
func Problems(err error) []string {
	var v validationError
	if errors.As(err, &v) {
		return v.problems
	}
	return nil
}
