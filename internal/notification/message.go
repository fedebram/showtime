package notification

import (
	"time"

	"github.com/google/uuid"
)

// Message is a notification to send.
type Message struct {
	ID        uuid.UUID `json:"id"`
	Kind      string    `json:"kind"`
	BookingID uuid.UUID `json:"booking_id"`
	SentAt    time.Time `json:"sent_at"`
}
