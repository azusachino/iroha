package httpapi

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	futureScopeCode       = "future_date_not_allowed"
	futureScopeMessage    = "future dates are not available"
	invalidScopeCode      = "invalid_read_scope"
	invalidScopeMessage   = "invalid read scope"
	calendarDateLayout    = "2006-01-02"
	calendarMonthLayout   = "2006-01"
	calendarYearLayout    = "2006"
	canonicalScopeVersion = "scope-v1"
)

// rejectFutureReadScope is the request-boundary half of the temporal scope
// contract. It runs before readCache so malformed/future scopes cannot create
// cache entries. Handlers resolve the same input again to map it into their
// domain-specific filters.
func (s *Server) rejectFutureReadScope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}

		now := s.clockNow()
		input, active, err := readScopeInput(r.URL.Query(), s.deps.Config.Server.Timezone)
		if err != nil {
			writeReadScopeError(w, err)
			return
		}
		if active {
			if _, err := ResolveReadScope(input, now); err != nil {
				writeReadScopeError(w, err)
				return
			}
		}

		query := r.URL.Query()
		location, err := scopeLocation(query, s.deps.Config.Server.Timezone)
		if err == nil {
			nowInstant := now
			if futureYearValue(query.Get("completed_year"), nowInstant.In(location)) ||
				futureInstantValue(query.Get("started_from"), nowInstant) {
				writeFutureScopeError(w)
				return
			}
			// Existing range endpoints allow a single lower bound. Keep that
			// compatibility form, but still prevent a future lower bound before
			// it reaches a handler.
			if !active && futureCalendarValue(query.Get("from"), nowInstant.In(location), calendarDateLayout) {
				writeFutureScopeError(w)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) clockNow() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

// readScopeInput accepts one canonical scalar/range form and the old aliases
// during migration. It returns active=false for the legacy single-bound form
// (`from` without `to`) because those endpoints intentionally support it.
func readScopeInput(query url.Values, fallbackTimezone string) (ReadScopeInput, bool, error) {
	if err := rejectRepeatedScopeValues(query); err != nil {
		return ReadScopeInput{}, false, err
	}
	aliases := make([]string, 0, 2)
	for _, name := range []string{"date", "month", "year", "end"} {
		if query.Get(name) != "" {
			aliases = append(aliases, name)
		}
	}
	if len(aliases) > 1 {
		if len(aliases) == 2 && aliases[0] == "month" && aliases[1] == "year" &&
			isLegacyYear(query.Get("year")) && isLegacyMonthNumber(query.Get("month")) {
			aliases = []string{"month"}
		} else {
			return ReadScopeInput{}, false, ErrConflictingScope
		}
	}

	input := ReadScopeInput{Timezone: query.Get("timezone")}
	if input.Timezone == "" {
		input.Timezone = fallbackTimezone
	}
	if query.Get("scope") != "" {
		if query.Get("scope") != string(ScopeLifetime) {
			return ReadScopeInput{}, false, ErrInvalidReadScope
		}
		input.Lifetime = true
	}
	if len(aliases) == 1 {
		switch aliases[0] {
		case "date":
			input.Date = query.Get("date")
		case "month":
			input.Date = query.Get("month")
			if isLegacyYear(query.Get("year")) && isLegacyMonthNumber(input.Date) {
				month := input.Date
				if len(month) == 1 {
					month = "0" + month
				}
				input.Date = query.Get("year") + "-" + month
			}
		case "year":
			input.Date = query.Get("year")
		case "end":
			input.Date = query.Get("end")
		}
	}
	input.From = query.Get("from")
	input.To = query.Get("to")
	if input.Lifetime || input.Date != "" {
		if input.From != "" || input.To != "" {
			return ReadScopeInput{}, false, ErrConflictingScope
		}
		return input, true, nil
	}
	if input.From != "" && input.To != "" {
		return input, true, nil
	}
	return input, false, nil
}

func isLegacyYear(value string) bool {
	year, err := strconv.Atoi(value)
	return len(value) == len(calendarYearLayout) && err == nil && year > 0
}

func isLegacyMonthNumber(value string) bool {
	if value == "" || !allReadScopeDigits(value) {
		return false
	}
	month, err := strconv.Atoi(value)
	return err == nil && month >= 1 && month <= 12
}

func rejectRepeatedScopeValues(query url.Values) error {
	for _, name := range []string{"date", "scope", "month", "year", "end", "from", "to", "timezone"} {
		if len(query[name]) > 1 {
			return ErrConflictingScope
		}
	}
	return nil
}

func (s *Server) resolveReadScope(r *http.Request) (ReadScope, bool, error) {
	input, active, err := readScopeInput(r.URL.Query(), s.deps.Config.Server.Timezone)
	if err != nil || !active {
		return ReadScope{}, active, err
	}
	scope, err := ResolveReadScope(input, s.clockNow())
	return scope, true, err
}

func scopeLocation(query url.Values, fallback string) (*time.Location, error) {
	timezone := query.Get("timezone")
	if timezone == "" {
		timezone = fallback
	}
	return loadReadScopeTimezone(timezone)
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
	if len(value) != len(calendarYearLayout) {
		return false
	}
	year, err := strconv.Atoi(value)
	return err == nil && year > now.Year()
}

func futureInstantValue(value string, now time.Time) bool {
	if value == "" {
		return false
	}
	parsed, err := time.Parse(time.RFC3339, value)
	return err == nil && parsed.After(now)
}

func writeReadScopeError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrFutureReadScope) {
		writeFutureScopeError(w)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if errors.Is(err, ErrInvalidScopeRange) {
		writeContractError(w, http.StatusBadRequest, "invalid_date_range", "invalid date range")
		return
	}
	writeContractError(w, http.StatusBadRequest, invalidScopeCode, invalidScopeMessage)
}

func writeFutureScopeError(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	writeContractError(w, http.StatusBadRequest, futureScopeCode, futureScopeMessage)
}

func canonicalScopeDate(scope ReadScope) string {
	switch scope.Kind {
	case ScopeYear:
		return scope.Calendar.From.Format(calendarYearLayout)
	case ScopeMonth:
		return scope.Calendar.From.Format(calendarMonthLayout)
	case ScopeDay:
		return scope.Calendar.From.Format(calendarDateLayout)
	default:
		return ""
	}
}

func cloneValues(input url.Values) url.Values {
	output := make(url.Values, len(input))
	for key, values := range input {
		output[key] = append([]string(nil), values...)
	}
	return output
}
