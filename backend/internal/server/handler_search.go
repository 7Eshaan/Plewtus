package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"plewtus/internal/model"
	"plewtus/internal/usage"
)

// handleSearch handles the POST /search endpoint.
func (s *Server) handleSearch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Authenticate and enforce rate limits & quotas
		if s.usageMgr != nil {
			checkRes, err := s.usageMgr.Check(r)
			if checkRes != nil {
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(checkRes.RateLimitLimit))
				w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(checkRes.RateLimitRemaining))
				w.Header().Set("X-RateLimit-Reset", strconv.Itoa(checkRes.RateLimitResetSec))
				w.Header().Set("X-Quota-Limit", strconv.Itoa(checkRes.QuotaLimit))
				w.Header().Set("X-Quota-Remaining", strconv.Itoa(checkRes.QuotaRemaining))
				w.Header().Set("X-Quota-Reset", strconv.Itoa(checkRes.QuotaResetSec))
			}

			if err != nil {
				if errors.Is(err, usage.ErrAuthRequired) || errors.Is(err, usage.ErrInvalidAPIKey) {
					WriteJSON(w, http.StatusUnauthorized, model.ErrorResponse{
						Error:   "unauthorized",
						Message: err.Error(),
						Code:    http.StatusUnauthorized,
					})
					return
				}

				if errors.Is(err, usage.ErrRateLimitExceeded) {
					if checkRes != nil && checkRes.RetryAfterSec > 0 {
						w.Header().Set("Retry-After", strconv.Itoa(checkRes.RetryAfterSec))
					}
					WriteJSON(w, http.StatusTooManyRequests, model.ErrorResponse{
						Error:   "rate_limit_exceeded",
						Message: err.Error(),
						Code:    http.StatusTooManyRequests,
					})
					return
				}

				if errors.Is(err, usage.ErrQuotaExceeded) {
					if checkRes != nil && checkRes.RetryAfterSec > 0 {
						w.Header().Set("Retry-After", strconv.Itoa(checkRes.RetryAfterSec))
					}
					WriteJSON(w, http.StatusTooManyRequests, model.ErrorResponse{
						Error:   "quota_exceeded",
						Message: err.Error(),
						Code:    http.StatusTooManyRequests,
					})
					return
				}

				WriteJSON(w, http.StatusInternalServerError, model.ErrorResponse{
					Error:   "usage_check_error",
					Message: err.Error(),
					Code:    http.StatusInternalServerError,
				})
				return
			}
		}

		// 2. Decode and validate request payload
		var req model.SearchRequest
		const maxSearchRequestBytes = 1 << 20
		r.Body = http.MaxBytesReader(w, r.Body, maxSearchRequestBytes)
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&req); err != nil {
			WriteJSON(w, http.StatusBadRequest, model.ErrorResponse{
				Error:   "invalid_json",
				Message: "Malformed JSON payload: " + err.Error(),
				Code:    http.StatusBadRequest,
			})
			return
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			WriteJSON(w, http.StatusBadRequest, model.ErrorResponse{
				Error: "invalid_json", Message: "Request body must contain exactly one JSON value.", Code: http.StatusBadRequest,
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

		// 3. Execute search
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
