package web

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/yardenshoham/onepiece/internal/web/pages"
	"github.com/yardenshoham/onepiece/pkg/poller"
)

//go:embed static
var staticFiles embed.FS

// Config controls optional server features.
type Config struct {
	PostHogAPIKey string
	PostHogHost   string
}

// Server is the HTTP server for the One Piece tracker dashboard.
type Server struct {
	logger    *slog.Logger
	poller    *poller.Poller
	mux       *http.ServeMux
	analytics pages.AnalyticsConfig
}

// NewServer creates an HTTP server with all routes registered.
func NewServer(logger *slog.Logger, p *poller.Poller, config Config) *Server {
	s := &Server{
		logger: logger,
		poller: p,
		mux:    http.NewServeMux(),
		analytics: pages.AnalyticsConfig{
			PostHogAPIKey: config.PostHogAPIKey,
			PostHogHost:   config.PostHogHost,
		},
	}

	s.mux.Handle("GET /static/", http.FileServerFS(staticFiles))
	s.mux.HandleFunc("GET /{$}", s.handleDashboard)
	s.mux.HandleFunc("GET /about", s.handleAbout)
	s.mux.HandleFunc("GET /health", s.handleHealth)

	return s
}

// ListenAndServe starts the server. Blocks until ctx is cancelled, then shuts down gracefully.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.loggingMiddleware(s.recoveryMiddleware(s.mux)),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("server starting", "addr", addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		s.logger.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func (s *Server) handleDashboard(w http.ResponseWriter, _ *http.Request) {
	d := s.poller.Dashboard()
	if d == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := pages.LoadingPage(s.analytics).Render(w); err != nil {
			s.logger.Error("rendering loading page", "error", err)
		}
		return
	}

	if d.EpisodesWatched == 0 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := pages.Layout("One Piece Tracker", "/", 7200, s.analytics).Render(w); err != nil {
			s.logger.Error("rendering empty dashboard", "error", err)
		}
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.DashboardPage(d, s.analytics).Render(w); err != nil {
		s.logger.Error("rendering dashboard", "error", err)
	}
}

func (s *Server) handleAbout(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.AboutPage(s.analytics).Render(w); err != nil {
		s.logger.Error("rendering about page", "error", err)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	if s.poller.Dashboard() == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, "not ready")
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		s.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration", time.Since(start).Round(time.Microsecond),
		)
	})
}

func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				s.logger.Error("panic recovered", "error", err, "path", r.URL.Path)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
