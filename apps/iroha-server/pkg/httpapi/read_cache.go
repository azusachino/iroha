package httpapi

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
	"github.com/azusachino/iroha/apps/iroha-runtime/revisions"
	"gorm.io/gorm"
)

const (
	// Bump this when a cached JSON representation or key input changes. The
	// cache is shared across rollouts, so a new server must not reuse an older
	// contract or identity scheme.
	// Bump whenever a cached wire representation or range interpretation
	// changes; old Valkey entries must never satisfy the new contract.
	readCacheKeyVersion = "v13"
	readCacheTTL        = 24 * time.Hour
)

type readSnapshotContextKey struct{}

var errReadCacheUncacheable = errors.New("read response is not cacheable")

// ReadCacheInterceptor intercepts HTTP read requests to serve cached JSON responses,
// enforce repeatable-read snapshots for multi-query reports, and coalesce concurrent
// misses through single-flight backend loading.
type ReadCacheInterceptor struct {
	cache    *cache.Client
	db       *gorm.DB
	timezone string
	clock    func() time.Time
}

// NewReadCacheInterceptor constructs an interceptor with explicit dependencies.
func NewReadCacheInterceptor(c *cache.Client, db *gorm.DB, timezone string, clock func() time.Time) *ReadCacheInterceptor {
	if clock == nil {
		clock = time.Now
	}
	return &ReadCacheInterceptor{
		cache:    c,
		db:       db,
		timezone: timezone,
		clock:    clock,
	}
}

func (s *Server) readCacheInterceptor() *ReadCacheInterceptor {
	tz := ""
	var clock func() time.Time
	if s != nil {
		tz = s.deps.Config.Server.Timezone
		clock = s.clockNow
		return &ReadCacheInterceptor{
			cache:    s.deps.Cache,
			db:       s.deps.DB,
			timezone: tz,
			clock:    clock,
		}
	}
	return NewReadCacheInterceptor(nil, nil, "", nil)
}

// Intercept wraps an HTTP handler with the read-cache lifecycle:
// namespace detection -> snapshot binding -> revision check -> cache lookup -> single-flight load -> response buffer.
func (i *ReadCacheInterceptor) Intercept(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		namespace, ok := readCacheNamespace(r)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		requestContext, finishSnapshot, err := i.Snapshot(r.Context(), namespace)
		if err != nil {
			w.Header().Set("X-Iroha-Cache", "BYPASS")
			next.ServeHTTP(w, r)
			return
		}
		defer finishSnapshot()
		request := r.WithContext(requestContext)
		if i.cache == nil {
			next.ServeHTTP(w, request)
			return
		}
		if i.cache.IsDegraded(namespace) {
			w.Header().Set("X-Iroha-Cache", "BYPASS")
			next.ServeHTTP(w, request)
			return
		}

		vector, err := i.RevisionVector(request.Context(), namespace)
		if err != nil {
			w.Header().Set("X-Iroha-Cache", "BYPASS")
			next.ServeHTTP(w, request)
			return
		}
		key := cache.KeyWithRevisionVector(i.Key(request), vector)
		body, generation, ok := cache.GetWithGeneration[[]byte](request.Context(), i.cache, namespace, key)
		if ok {
			w.Header().Set("X-Iroha-Cache", "HIT")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
			return
		}

		w.Header().Set("X-Iroha-Cache", "MISS")
		loaded := false
		body, err = cache.GetOrLoadAtGeneration(request.Context(), i.cache, namespace, key, generation, readCacheTTL, func() ([]byte, error) {
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

func (s *Server) readCache(next http.Handler) http.Handler {
	return s.readCacheInterceptor().Intercept(next)
}

func (i *ReadCacheInterceptor) Snapshot(ctx context.Context, namespace string) (context.Context, func(), error) {
	if namespace != cache.NamespaceReports || i.db == nil {
		return ctx, func() {}, nil
	}
	tx := i.db.WithContext(ctx).Begin()
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

func (i *ReadCacheInterceptor) RevisionVector(ctx context.Context, namespace string) (map[string]int64, error) {
	if tx := readSnapshotDB(ctx); tx != nil {
		return revisions.Read(tx.WithContext(ctx), namespace)
	}
	if i.db == nil {
		return nil, nil
	}
	return revisions.Read(i.db.WithContext(ctx), namespace)
}

func (s *Server) readCacheRevisionVector(ctx context.Context, namespace string) (map[string]int64, error) {
	return s.readCacheInterceptor().RevisionVector(ctx, namespace)
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

func (i *ReadCacheInterceptor) Key(r *http.Request) string {
	key := readCacheKeyVersion + " " + r.Method + " " + r.URL.Path
	queryValues := i.CanonicalScopeQuery(r)
	effectiveTimezone := queryValues.Get("timezone")
	queryValues.Del("timezone")
	if query := queryValues.Encode(); query != "" {
		key += "?" + query
	}
	if effectiveTimezone == "" {
		effectiveTimezone = i.timezone
	}
	if effectiveTimezone != "" {
		key += "|effective_timezone=" + url.QueryEscape(effectiveTimezone)
	}
	return key
}

func (s *Server) readCacheKey(r *http.Request) string {
	return s.readCacheInterceptor().Key(r)
}

func (i *ReadCacheInterceptor) CanonicalScopeQuery(r *http.Request) url.Values {
	query := cloneValues(r.URL.Query())
	if location, err := scopeLocation(query, i.timezone); err == nil {
		query.Set("timezone", location.String())
	}
	input, active, err := readScopeInput(query, i.timezone)
	if err != nil || !active {
		return query
	}
	now := time.Now()
	if i.clock != nil {
		now = i.clock()
	}
	scope, err := ResolveReadScope(input, now)
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
