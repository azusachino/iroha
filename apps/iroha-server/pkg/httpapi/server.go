package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	imports "github.com/azusachino/iroha/apps/iroha-imports"
	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
	"github.com/azusachino/iroha/apps/iroha-runtime/config"
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/rawfiles"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/briefing"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/coverage"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/daily"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/expenses"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/geocode"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/media"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/metrics"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/metricseries"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/sleep"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/tasks"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"gorm.io/gorm"
)

// apiRateLimitPerMin is the per-client request budget (per minute) for the
// whole API. It stays well clear of normal browsing and history-wide sweeps;
// the tighter limits below guard the routes an attacker can reach.
const apiRateLimitPerMin = 6000

const (
	// authRateLimitPerMin bounds login and setup attempts per client.
	authRateLimitPerMin = 10
	// intakeRateLimitPerMin bounds intake requests per client, before the
	// credential is checked.
	intakeRateLimitPerMin = 30
)

const (
	readyzTimeout  = 2 * time.Second
	statusReady    = "ready"
	statusNotReady = "not_ready"
)

type Dependencies struct {
	Config              config.Config
	Logger              *slog.Logger
	DB                  *gorm.DB
	ActivityService     *activities.Service
	SleepService        *sleep.Service
	DailyService        *daily.Service
	ExpenseService      *expenses.Service
	MediaService        *media.Service
	MetricRegistry      *metrics.Registry
	MetricSeriesService *metricseries.Service
	BriefingRegistry    *briefing.Registry
	CoverageService     *coverage.Service
	ImportService       *imports.Service
	RawFileService      *rawfiles.Service
	Cache               *cache.Client
	GeocodeService      *geocode.Service
	JobEnqueuer         imports.Enqueuer
	JobsService         *jobs.Service
	TaskService         *tasks.Service
	// HealthIntakeCredentials verifies the HAE intake token; nil fails closed.
	HealthIntakeCredentials HealthIntakeVerifier
	// IntakeCredentialAdmin issues and revokes HAE tokens from the admin page.
	IntakeCredentialAdmin IntakeCredentialAdmin
	// Auth authenticates the owner; nil fails closed on every private route.
	Auth Authenticator
	// Passkeys is the WebAuthn surface; nil or unconfigured answers 503.
	Passkeys PasskeyManager
	// ParserVersion is reported on the admin system page.
	ParserVersion  string
	ReadyCheck     func(context.Context) error
	MaxUploadBytes int64
	AllowedOrigins []string
	Now            func() time.Time
}

type Server struct {
	deps           Dependencies
	mux            chi.Router
	now            func() time.Time
	trustedProxies []netip.Prefix
	intakeQuota    *intakeQuota
	publicCache    *publicSnapshotCache
}

func NewServer(deps Dependencies) http.Handler {
	if deps.Config.Server.Timezone == "" {
		deps.Config.Server.Timezone = config.Default().Server.Timezone
	}
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	if deps.MaxUploadBytes == 0 {
		deps.MaxUploadBytes = 2 << 30
	}

	server := &Server{
		deps: deps,
		mux:  chi.NewRouter(),
		now:  time.Now,
	}
	if deps.Now != nil {
		server.now = deps.Now
	}
	var invalid []string
	server.trustedProxies, invalid = parseTrustedProxies(deps.Config.Server.TrustedProxies)
	for _, value := range invalid {
		deps.Logger.Warn("ignoring invalid trusted proxy CIDR", "value", value)
	}
	server.intakeQuota = newIntakeQuota(server.now)
	server.publicCache = &publicSnapshotCache{}
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
	s.mux.Get("/readyz", s.handleReadyz)
	s.mux.Route("/public/v1", s.publicRoutes)
	s.mux.Route("/api/v1", func(r chi.Router) {
		// Private API: CORS limited to configured origins. Every route below
		// requires an owner session (ADR-0008) except login/setup and the HAE
		// intake endpoint, which authenticates with its own credential.
		r.Use(corsMiddleware(s.deps.AllowedOrigins))
		r.Use(s.limitByClient(apiRateLimitPerMin, time.Minute))
		r.Get("/auth/session", s.handleAuthSession)
		r.With(s.limitByClient(authRateLimitPerMin, time.Minute)).Post("/auth/setup", s.handleAuthSetup)
		r.With(s.limitByClient(authRateLimitPerMin, time.Minute)).Post("/auth/login", s.handleAuthLogin)
		r.With(s.limitByClient(authRateLimitPerMin, time.Minute)).Post("/auth/passkey/begin", s.handleBeginPasskeyLogin)
		r.With(s.limitByClient(authRateLimitPerMin, time.Minute)).Post("/auth/passkey/finish", s.handleFinishPasskeyLogin)
		r.With(
			s.limitByClient(intakeRateLimitPerMin, time.Minute),
			s.requireIntakeCredential,
			s.limitPerIntakeCredential,
		).Post("/intake/health", s.handleHealthIntake)
		r.Group(s.privateRoutes)
	})
}

func (s *Server) privateRoutes(r chi.Router) {
	r.Use(s.requireSession)
	r.Use(s.requireCSRF)
	r.Use(s.rejectFutureReadScope)
	r.Use(s.readCache)
	r.Post("/auth/logout", s.handleAuthLogout)
	r.Patch("/account", s.handleUpdateAccount)
	r.With(s.limitByClient(authRateLimitPerMin, time.Minute)).Post("/account/reauth", s.handleReauth)
	r.Route("/account/passkeys", func(r chi.Router) {
		r.Get("/", s.handleListPasskeys)
		r.Post("/register/begin", s.handleBeginPasskeyRegistration)
		r.Post("/register/finish", s.handleFinishPasskeyRegistration)
		r.Patch("/{passkeyId}", s.handleRenamePasskey)
		r.Delete("/{passkeyId}", s.handleDeletePasskey)
	})
	r.Get("/admin/system", s.handleSystem)
	r.Route("/admin/schedules", func(r chi.Router) {
		r.Get("/", s.handleListSchedules)
		r.Patch("/{kind}", s.handleUpdateSchedule)
		r.Post("/{kind}/run", s.handleRunSchedule)
	})
	r.Route("/admin/intake-credentials", func(r chi.Router) {
		r.Get("/", s.handleListIntakeCredentials)
		r.Post("/", s.handleIssueIntakeCredential)
		r.Delete("/{credentialId}", s.handleRevokeIntakeCredential)
	})
	r.Get("/briefing", s.handleBriefing)
	r.Get("/coverage", s.handleCoverage)
	r.Get("/connections", s.handleListConnections)
	r.Post("/media/matching-decisions", s.handleRecordMatchingDecision)
	r.Get("/metrics", s.handleListMetrics)
	r.Get("/metrics/{metricId}", s.handleGetMetric)
	r.Get("/metrics/{metricId}/series", s.handleMetricSeries)
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
		r.Get("/overview", s.handleActivityOverview)
		r.Get("/summary", s.handleActivitySummary)
		r.Get("/bounds", s.handleActivityBounds)
		r.Get("/routes", s.handleActivityRoutes)
		r.Get("/{activityId}", s.handleGetActivity)
		r.Get("/{activityId}/route", s.handleGetActivityRoute)
		r.Get("/{activityId}/samplings", s.handleGetActivitySamplings)
		r.Get("/{activityId}/laps", s.handleGetActivityLaps)
	})
	r.Route("/sleep", func(r chi.Router) {
		r.Get("/", s.handleListSleep)
		r.Get("/overview", s.handleSleepOverview)
		r.Get("/aggregates", s.handleSleepAggregates)
		r.Get("/bounds", s.handleSleepBounds)
		r.Get("/{sleepId}", s.handleGetSleep)
		r.Get("/{sleepId}/segments", s.handleGetSleepSegments)
	})
	r.Route("/daily", func(r chi.Router) {
		r.Get("/dates", s.handleDailyDates)
		r.Get("/bounds", s.handleDailyBounds)
		r.Get("/", s.handleListDaily)
		r.Get("/aggregates", s.handleDailyAggregates)
	})
	r.Route("/expenses", func(r chi.Router) {
		r.Post("/", s.handleCreateExpense)
		r.Get("/", s.handleListExpenses)
		r.Get("/bounds", s.handleExpenseBounds)
		r.Post("/statements/preview", s.handlePreviewExpenseStatement)
		r.Post("/statements", s.handleImportExpenseStatement)
		r.Get("/{expenseId}", s.handleGetExpense)
		r.Put("/{expenseId}", s.handleReplaceExpense)
		r.Delete("/{expenseId}", s.handleDeleteExpense)
	})
	r.Route("/reports", func(r chi.Router) {
		r.Get("/monthly-series", s.handleMonthlyReportSeries)
		r.Get("/monthly", s.handleMonthlyReport)
	})
	r.Route("/media", func(r chi.Router) {
		r.Post("/sync/{connectorId}", s.handleEnqueueMediaSync)
		r.Get("/aggregates", s.handleMediaAggregates)
		r.Post("/events", s.handleCreateMediaEvent)
		r.Get("/events", s.handleListMediaEvents)
		r.Get("/changes", s.handleListMediaChanges)
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
		r.Post("/{jobId}/retry", s.handleRetryJob)
		r.Post("/{jobId}/cancel", s.handleCancelJob)
	})
	r.Post("/actions/{action}", s.handleAction)
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

// corsMiddleware builds a read-only CORS handler for the given origins.
func corsMiddleware(origins []string) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins: origins,
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowedHeaders: []string{"Accept", "Content-Type", csrfHeaderName},
		ExposedHeaders: []string{"Retry-After", "X-Request-ID", "X-Iroha-Cache"},
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
		)
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if s.deps.ReadyCheck == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": statusNotReady})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), readyzTimeout)
	defer cancel()
	if err := s.deps.ReadyCheck(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": statusNotReady})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": statusReady})
}
