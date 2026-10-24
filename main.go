package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/fedebram/showtime/internal/show"
)

type Show struct {
	ID         int    `json:"id"`
	MovieTitle string `json:"movieTitle"`
	RoomID     int    `json:"roomId"`
	StartsAt   string `json:"startsAt"`
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello qubica!")
	})

	store := show.NewStore()
	handler := show.NewHandler(store)
	mux.HandleFunc("GET /shows", handler.List)
	mux.HandleFunc("GET /shows/{id}", handler.Get)

	log.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
