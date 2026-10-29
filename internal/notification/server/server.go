package server

import (
	"net/http"

	"github.com/fedebram/showtime/internal/notification"
)

type Server struct {
	notifications *notification.Service
	mux           *http.ServeMux
}

func NewServer(notifications *notification.Service) *Server {
	s := &Server{notifications: notifications, mux: http.NewServeMux()}

	s.mux.HandleFunc("POST /notifications", s.sendNotification)

	s.mux.HandleFunc("GET /notifications", s.listNotifications)

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
