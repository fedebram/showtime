package booking

// right now we place everything inside booking. It is fine like that

// Room represents a screening room in the cinema.
type Room struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

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

// Reservation records seats booked for a show.
type Reservation struct {
	ID      int   `json:"id"`
	ShowID  int   `json:"show_id"`
	SeatIDs []int `json:"seat_ids"`
}
