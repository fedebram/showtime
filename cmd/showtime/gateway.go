package main

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/fedebram/showtime/internal/gateway"
)

// newServeGatewayCmd runs the gateway.
func newServeGatewayCmd() *cobra.Command {
	var addr, bookingURL string
	cmd := &cobra.Command{
		Use:   "gateway",
		Short: "The public entry point",
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := url.Parse(bookingURL)
			// we just require scheme and host. url parse accepts almost anything
			if err != nil || target.Scheme == "" || target.Host == "" {
				return fmt.Errorf("invalid --booking-url %q: want like http://localhost:8080", bookingURL)
			}
			return serve(cmd.Context(), addr, gateway.New(target))
		},
	}
	cmd.Flags().StringVar(&addr, "addr", ":8000", "listen address")
	cmd.Flags().StringVar(&bookingURL, "booking-url", "http://localhost:8080", "booking service base URL")
	return cmd
}
