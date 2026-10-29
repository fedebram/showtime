package server

import (
	"log/slog"
	"net/http"

	"github.com/fedebram/showtime/internal/httpjson"
)

// writeError maps the notification errors to http status codes.
func writeError(w http.ResponseWriter, err error) {
	// no client errors for now. So just 500
	slog.Error("internal error", "err", err)
	httpjson.WriteErrors(w, http.StatusInternalServerError, "internal error")
}
