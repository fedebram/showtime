package booking

import "errors"

// just a bunch of sentinel errors.
// the purpose is that on the http layer we map through http status codes

var (
	ErrSeatUnavailable = errors.New("seat not available")
	ErrNotEnoughSeats  = errors.New("not enough seats available")
)
