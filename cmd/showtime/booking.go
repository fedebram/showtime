package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/fedebram/showtime/internal/booking"
	bookingmemory "github.com/fedebram/showtime/internal/booking/memory"
	bookingserver "github.com/fedebram/showtime/internal/booking/server"
	bookingsqlite "github.com/fedebram/showtime/internal/booking/sqlite"
	"github.com/fedebram/showtime/internal/seed"
)

// newServeBookingCmd runs the booking service alone.
func newServeBookingCmd() *cobra.Command {
	var addr, db, notificationURL string
	cmd := &cobra.Command{
		Use:   "booking",
		Short: "Run the booking service: shows, seats and bookings",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			repo, closeRepo, err := openBookingRepo(db)
			if err != nil {
				return err
			}
			defer closeRepo()

			if err := seed.Seed(ctx, repo, time.Now()); err != nil {
				return fmt.Errorf("seed: %w", err)
			}
			bookings := booking.NewService(repo, time.Now)

			relay := booking.NewRelay(repo, newHTTPPublisher(notificationURL), time.Now)
			relayCtx, stopRelay := context.WithCancel(ctx)
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				relay.Run(relayCtx)
			}()

			err = serve(ctx, addr, bookingserver.NewServer(bookings))

			stopRelay()
			wg.Wait()

			return err
		},
	}
	cmd.Flags().StringVar(&addr, "addr", ":8080", "listen address")
	cmd.Flags().StringVar(&db, "db", "", "SQLite database file (empty: in memory)")
	cmd.Flags().StringVar(&notificationURL, "notification-url", "http://localhost:8081", "notification service base URL")
	return cmd
}

// openBookingRepo opens SQLite at path, or memory if path is empty.
// The returned func closes it.
func openBookingRepo(path string) (booking.Repository, func() error, error) {
	if path == "" {
		slog.Info("booking: using in-memory repository")
		return bookingmemory.NewRepository(), func() error { return nil }, nil
	}
	repo, err := bookingsqlite.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open booking db: %w", err)
	}
	slog.Info("booking: using SQLite", "path", path)
	return repo, repo.Close, nil
}
