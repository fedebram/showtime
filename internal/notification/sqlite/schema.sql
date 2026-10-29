-- the messages already sent. The id comes from the sender:
-- the primary key is what makes a repeat impossible to store twice.
-- no foreign key on booking_id: bookings live in another service's database.
CREATE TABLE IF NOT EXISTS notifications (
    id         TEXT PRIMARY KEY,
    kind       TEXT NOT NULL,
    booking_id TEXT NOT NULL, --to have the bookings maybe a fat outbox message?
    sent_at    TEXT NOT NULL
) STRICT;
