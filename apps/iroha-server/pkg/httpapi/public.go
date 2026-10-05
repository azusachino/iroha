package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	readcache "github.com/azusachino/iroha/apps/iroha-runtime/cache"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/publicexport"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

const (
	// publicRateLimitPerMin bounds anonymous reads per client.
	publicRateLimitPerMin = 120
	// publicSnapshotTTL caps how long one projection is served. It is also
	// keyed on the activity revision, so a new or changed activity rebuilds
	// it at once and anonymous traffic never triggers more than that.
	publicSnapshotTTL = 24 * time.Hour
	// publicCacheControl lets browsers and the edge keep public data a day.
	publicCacheControl = "public, max-age=86400"
)

// publicSnapshotCache holds the last validated projection. Rebuilds are
// serialized, so a burst of visitors triggers one computation.
type publicSnapshotCache struct {
	mu       sync.Mutex
	encoded  map[string]publicRepresentation
	builtAt  time.Time
	revision string
}

// publicRoutes serves the anonymous, read-only public projection (ADR-0008:
// only the sanitized projection is public). It is mounted outside /api/v1
// and never sees owner sessions.
func (s *Server) publicRoutes(r chi.Router) {
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{http.MethodGet, http.MethodHead, http.MethodOptions},
		MaxAge:         300,
	}))
	r.Use(s.limitByClient(publicRateLimitPerMin, time.Minute))
	for _, part := range []string{"summary", "activities", "routes", "meta"} {
		r.Get("/"+part, s.servePublic(part))
	}
	r.Get("/activities/{activityId}", s.handlePublicActivityDetail)
}

type publicRepresentation struct {
	body []byte
	etag string
}

func encodePublicSnapshot(snapshot publicexport.Snapshot) (map[string]publicRepresentation, error) {
	encoded := make(map[string]publicRepresentation)
	for part, value := range map[string]any{"summary": snapshot.Summary, "activities": snapshot.Activities, "routes": snapshot.Routes, "meta": snapshot.Meta} {
		var body bytes.Buffer
		if err := json.NewEncoder(&body).Encode(value); err != nil {
			return nil, err
		}
		encoded[part] = publicRepresentation{body: body.Bytes(), etag: fmt.Sprintf("\"%x\"", sha256.Sum256(body.Bytes()))}
	}
	return encoded, nil
}

// Keep whole-document semantics while delegating validator syntax and
// precondition precedence to net/http (ADR-0010).
type wholePublicResponse struct{ http.ResponseWriter }

func (w wholePublicResponse) WriteHeader(status int) {
	w.Header().Del("Accept-Ranges")
	if status == http.StatusPreconditionFailed {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Del("ETag")
		w.Header().Del("Content-Type")
	}
	w.ResponseWriter.WriteHeader(status)
}

func servePublicRepresentation(w http.ResponseWriter, r *http.Request, value publicRepresentation) {
	request := r.Clone(r.Context())
	request.Header.Del("Range")
	request.Header.Del("If-Range")
	for _, header := range []string{"If-Match", "If-None-Match"} {
		if fields := r.Header.Values(header); len(fields) > 0 {
			request.Header.Set(header, strings.Join(fields, ","))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", publicCacheControl)
	w.Header().Set("ETag", value.etag)
	http.ServeContent(wholePublicResponse{w}, request, "", time.Time{}, bytes.NewReader(value.body))
}

func (s *Server) servePublic(part string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snap, err := s.publicSnapshot(r)
		if err != nil {
			s.deps.Logger.Error("build public projection", "error", err)
			writeContractError(w, http.StatusInternalServerError, "public_unavailable", "public data is unavailable")
			return
		}
		servePublicRepresentation(w, r, snap[part])
	}
}

func (s *Server) publicSnapshot(r *http.Request) (map[string]publicRepresentation, error) {
	cache := s.publicCache
	cache.mu.Lock()
	defer cache.mu.Unlock()
	now := s.now()
	vector, err := s.readCacheRevisionVector(r.Context(), readcache.NamespaceActivities)
	if err != nil {
		return nil, err
	}
	revision := fmt.Sprint(vector)
	if !cache.builtAt.IsZero() && now.Sub(cache.builtAt) < publicSnapshotTTL && revision == cache.revision {
		return cache.encoded, nil
	}
	snap, err := publicexport.BuildSnapshot(r.Context(), s.deps.ActivityService, s.deps.GeocodeService, now)
	if err != nil {
		return nil, err
	}
	encoded, err := encodePublicSnapshot(snap)
	if err != nil {
		return nil, err
	}
	cache.encoded, cache.builtAt, cache.revision = encoded, now, revision
	return encoded, nil
}

func (s *Server) handlePublicActivityDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "activityId")
	if !strings.HasPrefix(id, ids.ActivityPrefix+"_") {
		writeContractError(w, http.StatusNotFound, "not_found", "activity not found")
		return
	}
	detail, found, err := publicexport.Detail(s.deps.ActivityService, id)
	switch {
	case err != nil:
		s.deps.Logger.Error("build public activity detail", "error", err)
		writeContractError(w, http.StatusInternalServerError, "public_unavailable", "public data is unavailable")
	case !found:
		writeContractError(w, http.StatusNotFound, "not_found", "activity not found")
	default:
		w.Header().Set("Cache-Control", publicCacheControl)
		writeJSON(w, http.StatusOK, detail)
	}
}
