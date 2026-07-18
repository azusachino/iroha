package httpapi

import (
	"log/slog"
	"net"
	"net/http"
	"time"

	imports "github.com/azusachino/iroha/apps/iroha-imports"
	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
	"github.com/azusachino/iroha/apps/iroha-runtime/config"
	"github.com/azusachino/iroha/apps/iroha-runtime/rawfiles"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/briefing"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/daily"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/media"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/sleep"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
)

// Per-IP request budgets (per minute). Keyed off middleware.RealIP. The
// geocode proxy is stricter because each hit fans out to Nominatim, which
// enforces its own ~1 req/s policy.
const (
	apiRateLimitPerMin     = 120
	publicRateLimitPerMin  = 60
	geocodeRateLimitPerMin = 10
	// Local dev (LocalNoAuth) is single-user and does history-wide sweeps, so
	// the per-IP budgets are lifted well clear of normal browsing. The geocode
	// limit is NOT lifted — it protects the external Nominatim service, not us.
	localRateLimitPerMin = 6000
)

type Dependencies struct {
	Config           config.Config
	Logger           *slog.Logger
	ActivityService  *activities.Service
	SleepService     *sleep.Service
	DailyService     *daily.Service
	MediaService     *media.Service
	BriefingRegistry *briefing.Registry
	ImportService    *imports.Service
	RawFileService   *rawfiles.Service
	Cache            *cache.Client
	MaxUploadBytes   int64
	AllowedOrigins   []string
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
	if server.deps.BriefingRegistry == nil {
		server.deps.BriefingRegistry, _ = briefing.NewRegistry()
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

	apiLimit, publicLimit := apiRateLimitPerMin, publicRateLimitPerMin
	if s.deps.Config.Auth.LocalNoAuth {
		apiLimit, publicLimit = localRateLimitPerMin, localRateLimitPerMin
	}

	s.mux.Get("/healthz", s.handleHealthz)
	s.mux.Route("/api/v1", func(r chi.Router) {
		// Private API: CORS limited to configured origins.
		r.Use(corsMiddleware(s.deps.AllowedOrigins))
		r.Use(limitByIP(apiLimit))
		r.Get("/briefing", s.handleBriefing)
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
		r.Route("/sleep", func(r chi.Router) {
			r.Get("/", s.handleListSleep)
			r.Get("/aggregates", s.handleSleepAggregates)
			r.Get("/{sleepId}/segments", s.handleGetSleepSegments)
		})
		r.Route("/daily", func(r chi.Router) {
			r.Get("/", s.handleListDaily)
			r.Get("/aggregates", s.handleDailyAggregates)
		})
		r.Route("/media", func(r chi.Router) {
			r.Get("/aggregates", s.handleMediaAggregates)
			r.Get("/events", s.handleListMediaEvents)
			r.Get("/", s.handleListMedia)
			r.Get("/{mediaId}", s.handleGetMedia)
		})
	})

	// Public, sanitized, cache-backed views for the public page. No auth, and
	// CORS open to any origin since the data is already sanitized.
	s.mux.Route("/public/v1", func(r chi.Router) {
		r.Use(corsMiddleware([]string{"*"}))
		r.Use(limitByIP(publicLimit))
		r.Get("/summary", s.handlePublicSummary)
		r.Get("/activities", s.handlePublicActivities)
		r.Get("/routes", s.handlePublicRoutes)
		r.With(limitByIP(geocodeRateLimitPerMin)).Get("/geocode", s.handlePublicGeocode)
	})
}

// limitByIP builds a per-IP rate limiter (per minute). It keys off the client
// address resolved by middleware.RealIP into r.RemoteAddr, stated explicitly to
// avoid the deprecated LimitByIP helper.
func limitByIP(perMinute int) func(http.Handler) http.Handler {
	return httprate.LimitBy(perMinute, time.Minute, keyByRemoteIP)
}

func keyByRemoteIP(r *http.Request) (string, error) {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host, nil
	}
	return r.RemoteAddr, nil
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
