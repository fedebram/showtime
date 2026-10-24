package booking

// right now we place everything inside booking. It is fine like that

// Show represents a movie playing in a room at a scheduled time.
type Show struct {
	ID         int    `json:"id"`
	MovieTitle string `json:"movie_title"`
	RoomID     int    `json:"room_id"`
	StartsAt   string `json:"starts_at"`
}

// Seat represents a physical seat in a room.
type Seat struct {
	ID     int    `json:"id"`
	RoomID int    `json:"room_id"`
	Row    string `json:"row"`
	Number int    `json:"number"`
}
