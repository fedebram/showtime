package server

import (
	"net/http"

	"github.com/fedebram/showtime/internal/store"
)

type Server struct {
	store *store.MemoryStore
	mux   *http.ServeMux
}

func NewServer(memory *store.MemoryStore) *Server {
	s := &Server{store: memory, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /shows", s.listShows)
	s.mux.HandleFunc("GET /shows/{id}", s.getShow)
	return s
}

// we need serve http to satisfy the handler interface

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
