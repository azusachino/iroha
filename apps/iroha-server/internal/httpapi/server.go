package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/azusachino/iroha/apps/iroha-server/internal/activities"
	"github.com/azusachino/iroha/apps/iroha-server/internal/cache"
	"github.com/azusachino/iroha/apps/iroha-server/internal/config"
	"github.com/azusachino/iroha/apps/iroha-server/internal/imports"
	"github.com/azusachino/iroha/apps/iroha-server/internal/rawfiles"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Dependencies struct {
	Config          config.Config
	Logger          *slog.Logger
	ActivityService *activities.Service
	ImportService   *imports.Service
	RawFileService  *rawfiles.Service
	Cache           *cache.Client
	MaxUploadBytes  int64
	AllowedOrigins  []string
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
			r.With(s.requireUploadAuth).Post("/", s.handleCreateRawFile)
			r.Get("/", s.handleListRawFiles)
			r.Get("/{rawFileId}", s.handleGetRawFile)
		})
		r.Route("/imports", func(r chi.Router) {
			r.With(s.requireUploadAuth).Post("/", s.handleCreateImportJob)
			r.Get("/", s.handleListImportJobs)
			r.Get("/{importId}", s.handleGetImportJob)
		})
		r.Route("/activities", func(r chi.Router) {
			r.Get("/", s.handleListActivities)
			r.Get("/{activityId}", s.handleGetActivity)
			r.Get("/{activityId}/route", s.handleGetActivityRoute)
			r.Get("/{activityId}/samplings", s.handleGetActivitySamplings)
			r.Get("/{activityId}/laps", s.handleGetActivityLaps)
		})
	})

	// Public, sanitized, cache-backed views for the public page. No auth.
	s.mux.Route("/public/v1", func(r chi.Router) {
		r.Get("/summary", s.handlePublicSummary)
		r.Get("/activities", s.handlePublicActivities)
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
