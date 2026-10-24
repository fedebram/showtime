package show

// Show represents a movie playing in a room at a scheduled time.
type Show struct {
	ID         int    `json:"id"`
	MovieTitle string `json:"movie_title"`
	RoomID     int    `json:"room_id"`
	StartsAt   string `json:"starts_at"`
}
