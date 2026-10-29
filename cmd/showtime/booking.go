package main

import (
	"fmt"
	"log/slog"
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
	var addr, db string
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

			return serve(ctx, addr, bookingserver.NewServer(bookings))
		},
	}
	cmd.Flags().StringVar(&addr, "addr", ":8080", "listen address")
	cmd.Flags().StringVar(&db, "db", "", "SQLite database file (empty: in memory)")
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
