package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fedebram/showtime/internal/booking"
	"github.com/fedebram/showtime/internal/booking/memory"
	"github.com/fedebram/showtime/internal/booking/sqlite"
	"github.com/fedebram/showtime/internal/seed"
	"github.com/fedebram/showtime/internal/server"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	dbPath := flag.String("db", "", "SQLite database file (empty: in memory)")
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var repo booking.Repository
	if *dbPath == "" {
		repo = memory.NewRepository()
		log.Println("using in-memory repository")
	} else {
		sqlRepo, err := sqlite.Open(*dbPath)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		defer sqlRepo.Close()
		repo = sqlRepo
		log.Printf("using SQLite at %s", *dbPath)
	}

	if err := seed.Seed(ctx, repo, time.Now()); err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	bookings := booking.NewService(repo, time.Now)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           server.NewServer(bookings),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("Server running at http://localhost%s", *addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
		stop()
		log.Println("shutting down...")
	}

	// stop accepting new connections and wait for in-flight requests to finish
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen: %w", err)
	}
	log.Println("server stopped")
	return nil
}
