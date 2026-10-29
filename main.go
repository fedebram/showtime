package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/fedebram/showtime/internal/booking"
	"github.com/fedebram/showtime/internal/booking/memory"
	"github.com/fedebram/showtime/internal/seed"
	"github.com/fedebram/showtime/internal/server"
)

func main() {
	ctx := context.Background()
	memRepo := memory.NewRepository()
	if err := seed.Seed(ctx, memRepo, time.Now()); err != nil {
		log.Fatalf("seed: %v", err)
	}
	bookings := booking.NewService(memRepo, time.Now)
	s := server.NewServer(bookings)

	log.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", s))
}
