package show

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	// waste space, but the json then is more readable
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(h.store.List()); err != nil {
		log.Printf("encode shows: %v", err)
	}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		http.Error(w, "invalid show ID", http.StatusBadRequest)
		return
	}

	show, ok := h.store.Get(id)
	if !ok {
		http.Error(w, "show not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(show); err != nil {
		log.Printf("encode show: %v", err)
	}
}
