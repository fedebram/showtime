package main

import (
	"log"
	"net/http"

	"github.com/fedebram/showtime/internal/server"
	"github.com/fedebram/showtime/internal/store"
)

func main() {
	memoryStore := store.NewMemoryStore()
	s := server.NewServer(memoryStore)

	log.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", s))
}
