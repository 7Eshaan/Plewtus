package server

import (
	"encoding/json"
	"net/http"
	"time"

	"plewtus/internal/model"
)

// handleSearch handles the POST /search endpoint.
func (s *Server) handleSearch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		var req model.SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteJSON(w, http.StatusBadRequest, model.ErrorResponse{
				Error:   "invalid_json",
				Message: "Malformed JSON payload: " + err.Error(),
				Code:    http.StatusBadRequest,
			})
			return
		}

		req.Normalize(s.cfg.DefaultLimit, s.cfg.MaxLimit, s.cfg.DefaultProviders)
		if err := req.Validate(); err != nil {
			WriteJSON(w, http.StatusBadRequest, model.ErrorResponse{
				Error:   "validation_error",
				Message: err.Error(),
				Code:    http.StatusBadRequest,
			})
			return
		}

		// Phase 1: Return the established SearchResponse contract.
		// In subsequent phases, this will be served via the search orchestrator.
		resp := model.NewSearchResponse(req.Query)
		resp.Metadata.TookMs = time.Since(start).Milliseconds()
		resp.Metadata.Providers.Queried = req.Providers

		WriteJSON(w, http.StatusOK, resp)
	}
}
