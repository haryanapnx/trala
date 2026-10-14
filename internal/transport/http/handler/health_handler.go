package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/haryanapnx/trala/internal/transport/http/response"
)

// Pinger is the minimal dependency the health handler needs, so the transport
// layer does not import the database driver directly.
type Pinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	db Pinger
}

func NewHealthHandler(db Pinger) *HealthHandler {
	return &HealthHandler{db: db}
}

// Live is a liveness probe: the process is up.
func (h *HealthHandler) Live(w http.ResponseWriter, _ *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready is a readiness probe: the process can reach its dependencies.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		response.JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
