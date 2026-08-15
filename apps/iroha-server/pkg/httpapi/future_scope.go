package httpapi

import (
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	futureScopeCode    = "future_date_not_allowed"
	futureScopeMessage = "future dates are not available"
)

// rejectFutureReadScope runs before readCache so arbitrary future query values
// cannot create cache keys or make a canonical read before being rejected.
// Exclusive range ends are intentionally not checked here: current month/year
// callers use the next calendar boundary as their upper bound. Their starts,
// and all scalar period selectors, remain bounded to observed time.
func (s *Server) rejectFutureReadScope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}

		location, err := scopeLocation(r.URL.Query(), s.deps.Config.Server.Timezone)

		if err == nil {
			clock := s.now
			if clock == nil {
				clock = time.Now
			}
			nowInstant := clock()
			now := nowInstant.In(location)
			query := r.URL.Query()
			if futureCalendarValue(query.Get("date"), now, "2006-01-02") ||
				futureCalendarValue(query.Get("month"), now, "2006-01") ||
				futureCalendarValue(query.Get("end"), now, "2006-01") ||
				futureYearValue(query.Get("year"), now) ||
				futureYearValue(query.Get("completed_year"), now) ||
				futureCalendarValue(query.Get("from"), now, "2006-01-02") ||
				futureInstantValue(query.Get("started_from"), nowInstant) {
				writeFutureScopeError(w)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func scopeLocation(query url.Values, fallback string) (*time.Location, error) {
	timezone := query.Get("timezone")
	if timezone == "" {
		timezone = fallback
	}
	if timezone == "" {
		timezone = "Asia/Tokyo"
	}
	return time.LoadLocation(timezone)
}

func futureCalendarValue(value string, now time.Time, layout string) bool {
	if value == "" {
		return false
	}
	parsed, err := time.Parse(layout, value)
	if err != nil || parsed.Format(layout) != value {
		return false
	}
	return value > now.Format(layout)
}

func futureYearValue(value string, now time.Time) bool {
	if len(value) != 4 {
		return false
	}
	year, err := strconv.Atoi(value)
	if err != nil {
		return false
	}
	return year > now.Year()
}

func futureInstantValue(value string, now time.Time) bool {
	if value == "" {
		return false
	}
	parsed, err := time.Parse(time.RFC3339, value)
	return err == nil && parsed.After(now)
}

func writeFutureScopeError(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	writeContractError(w, http.StatusBadRequest, futureScopeCode, futureScopeMessage)
}
