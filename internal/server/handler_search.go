package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"plewtus/internal/model"
)

// handleSearch handles the POST /search endpoint.
func (s *Server) handleSearch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		resp, err := s.searcher.Search(r.Context(), &req)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				WriteJSON(w, http.StatusRequestTimeout, model.ErrorResponse{
					Error:   "request_timeout",
					Message: "Request timed out or was cancelled: " + err.Error(),
					Code:    http.StatusRequestTimeout,
				})
				return
			}
			WriteJSON(w, http.StatusInternalServerError, model.ErrorResponse{
				Error:   "search_error",
				Message: err.Error(),
				Code:    http.StatusInternalServerError,
			})
			return
		}

		WriteJSON(w, http.StatusOK, resp)
	}
}
