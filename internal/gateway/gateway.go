// Package gateway is the single entry point for clients: a reverse proxy
// that forwards the public routes to the service that owns them.
package gateway

import (
	_ "embed"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/fedebram/showtime/internal/httpjson"
)

// New returns the gateway handler
func New(bookingURL *url.URL) http.Handler {
	booking := newProxy(bookingURL)

	mux := http.NewServeMux()
	mux.Handle("/shows", booking)
	mux.Handle("/shows/", booking)
	mux.Handle("/bookings", booking)
	mux.Handle("/bookings/", booking)
	return mux
}

// newProxy forwards every request to target, keeping path and query.
func newProxy(target *url.URL) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			// tell the service who the real client is
			r.SetXForwarded()
		},
		// the service is down or unreachable: answer like our other errors
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Error("gateway: proxy failed", "target", target.String(), "path", r.URL.Path, "err", err)
			httpjson.WriteErrors(w, http.StatusBadGateway, "service unavailable")
		},
	}
}
