package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/azusachino/iroha/apps/iroha-server/internal/config"
	"github.com/azusachino/iroha/apps/iroha-server/internal/rawfiles"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Dependencies struct {
	Config         config.Config
	Logger         *slog.Logger
	RawFileService *rawfiles.Service
	MaxUploadBytes int64
	AllowedOrigins []string
}

type Server struct {
	deps Dependencies
	mux  chi.Router
}

func NewServer(deps Dependencies) http.Handler {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	if deps.MaxUploadBytes == 0 {
		deps.MaxUploadBytes = 2 << 30
	}

	server := &Server{
		deps: deps,
		mux:  chi.NewRouter(),
	}
	server.routes()
	return server
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.Use(middleware.RequestID)
	s.mux.Use(middleware.RealIP)
	s.mux.Use(middleware.Recoverer)

	s.mux.Get("/healthz", s.handleHealthz)
	s.mux.Route("/api/v1", func(r chi.Router) {
		r.Route("/raw-files", func(r chi.Router) {
			r.Post("/", s.handleCreateRawFile)
			r.Get("/", s.handleListRawFiles)
			r.Get("/{rawFileId}", s.handleGetRawFile)
		})
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
