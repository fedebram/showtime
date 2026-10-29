package server

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/fedebram/showtime/internal/httpjson"
	"github.com/fedebram/showtime/internal/notification"
)

// SendRequest is a message to send. sent_at is not here:
// the service sets it when it sends.
type SendRequest struct {
	ID        uuid.UUID `json:"id"`
	Kind      string    `json:"kind"`
	BookingID uuid.UUID `json:"booking_id"`
}

// problems returns every problem of r.
func (r SendRequest) problems() []string {
	var problems []string
	if r.ID == uuid.Nil {
		problems = append(problems, "id is required")
	}
	if r.Kind == "" {
		problems = append(problems, "kind is required")
	}
	if r.BookingID == uuid.Nil {
		problems = append(problems, "booking_id is required")
	}
	return problems
}

func (s *Server) sendNotification(w http.ResponseWriter, r *http.Request) {
	var req SendRequest
	if !httpjson.DecodeJSON(w, r, &req) {
		return
	}
	if problems := req.problems(); len(problems) > 0 {
		httpjson.WriteErrors(w, http.StatusBadRequest, problems...)
		return
	}

	err := s.notifications.Send(r.Context(), notification.Message{
		ID:        req.ID,
		Kind:      req.Kind,
		BookingID: req.BookingID,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

const (
	defaultListLimit = 50
	maxListLimit     = 100
)

func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request) {
	limit := defaultListLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxListLimit {
			httpjson.WriteErrors(w, http.StatusBadRequest, "limit must be between 1 and 100")
			return
		}
		limit = n
	}

	msgs, err := s.notifications.List(r.Context(), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, msgs)
}
