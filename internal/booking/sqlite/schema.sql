-- every statement is idempotent: the schema runs at each Open.

CREATE TABLE IF NOT EXISTS rooms (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
) STRICT;

CREATE TABLE IF NOT EXISTS shows (
    id          TEXT PRIMARY KEY,
    movie_title TEXT NOT NULL,
    room_id     TEXT NOT NULL REFERENCES rooms (id),
    starts_at   TEXT NOT NULL
) STRICT;

-- seat_row and seat_number: ROW is a SQL keyword
CREATE TABLE IF NOT EXISTS seats (
    id          TEXT PRIMARY KEY,
    room_id     TEXT NOT NULL REFERENCES rooms (id),
    seat_row    TEXT NOT NULL,
    seat_number INTEGER NOT NULL CHECK (seat_number > 0),
    UNIQUE (room_id, seat_row, seat_number)
) STRICT;

CREATE TABLE IF NOT EXISTS bookings (
    id         TEXT PRIMARY KEY,
    -- expired is never stored: it is computed by Booking.CurrentStatus
    status     TEXT NOT NULL CHECK (status IN ('pending', 'confirmed', 'cancelled')),
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    token_hash TEXT NOT NULL
) STRICT;

CREATE TABLE IF NOT EXISTS reservations (
    id         TEXT PRIMARY KEY,
    booking_id TEXT NOT NULL REFERENCES bookings (id),
    show_id    TEXT NOT NULL REFERENCES shows (id)
) STRICT;
CREATE INDEX IF NOT EXISTS reservations_booking ON reservations (booking_id);
CREATE INDEX IF NOT EXISTS reservations_show ON reservations (show_id);

-- the SeatIDs list of a reservation
-- deleting a reservation deletes its seat links too.
CREATE TABLE IF NOT EXISTS reservation_seats (
    reservation_id TEXT NOT NULL REFERENCES reservations (id) ON DELETE CASCADE,
    seat_id        TEXT NOT NULL REFERENCES seats (id),
    -- a seat appears at most once in a reservation
    PRIMARY KEY (reservation_id, seat_id)
) STRICT;
