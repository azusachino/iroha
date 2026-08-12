package httpapi

import (
	"bytes"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	imports "github.com/azusachino/iroha/apps/iroha-imports"
	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
	"github.com/azusachino/iroha/apps/iroha-runtime/config"
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/rawfiles"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/briefing"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/daily"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/expenses"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/geocode"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/media"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/mediaresolution"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/metrics"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/sleep"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/tasks"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
)

// apiRateLimitPerMin is the per-peer request budget (per minute). The private
// API is unauthenticated by design: iroha is a single-user personal
// deployment (NAS/private network), and access control is the network
// boundary, not an application-level credential. The budget is lifted well
// clear of normal browsing/history-wide sweeps accordingly.
const apiRateLimitPerMin = 6000

const readCacheTTL = 24 * time.Hour

type Dependencies struct {
	Config                 config.Config
	Logger                 *slog.Logger
	ActivityService        *activities.Service
	SleepService           *sleep.Service
	DailyService           *daily.Service
	ExpenseService         *expenses.Service
	MediaService           *media.Service
	MediaResolutionService *mediaresolution.Service
	MetricRegistry         *metrics.Registry
	BriefingRegistry       *briefing.Registry
	ImportService          *imports.Service
	RawFileService         *rawfiles.Service
	Cache                  *cache.Client
	GeocodeService         *geocode.Service
	JobEnqueuer            imports.Enqueuer
	JobsService            *jobs.Service
	TaskService            *tasks.Service
	MaxUploadBytes         int64
	AllowedOrigins         []string
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
	if server.deps.MetricRegistry == nil {
		server.deps.MetricRegistry, _ = metrics.DefaultRegistry()
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
	s.mux.Use(middleware.Recoverer)
	s.mux.Use(s.accessLog)

	s.mux.Get("/healthz", s.handleHealthz)
	s.mux.Route("/api/v1", func(r chi.Router) {
		// Private API: CORS limited to configured origins. Unauthenticated —
		// see the rate-limit budget comment above for why.
		r.Use(corsMiddleware(s.deps.AllowedOrigins))
		r.Use(limitByIP(apiRateLimitPerMin))
		r.Use(s.readCache)
		r.Get("/briefing", s.handleBriefing)
		r.Get("/metrics", s.handleListMetrics)
		r.Get("/metrics/{metricId}", s.handleGetMetric)
		r.Route("/raw-files", func(r chi.Router) {
			r.Post("/", s.handleCreateRawFile)
			r.Get("/", s.handleListRawFiles)
			r.Get("/{rawFileId}", s.handleGetRawFile)
		})
		r.Route("/imports", func(r chi.Router) {
			r.Post("/", s.handleCreateImportJob)
			r.Get("/", s.handleListImportJobs)
			r.Get("/{importId}", s.handleGetImportJob)
		})
		r.Route("/activities", func(r chi.Router) {
			r.Get("/", s.handleListActivities)
			r.Get("/summary", s.handleActivitySummary)
			r.Get("/routes", s.handleActivityRoutes)
			r.Get("/{activityId}", s.handleGetActivity)
			r.Get("/{activityId}/route", s.handleGetActivityRoute)
			r.Get("/{activityId}/samplings", s.handleGetActivitySamplings)
			r.Get("/{activityId}/laps", s.handleGetActivityLaps)
		})
		r.Route("/sleep", func(r chi.Router) {
			r.Get("/", s.handleListSleep)
			r.Get("/aggregates", s.handleSleepAggregates)
			r.Get("/{sleepId}", s.handleGetSleep)
			r.Get("/{sleepId}/segments", s.handleGetSleepSegments)
		})
		r.Route("/daily", func(r chi.Router) {
			r.Get("/", s.handleListDaily)
			r.Get("/aggregates", s.handleDailyAggregates)
		})
		r.Route("/expenses", func(r chi.Router) {
			r.Post("/", s.handleCreateExpense)
			r.Get("/", s.handleListExpenses)
			r.Get("/{expenseId}", s.handleGetExpense)
			r.Put("/{expenseId}", s.handleReplaceExpense)
			r.Delete("/{expenseId}", s.handleDeleteExpense)
		})
		r.Route("/reports", func(r chi.Router) {
			r.Get("/monthly", s.handleMonthlyReport)
		})
		r.Route("/media", func(r chi.Router) {
			r.Post("/sync/{connectorId}", s.handleEnqueueMediaSync)
			r.Get("/aggregates", s.handleMediaAggregates)
			r.Get("/events", s.handleListMediaEvents)
			r.Get("/resolution-tasks", s.handleListMediaResolutionTasks)
			r.Patch("/resolution-tasks/{taskId}", s.handleUpdateMediaResolutionTask)
			r.Get("/", s.handleListMedia)
			r.Get("/{mediaId}", s.handleGetMedia)
		})
		r.Route("/tasks", func(r chi.Router) {
			r.Get("/", s.handleListTasks)
			r.Post("/", s.handleCreateTask)
			r.Patch("/{taskId}", s.handleUpdateTask)
		})
		r.Route("/jobs", func(r chi.Router) {
			r.Get("/", s.handleListJobs)
			r.Get("/{jobId}", s.handleGetJob)
		})
		r.Post("/actions/{action}", s.handleAction)
	})
}

// limitByIP builds a per-peer rate limiter (per minute). It intentionally keys
// off r.RemoteAddr instead of forwarded headers, which are not trusted here.
func limitByIP(perMinute int) func(http.Handler) http.Handler {
	return httprate.LimitBy(perMinute, time.Minute, keyByRemoteIP, httprate.WithLimitHandler(rateLimitResponse))
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
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowedHeaders: []string{"Accept", "Content-Type"},
		ExposedHeaders: []string{"Retry-After", "X-Request-ID", "X-Iroha-Cache"},
		MaxAge:         300,
	})
}

// readCache caches successful JSON reads over the imported, single-user data.
// The import pipeline advances each namespace generation after a successful
// write, so a long TTL is only a safety net for data that is otherwise static.
func (s *Server) readCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		namespace, ok := readCacheNamespace(r)
		if !ok || s.deps.Cache == nil {
			next.ServeHTTP(w, r)
			return
		}

		key := readCacheKey(r)
		if body, ok := cache.Get[[]byte](r.Context(), s.deps.Cache, namespace, key); ok {
			w.Header().Set("X-Iroha-Cache", "HIT")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
			return
		}

		w.Header().Set("X-Iroha-Cache", "MISS")
		wrapped := &readCacheResponseWriter{ResponseWriter: w}
		next.ServeHTTP(wrapped, r)
		if wrapped.status != http.StatusOK || wrapped.body.Len() == 0 || !isJSONContentType(wrapped.Header().Get("Content-Type")) {
			return
		}
		cache.Set(r.Context(), s.deps.Cache, namespace, key, readCacheTTL, wrapped.body.Bytes())
	})
}

type readCacheResponseWriter struct {
	http.ResponseWriter
	body   bytes.Buffer
	status int
}

func (w *readCacheResponseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *readCacheResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if w.status == http.StatusOK {
		_, _ = w.body.Write(body)
	}
	return w.ResponseWriter.Write(body)
}

func readCacheNamespace(r *http.Request) (string, bool) {
	if r.Method != http.MethodGet {
		return "", false
	}
	if r.URL.Path == "/api/v1/media/sync" || strings.HasPrefix(r.URL.Path, "/api/v1/media/sync/") {
		return "", false
	}
	if r.URL.Path == "/api/v1/expenses" || strings.HasPrefix(r.URL.Path, "/api/v1/expenses/") ||
		r.URL.Path == "/api/v1/reports/monthly" {
		return "", false
	}
	for prefix, namespace := range map[string]string{
		"/api/v1/activities": cache.NamespaceActivities,
		"/api/v1/briefing":   cache.NamespaceBriefing,
		"/api/v1/daily":      cache.NamespaceDaily,
		"/api/v1/media":      cache.NamespaceMedia,
		"/api/v1/sleep":      cache.NamespaceSleep,
	} {
		if r.URL.Path == prefix || strings.HasPrefix(r.URL.Path, prefix+"/") {
			return namespace, true
		}
	}
	return "", false
}

func readCacheKey(r *http.Request) string {
	key := r.Method + " " + r.URL.Path
	if query := r.URL.Query().Encode(); query != "" {
		key += "?" + query
	}
	return key
}

func isJSONContentType(value string) bool {
	return strings.HasPrefix(strings.ToLower(value), "application/json")
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
		)
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
