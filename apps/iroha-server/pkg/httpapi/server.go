package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/cache"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/config"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/imports"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/rawfiles"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
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
		// Private API: CORS limited to configured origins.
		r.Use(corsMiddleware(s.deps.AllowedOrigins))
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

	// Public, sanitized, cache-backed views for the public page. No auth, and
	// CORS open to any origin since the data is already sanitized.
	s.mux.Route("/public/v1", func(r chi.Router) {
		r.Use(corsMiddleware([]string{"*"}))
		r.Get("/summary", s.handlePublicSummary)
		r.Get("/activities", s.handlePublicActivities)
		r.Get("/routes", s.handlePublicRoutes)
	})
}

// corsMiddleware builds a read-only CORS handler for the given origins.
func corsMiddleware(origins []string) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins: origins,
		AllowedMethods: []string{http.MethodGet, http.MethodOptions},
		AllowedHeaders: []string{"Accept", "Content-Type"},
		MaxAge:         300,
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
