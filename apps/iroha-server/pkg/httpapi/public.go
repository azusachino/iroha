package httpapi

import (
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
	snapshot publicexport.Snapshot
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
	r.Get("/summary", s.servePublic(func(snap publicexport.Snapshot) any { return snap.Summary }))
	r.Get("/activities", s.servePublic(func(snap publicexport.Snapshot) any { return snap.Activities }))
	r.Get("/routes", s.servePublic(func(snap publicexport.Snapshot) any { return snap.Routes }))
	r.Get("/meta", s.servePublic(func(snap publicexport.Snapshot) any { return snap.Meta }))
	r.Get("/activities/{activityId}", s.handlePublicActivityDetail)
}

func (s *Server) servePublic(pick func(publicexport.Snapshot) any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snap, err := s.publicSnapshot(r)
		if err != nil {
			s.deps.Logger.Error("build public projection", "error", err)
			writeContractError(w, http.StatusInternalServerError, "public_unavailable", "public data is unavailable")
			return
		}
		w.Header().Set("Cache-Control", publicCacheControl)
		writeJSON(w, http.StatusOK, pick(snap))
	}
}

func (s *Server) publicSnapshot(r *http.Request) (publicexport.Snapshot, error) {
	cache := s.publicCache
	cache.mu.Lock()
	defer cache.mu.Unlock()
	now := s.now()
	vector, err := s.readCacheRevisionVector(r.Context(), readcache.NamespaceActivities)
	if err != nil {
		return publicexport.Snapshot{}, err
	}
	revision := fmt.Sprint(vector)
	if !cache.builtAt.IsZero() && now.Sub(cache.builtAt) < publicSnapshotTTL && revision == cache.revision {
		return cache.snapshot, nil
	}
	snap, err := publicexport.BuildSnapshot(r.Context(), s.deps.ActivityService, s.deps.GeocodeService, now)
	if err != nil {
		return publicexport.Snapshot{}, err
	}
	cache.snapshot, cache.builtAt, cache.revision = snap, now, revision
	return snap, nil
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
