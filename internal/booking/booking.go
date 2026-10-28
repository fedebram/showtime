package booking

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// HoldDuration is how long a pending booking keeps its seats.
const HoldDuration = 10 * time.Minute

func AssignSeats(available []Seat, req ReservationRequest) ([]uuid.UUID, error) {
	// we assign seats from the quantity or from the seatIDs slice.

	if req.Quantity > 0 {
		if len(available) < req.Quantity {
			return nil, ErrNotEnoughSeats
		}
		seatIDs := make([]uuid.UUID, 0, req.Quantity)
		// from index 0 through quantity-1
		for _, seat := range available[:req.Quantity] {
			seatIDs = append(seatIDs, seat.ID)
		}
		return seatIDs, nil
	}

	free := make(map[uuid.UUID]bool, len(available))
	for _, seat := range available {
		free[seat.ID] = true
	}
	seatIDs := make([]uuid.UUID, 0, len(req.SeatIDs))
	for _, id := range req.SeatIDs {
		if !free[id] {
			return nil, fmt.Errorf("%w: %s", ErrSeatUnavailable, id)
		}
		seatIDs = append(seatIDs, id)
	}
	return seatIDs, nil
}
