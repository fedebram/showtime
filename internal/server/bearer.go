package server

import (
	"net/http"
	"strings"
)

// bearerToken returns the secret from "Authorization: Bearer <secret>".
func bearerToken(r *http.Request) (string, bool) {
	// it is out of the scope of this exercise to adhere strictly to http rfc 9110
	secret, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	return secret, ok && secret != ""
}
