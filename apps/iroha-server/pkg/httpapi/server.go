package httpapi

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	imports "github.com/azusachino/iroha/apps/iroha-imports"
	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
	"github.com/azusachino/iroha/apps/iroha-runtime/config"
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/rawfiles"
	"github.com/azusachino/iroha/apps/iroha-runtime/revisions"
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
	// Bump this when a cached JSON representation or key input changes. The
	// cache is shared across rollouts, so a new server must not reuse an older
	// contract or identity scheme.
	// Bump whenever a cached wire representation or range interpretation
	// changes; old Valkey entries must never satisfy the new contract.
	readCacheKeyVersion = "v13"
	readCacheTTL        = 24 * time.Hour
	readyzTimeout       = 2 * time.Second
	statusReady         = "ready"
	statusNotReady      = "not_ready"
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

type readSnapshotContextKey struct{}

var errReadCacheUncacheable = errors.New("read response is not cacheable")

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

// readCache caches successful JSON reads over the imported, single-user data.
// Writers advance the primary-Postgres revision in the same transaction as
// the canonical write. The revision vector is part of the cache identity;
// backend generations remain a race-safety mechanism for in-flight loads.
func (s *Server) readCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		namespace, ok := readCacheNamespace(r)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		requestContext, finishSnapshot, err := s.readSnapshot(r.Context(), namespace)
		if err != nil {
			w.Header().Set("X-Iroha-Cache", "BYPASS")
			next.ServeHTTP(w, r)
			return
		}
		defer finishSnapshot()
		request := r.WithContext(requestContext)
		if s.deps.Cache == nil {
			next.ServeHTTP(w, request)
			return
		}
		if s.deps.Cache.IsDegraded(namespace) {
			w.Header().Set("X-Iroha-Cache", "BYPASS")
			next.ServeHTTP(w, request)
			return
		}

		vector, err := s.readCacheRevisionVector(request.Context(), namespace)
		if err != nil {
			w.Header().Set("X-Iroha-Cache", "BYPASS")
			next.ServeHTTP(w, request)
			return
		}
		key := cache.KeyWithRevisionVector(s.readCacheKey(request), vector)
		body, generation, ok := cache.GetWithGeneration[[]byte](request.Context(), s.deps.Cache, namespace, key)
		if ok {
			w.Header().Set("X-Iroha-Cache", "HIT")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
			return
		}

		w.Header().Set("X-Iroha-Cache", "MISS")
		loaded := false
		body, err = cache.GetOrLoadAtGeneration(request.Context(), s.deps.Cache, namespace, key, generation, readCacheTTL, func() ([]byte, error) {
			loaded = true
			wrapped := &readCacheResponseWriter{ResponseWriter: w}
			next.ServeHTTP(wrapped, request)
			if wrapped.status != http.StatusOK || wrapped.body.Len() == 0 || !isJSONContentType(wrapped.Header().Get("Content-Type")) {
				return nil, errReadCacheUncacheable
			}
			return wrapped.body.Bytes(), nil
		})
		if loaded || request.Context().Err() != nil {
			// The owner already wrote its response. A canceled waiter must not
			// start another read or interfere with the owner's work.
			return
		}
		if err != nil {
			// Fail-open with this request's own response headers/error context;
			// unsuccessful and non-JSON responses are never shared or cached.
			next.ServeHTTP(w, request)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
}

func (s *Server) readSnapshot(ctx context.Context, namespace string) (context.Context, func(), error) {
	if namespace != cache.NamespaceReports || s.deps.DB == nil {
		return ctx, func() {}, nil
	}
	tx := s.deps.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		return ctx, func() {}, tx.Error
	}
	if err := tx.Exec("set transaction isolation level repeatable read read only").Error; err != nil {
		_ = tx.Rollback().Error
		return ctx, func() {}, err
	}
	return context.WithValue(ctx, readSnapshotContextKey{}, tx), func() {
		_ = tx.WithContext(context.Background()).Rollback().Error
	}, nil
}

func readSnapshotDB(ctx context.Context) *gorm.DB {
	tx, _ := ctx.Value(readSnapshotContextKey{}).(*gorm.DB)
	return tx
}

func (s *Server) readCacheRevisionVector(ctx context.Context, namespace string) (map[string]int64, error) {
	if tx := readSnapshotDB(ctx); tx != nil {
		return revisions.Read(tx.WithContext(ctx), namespace)
	}
	if s.deps.DB == nil {
		return nil, nil
	}
	return revisions.Read(s.deps.DB.WithContext(ctx), namespace)
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
	for prefix, namespace := range map[string]string{
		"/api/v1/activities": cache.NamespaceActivities,
		"/api/v1/briefing":   cache.NamespaceBriefing,
		"/api/v1/coverage":   cache.NamespaceCoverage,
		"/api/v1/daily":      cache.NamespaceDaily,
		"/api/v1/media":      cache.NamespaceMedia,
		"/api/v1/sleep":      cache.NamespaceSleep,
		"/api/v1/metrics":    cache.NamespaceMetrics,
		"/api/v1/reports":    cache.NamespaceReports,
		"/api/v1/expenses":   cache.NamespaceExpenses,
	} {
		if r.URL.Path == prefix || strings.HasPrefix(r.URL.Path, prefix+"/") {
			return namespace, true
		}
	}
	return "", false
}

func (s *Server) readCacheKey(r *http.Request) string {
	key := readCacheKeyVersion + " " + r.Method + " " + r.URL.Path
	queryValues := s.canonicalScopeQuery(r)
	effectiveTimezone := queryValues.Get("timezone")
	queryValues.Del("timezone")
	if query := queryValues.Encode(); query != "" {
		key += "?" + query
	}
	if effectiveTimezone == "" {
		effectiveTimezone = s.deps.Config.Server.Timezone
	}
	if effectiveTimezone != "" {
		key += "|effective_timezone=" + url.QueryEscape(effectiveTimezone)
	}
	return key
}

func (s *Server) canonicalScopeQuery(r *http.Request) url.Values {
	query := cloneValues(r.URL.Query())
	if location, err := scopeLocation(query, s.deps.Config.Server.Timezone); err == nil {
		query.Set("timezone", location.String())
	}
	input, active, err := readScopeInput(query, s.deps.Config.Server.Timezone)
	if err != nil || !active {
		return query
	}
	scope, err := ResolveReadScope(input, s.clockNow())
	if err != nil {
		return query
	}
	for _, name := range []string{"date", "scope", "month", "year", "end", "from", "to"} {
		query.Del(name)
	}
	switch scope.Kind {
	case ScopeLifetime:
		query.Set("scope", string(ScopeLifetime))
	case ScopeRange:
		query.Set("from", scope.Calendar.From.Format(calendarDateLayout))
		query.Set("to", scope.Calendar.ToExclusive.Format(calendarDateLayout))
	default:
		query.Set("date", canonicalScopeDate(scope))
	}
	query.Set("_scope", canonicalScopeVersion)
	return query
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
