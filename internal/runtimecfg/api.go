package runtimecfg

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Handler serves the config CRUD API.
type Handler struct {
	store *Store
}

// NewHandler creates a config API handler.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// RegisterRoutes registers /admin/api/config routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, wrap func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("/admin/api/config", wrap(h.handleAll))
	mux.HandleFunc("/admin/api/config/", wrap(h.handleByKey))
}

// handleAll handles GET (list) and POST (create/update) on /api/config
func (h *Handler) handleAll(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.list(w, r)
	case http.MethodPost:
		h.upsert(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleByKey handles GET, PUT, DELETE on /api/config/{key}
func (h *Handler) handleByKey(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/admin/api/config/")
	if key == "" {
		h.handleAll(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.get(w, key)
	case http.MethodPut:
		h.put(w, r, key)
	case http.MethodDelete:
		h.del(w, key)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) list(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.store.List())
}

func (h *Handler) get(w http.ResponseWriter, key string) {
	e, ok := h.store.Get(key)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "key not found"})
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *Handler) upsert(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key and value required"})
		return
	}
	e := h.store.Set(body.Key, body.Value)
	writeJSON(w, http.StatusOK, e)
}

func (h *Handler) put(w http.ResponseWriter, r *http.Request, key string) {
	var body struct {
		Value interface{} `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "value required"})
		return
	}
	e := h.store.Set(key, body.Value)
	writeJSON(w, http.StatusOK, e)
}

func (h *Handler) del(w http.ResponseWriter, key string) {
	if !h.store.Delete(key) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "key not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
