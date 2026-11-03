package server

import (
	"net/http"
	"time"

	"plewtus/internal/model"
)

// handleHealth returns the service health status, version, and uptime.
func (s *Server) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uptime := time.Since(s.startTime).Seconds()
		resp := model.HealthResponse{
			Status:        "ok",
			Version:       s.cfg.Version,
			UptimeSeconds: uptime,
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
		}

		WriteJSON(w, http.StatusOK, resp)
	}
}
