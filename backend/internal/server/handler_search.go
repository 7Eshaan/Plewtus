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
		// 1. Authenticate first so unauthenticated callers always get 401.
		if s.usageMgr != nil {
			if err := s.usageMgr.Authenticate(r); err != nil {
				WriteJSON(w, http.StatusUnauthorized, model.ErrorResponse{
					Error: "unauthorized", Message: err.Error(), Code: http.StatusUnauthorized,
				})
				return
			}
		}

		// 2. Decode and validate the payload before charging rate limit/quota,
		// so malformed requests do not consume the caller's allowance.
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

		// 3. Enforce rate limits & quotas
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

				if errors.Is(err, usage.ErrRateLimitExceeded) || errors.Is(err, usage.ErrQuotaExceeded) {
					code := "rate_limit_exceeded"
					if errors.Is(err, usage.ErrQuotaExceeded) {
						code = "quota_exceeded"
					}
					if checkRes != nil && checkRes.RetryAfterSec > 0 {
						w.Header().Set("Retry-After", strconv.Itoa(checkRes.RetryAfterSec))
					}
					WriteJSON(w, http.StatusTooManyRequests, model.ErrorResponse{
						Error:   code,
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

		// 4. Execute search
		resp, err := s.searcher.Search(r.Context(), &req)
		if err != nil {
			if errors.Is(err, model.ErrInvalidRequest) {
				WriteJSON(w, http.StatusBadRequest, model.ErrorResponse{
					Error:   "validation_error",
					Message: err.Error(),
					Code:    http.StatusBadRequest,
				})
				return
			}
			if errors.Is(err, context.DeadlineExceeded) {
				WriteJSON(w, http.StatusGatewayTimeout, model.ErrorResponse{
					Error:   "request_timeout",
					Message: "Search timed out: " + err.Error(),
					Code:    http.StatusGatewayTimeout,
				})
				return
			}
			if errors.Is(err, context.Canceled) {
				WriteJSON(w, http.StatusRequestTimeout, model.ErrorResponse{
					Error:   "request_cancelled",
					Message: "Request was cancelled: " + err.Error(),
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
