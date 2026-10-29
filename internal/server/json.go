package server

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}

// decodeJSON reads a single JSON object from the request body into v.
// If the body is malformed it answers 400 and returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	const maxRequestBodyBytes = 1_000_000 // 1 MB
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodyBytes))
	decoder.DisallowUnknownFields()

	// an empty body is fine here (EOF): validation then says what's missing
	if err := decoder.Decode(v); err != nil && !errors.Is(err, io.EOF) {
		writeErrors(w, http.StatusBadRequest, "invalid request body")
		return false
	}

	// we reject additional json
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeErrors(w, http.StatusBadRequest, "expected a single JSON object")
		return false
	}
	return true
}
