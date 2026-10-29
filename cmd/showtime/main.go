package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// after the first signal, a second one kills the process right away
	context.AfterFunc(ctx, stop)

	if err := newRootCmd().ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "showtime",
		Short: "Cinema bookings: the booking and notification services",
		// a failing server is not a usage mistake: don't print the help
		SilenceUsage: true,
	}
	root.AddCommand(newServeCmd())
	return root
}
