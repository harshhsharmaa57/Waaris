package health

import (
	"encoding/json"
	"net/http"
)

// Handler exposes only infrastructure health. Domain endpoints belong to services.
type Handler struct {
	service string
}

func NewHandler(service string) Handler {
	return Handler{service: service}
}

func (h Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.respond)
	mux.HandleFunc("GET /readyz", h.respond)
}

func (h Handler) respond(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": h.service})
}
