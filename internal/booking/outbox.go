package booking

import (
	"time"

	"github.com/google/uuid"
)

// OutboxKind tells the receiver which notification to send.
type OutboxKind string

const (
	OutboxBookingConfirmed OutboxKind = "booking_confirmed"
	OutboxBookingCancelled OutboxKind = "booking_cancelled"
)

// OutboxMessage is a notification waiting to be delivered.
type OutboxMessage struct {
	ID        uuid.UUID
	Kind      OutboxKind
	BookingID uuid.UUID
	CreatedAt time.Time
	SentAt    time.Time
}
