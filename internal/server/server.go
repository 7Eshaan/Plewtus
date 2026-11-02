package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"plewtus/internal/config"
	"plewtus/internal/model"
)

// Searcher defines the interface for orchestrating search requests.
type Searcher interface {
	Search(ctx context.Context, req *model.SearchRequest) (*model.SearchResponse, error)
}

// Server coordinates HTTP routing, middleware, and lifecycle.
type Server struct {
	cfg       *config.Config
	logger    *slog.Logger
	searcher  Searcher
	startTime time.Time
	mux       *http.ServeMux
	server    *http.Server
}

// New creates and configures a new Server instance.
func New(cfg *config.Config, logger *slog.Logger, searcher Searcher) *Server {
	s := &Server{
		cfg:       cfg,
		logger:    logger,
		searcher:  searcher,
		startTime: time.Now(),
		mux:       http.NewServeMux(),
	}

	s.setupRoutes()
	return s
}

// setupRoutes registers all HTTP endpoints.
func (s *Server) setupRoutes() {
	s.mux.HandleFunc("GET /health", s.handleHealth())
	s.mux.HandleFunc("GET /healthz", s.handleHealth())
	s.mux.HandleFunc("POST /search", s.handleSearch())
}

// Handler returns the HTTP handler wrapped with core middleware.
func (s *Server) Handler() http.Handler {
	handler := http.Handler(s.mux)
	handler = CORSMiddleware(handler)
	handler = LoggingMiddleware(s.logger, handler)
	handler = RecoveryMiddleware(s.logger, handler)
	return handler
}

// Start begins listening and serving HTTP requests.
func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:         s.cfg.Addr(),
		Handler:      s.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("plewtus server starting",
		slog.String("addr", s.cfg.Addr()),
		slog.String("env", s.cfg.Env),
		slog.String("version", s.cfg.Version),
	)

	err := s.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down server")
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}
