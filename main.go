package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/fedebram/showtime/internal/booking"
	"github.com/fedebram/showtime/internal/booking/memory"
	"github.com/fedebram/showtime/internal/booking/sqlite"
	"github.com/fedebram/showtime/internal/seed"
	"github.com/fedebram/showtime/internal/server"
)

func main() {
	dbPath := flag.String("db", "", "SQLite database file (empty: in memory)")
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()

	ctx := context.Background()

	var repo booking.Repository
	if *dbPath == "" {
		repo = memory.NewRepository()
		log.Println("using in-memory repository")
	} else {
		sqlRepo, err := sqlite.Open(*dbPath)
		if err != nil {
			log.Fatalf("open db: %v", err)
		}
		defer sqlRepo.Close()
		repo = sqlRepo
		log.Printf("using SQLite at %s", *dbPath)
	}

	if err := seed.Seed(ctx, repo, time.Now()); err != nil {
		log.Fatalf("seed: %v", err)
	}
	bookings := booking.NewService(repo, time.Now)
	s := server.NewServer(bookings)

	log.Printf("Server running at http://localhost%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, s))
}
