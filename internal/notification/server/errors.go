package server

import (
	"log"
	"net/http"

	"github.com/fedebram/showtime/internal/httpjson"
)

// writeError maps the notification errors to http status codes.
func writeError(w http.ResponseWriter, err error) {
	// no client errors for now. So just 500
	log.Printf("internal error: %v", err)
	httpjson.WriteErrors(w, http.StatusInternalServerError, "internal error")
}
