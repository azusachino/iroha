package httpapi

import (
	"net/http"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/daily"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/expenses"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/media"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/sleep"
)

// readScopeCalendarBounds adapts the transport-neutral scope into the UTC
// calendar dates used by date-keyed tables. Timestamp-backed handlers use the
// scope.Instant bounds instead.
func (s *Server) readScopeCalendarBounds(w http.ResponseWriter, r *http.Request) (*time.Time, *time.Time, bool) {
	scope, active, err := s.resolveReadScope(r)
	if err != nil {
		writeReadScopeError(w, err)
		return nil, nil, false
	}
	if !active || scope.Kind == ScopeLifetime {
		return nil, nil, true
	}
	from, to := scope.Calendar.From, scope.Calendar.ToExclusive
	return &from, &to, true
}

func (s *Server) parseDailyFilters(w http.ResponseWriter, r *http.Request) (daily.ListFilters, bool) {
	filters, ok := parseDailyFilters(w, r)
	if !ok {
		return filters, ok
	}
	from, to, ok := s.readScopeCalendarBounds(w, r)
	if !ok {
		return daily.ListFilters{}, false
	}
	if from == nil || to == nil {
		return filters, true
	}
	filters.From, filters.To = from, to
	return filters, true
}

func (s *Server) parseDailyAggregateFilters(w http.ResponseWriter, r *http.Request) (daily.AggregateFilters, bool) {
	filters, ok := parseDailyAggregateFilters(w, r)
	if !ok {
		return filters, ok
	}
	from, to, ok := s.readScopeCalendarBounds(w, r)
	if !ok {
		return daily.AggregateFilters{}, false
	}
	if from == nil || to == nil {
		return filters, true
	}
	filters.From, filters.To = from, to
	return filters, true
}

func (s *Server) parseSleepFilters(w http.ResponseWriter, r *http.Request) (sleep.ListFilters, bool) {
	filters, ok := parseSleepFilters(w, r)
	if !ok {
		return filters, ok
	}
	from, to, ok := s.readScopeCalendarBounds(w, r)
	if !ok {
		return sleep.ListFilters{}, false
	}
	if from == nil || to == nil {
		return filters, true
	}
	filters.From, filters.To = from, to
	return filters, true
}

func (s *Server) parseSleepAggregateFilters(w http.ResponseWriter, r *http.Request) (sleep.AggregateFilters, bool) {
	filters, ok := parseSleepAggregateFilters(w, r)
	if !ok {
		return filters, ok
	}
	from, to, ok := s.readScopeCalendarBounds(w, r)
	if !ok {
		return sleep.AggregateFilters{}, false
	}
	if from == nil || to == nil {
		return filters, true
	}
	filters.From, filters.To = from, to
	return filters, true
}

func (s *Server) parseExpenseFilters(w http.ResponseWriter, r *http.Request) (expenses.ListFilters, bool) {
	filters, ok := parseExpenseFilters(w, r)
	if !ok {
		return filters, ok
	}
	from, to, ok := s.readScopeCalendarBounds(w, r)
	if !ok {
		return expenses.ListFilters{}, false
	}
	if from == nil || to == nil {
		return filters, true
	}
	filters.From, filters.To = from, to
	return filters, true
}

func (s *Server) parseMediaEventFilters(w http.ResponseWriter, r *http.Request) (media.EventListFilters, bool) {
	filters, ok := parseMediaEventFilters(w, r)
	if !ok {
		return filters, ok
	}
	scope, active, err := s.resolveReadScope(r)
	if err != nil {
		writeReadScopeError(w, err)
		return media.EventListFilters{}, false
	}
	if active && scope.Kind != ScopeLifetime {
		from, to := scope.Instant.From, scope.Instant.ToExclusive
		filters.From, filters.To = &from, &to
	} else if active {
		filters.From, filters.To = nil, nil
	}
	return filters, true
}
