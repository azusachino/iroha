package httpapi

import (
	"log/slog"
	"net"
	"net/http"
	"time"

	imports "github.com/azusachino/iroha/apps/iroha-imports"
	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
	"github.com/azusachino/iroha/apps/iroha-runtime/config"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/briefing"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/daily"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/geocode"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/media"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/rawfiles"
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
	GeocodeService   *geocode.Service
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
	s.mux.Use(requestIDResponseHeader)
	s.mux.Use(middleware.RealIP)
	s.mux.Use(middleware.Recoverer)
	s.mux.Use(s.accessLog)

	apiLimit, publicLimit := apiRateLimitPerMin, publicRateLimitPerMin
	if s.deps.Config.Auth.LocalNoAuth {
		apiLimit, publicLimit = localRateLimitPerMin, localRateLimitPerMin
	}

	s.mux.Get("/healthz", s.handleHealthz)
	s.mux.Route("/api/v1", func(r chi.Router) {
		// Private API: CORS limited to configured origins.
		r.Use(corsMiddleware(s.deps.AllowedOrigins))
		// Rate limiting runs before auth so malformed or missing-token floods are
		// bounded too. Valid JWTs are still keyed by subject when available.
		r.Use(limitByIdentity(apiLimit, s.deps.Config.Auth))
		r.Use(s.requireJWT("iroha:read"))
		r.Get("/briefing", s.handleBriefing)
		r.Route("/raw-files", func(r chi.Router) {
			r.With(s.requireJWT("iroha:write")).Post("/", s.handleCreateRawFile)
			r.Get("/", s.handleListRawFiles)
			r.Get("/{rawFileId}", s.handleGetRawFile)
		})
		r.Route("/imports", func(r chi.Router) {
			r.With(s.requireJWT("iroha:write")).Post("/", s.handleCreateImportJob)
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
	return httprate.LimitBy(perMinute, time.Minute, keyByRemoteIP, httprate.WithLimitHandler(rateLimitResponse))
}

func limitByIdentity(perMinute int, auth config.AuthConfig) func(http.Handler) http.Handler {
	return httprate.LimitBy(perMinute, time.Minute, func(r *http.Request) (string, error) {
		if claims, err := parseJWT(r, auth); err == nil && claims.Subject != "" {
			return "subject:" + claims.Subject, nil
		}
		if subject := authSubject(r); subject != "" {
			return "subject:" + subject, nil
		}
		return keyByRemoteIP(r)
	}, httprate.WithLimitHandler(rateLimitResponse))
}

func requestIDResponseHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestID := middleware.GetReqID(r.Context()); requestID != "" {
			w.Header().Set("X-Request-ID", requestID)
		}
		next.ServeHTTP(w, r)
	})
}

func rateLimitResponse(w http.ResponseWriter, _ *http.Request) {
	writeContractError(w, http.StatusTooManyRequests, "rate_limited", "rate limit exceeded")
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
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders: []string{"Retry-After", "X-Request-ID"},
		MaxAge:         300,
	})
}

func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		started := time.Now()
		next.ServeHTTP(wrapped, r)

		route := "unknown"
		if routeContext := chi.RouteContext(r.Context()); routeContext != nil && routeContext.RoutePattern() != "" {
			route = routeContext.RoutePattern()
		}
		s.deps.Logger.InfoContext(r.Context(), "http request",
			"request_id", middleware.GetReqID(r.Context()),
			"method", r.Method,
			"route", route,
			"status", wrapped.Status(),
			"bytes", wrapped.BytesWritten(),
			"duration_ms", time.Since(started).Milliseconds(),
			"subject", authSubject(r),
		)
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
