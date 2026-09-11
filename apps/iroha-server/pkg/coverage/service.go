package coverage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-runtime/revisions"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	CompletenessUnknown      = "unknown"
	CompletenessPartial      = "partial"
	CompletenessCovered      = "covered"
	CompletenessCoveredEmpty = "covered_empty"
)

var (
	ErrInvalidAssertion = errors.New("invalid coverage assertion")
	ErrInvalidQuery     = errors.New("invalid coverage query")
)

type Service struct {
	db *gorm.DB
}

type AssertInput struct {
	ID               uuid.UUID
	SourceInstanceID uuid.UUID
	SourceReceiptID  *uuid.UUID
	ImportSnapshotID *uuid.UUID
	Category         string
	ScopeJSON        json.RawMessage
	From             time.Time
	To               time.Time
	Timezone         string
	IngestionMode    string
	Completeness     string
	RecordedAt       time.Time
	CreatedAt        time.Time
}

type QueryFilters struct {
	SourceInstanceID uuid.UUID
	Category         string
	From             time.Time
	To               time.Time
}

type Interval struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type Assertion struct {
	ID               uuid.UUID       `json:"id"`
	SourceInstanceID uuid.UUID       `json:"source_instance_id"`
	SourceReceiptID  *uuid.UUID      `json:"source_receipt_id"`
	ImportSnapshotID *uuid.UUID      `json:"import_snapshot_id"`
	Category         string          `json:"category"`
	ScopeJSON        json.RawMessage `json:"scope"`
	Interval         Interval        `json:"interval"`
	Timezone         string          `json:"timezone"`
	IngestionMode    string          `json:"ingestion_mode"`
	Completeness     string          `json:"completeness"`
	RecordedAt       time.Time       `json:"recorded_at"`
}

type Result struct {
	SourceInstanceID uuid.UUID   `json:"source_instance_id"`
	Category         string      `json:"category"`
	Requested        Interval    `json:"requested"`
	State            string      `json:"state"`
	Covered          []Interval  `json:"covered"`
	Gaps             []Interval  `json:"gaps"`
	Assertions       []Assertion `json:"assertions"`
}

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

// Assert records evidence-backed collection coverage. The caller may provide
// a transaction so canonical writes and this assertion commit atomically.
func (s *Service) Assert(tx *gorm.DB, input AssertInput) (models.SourceCoverageAssertion, error) {
	if tx == nil {
		if s.db == nil {
			return models.SourceCoverageAssertion{}, ErrInvalidAssertion
		}
		var row models.SourceCoverageAssertion
		err := s.db.Transaction(func(inner *gorm.DB) error {
			var err error
			row, err = s.Assert(inner, input)
			return err
		})
		return row, err
	}
	db := tx
	if input.SourceInstanceID == uuid.Nil || input.Category == "" || input.From.IsZero() || !input.From.Before(input.To) || input.Timezone == "" || input.SourceReceiptID == nil && input.ImportSnapshotID == nil {
		return models.SourceCoverageAssertion{}, ErrInvalidAssertion
	}
	if _, err := time.LoadLocation(input.Timezone); err != nil {
		return models.SourceCoverageAssertion{}, fmt.Errorf("%w: timezone: %v", ErrInvalidAssertion, err)
	}
	if input.IngestionMode != "full_snapshot" && input.IngestionMode != "bounded_replacement" && input.IngestionMode != "incremental" {
		return models.SourceCoverageAssertion{}, ErrInvalidAssertion
	}
	if input.Completeness != CompletenessUnknown && input.Completeness != CompletenessPartial && input.Completeness != CompletenessCovered && input.Completeness != CompletenessCoveredEmpty {
		return models.SourceCoverageAssertion{}, ErrInvalidAssertion
	}
	if len(input.ScopeJSON) == 0 {
		input.ScopeJSON = json.RawMessage(`{}`)
	}
	var scope map[string]any
	if err := json.Unmarshal(input.ScopeJSON, &scope); err != nil || scope == nil {
		return models.SourceCoverageAssertion{}, ErrInvalidAssertion
	}
	if input.ID == uuid.Nil {
		input.ID = uuid.New()
	}
	if input.RecordedAt.IsZero() {
		input.RecordedAt = time.Now().UTC()
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = input.RecordedAt
	}
	row := models.SourceCoverageAssertion{
		ID: input.ID, SourceInstanceID: input.SourceInstanceID, SourceReceiptID: input.SourceReceiptID,
		ImportSnapshotID: input.ImportSnapshotID, Category: input.Category, ScopeJSON: input.ScopeJSON,
		IntervalStart: input.From, IntervalEnd: input.To, Timezone: input.Timezone,
		IngestionMode: input.IngestionMode, Completeness: input.Completeness,
		RecordedAt: input.RecordedAt, CreatedAt: input.CreatedAt,
	}
	if err := db.Create(&row).Error; err != nil {
		return models.SourceCoverageAssertion{}, err
	}
	if err := revisions.Bump(db, revisions.NamespaceCoverage, revisions.NamespaceBriefing, revisions.NamespaceMetrics, revisions.NamespaceReports); err != nil {
		return models.SourceCoverageAssertion{}, err
	}
	return row, nil
}

func (s *Service) Query(ctx context.Context, filters QueryFilters) (Result, error) {
	if s.db == nil || filters.SourceInstanceID == uuid.Nil || filters.Category == "" || filters.From.IsZero() || !filters.From.Before(filters.To) {
		return Result{}, ErrInvalidQuery
	}
	var rows []models.SourceCoverageAssertion
	if err := s.db.WithContext(ctx).
		Where("source_instance_id = ? and category = ? and interval_start < ? and interval_end > ?", filters.SourceInstanceID, filters.Category, filters.To, filters.From).
		Order("recorded_at desc, created_at desc, id desc").Find(&rows).Error; err != nil {
		return Result{}, err
	}
	requested := Interval{From: filters.From, To: filters.To}
	result := Result{SourceInstanceID: filters.SourceInstanceID, Category: filters.Category, Requested: requested, State: CompletenessUnknown, Covered: []Interval{}, Gaps: []Interval{}, Assertions: []Assertion{}}
	occupied := []Interval{}
	covered := []Interval{}
	hasKnown := false
	hasPartial := false
	allEmpty := true
	for _, row := range rows {
		interval, ok := intersect(Interval{From: row.IntervalStart, To: row.IntervalEnd}, requested)
		if !ok {
			continue
		}
		result.Assertions = append(result.Assertions, Assertion{
			ID: row.ID, SourceInstanceID: row.SourceInstanceID, SourceReceiptID: row.SourceReceiptID,
			ImportSnapshotID: row.ImportSnapshotID, Category: row.Category, ScopeJSON: row.ScopeJSON,
			Interval: interval, Timezone: row.Timezone, IngestionMode: row.IngestionMode,
			Completeness: row.Completeness, RecordedAt: row.RecordedAt,
		})
		remaining := subtract(interval, occupied)
		occupied = mergeIntervals(append(occupied, remaining...))
		switch row.Completeness {
		case CompletenessCovered, CompletenessCoveredEmpty:
			hasKnown = true
			if row.Completeness == CompletenessCovered {
				allEmpty = false
			}
			covered = mergeIntervals(append(covered, remaining...))
		case CompletenessPartial:
			hasPartial = true
		}
	}
	result.Covered = covered
	result.Gaps = complement(requested, covered)
	switch {
	case len(result.Gaps) == 0 && len(covered) > 0 && allEmpty:
		result.State = CompletenessCoveredEmpty
	case len(result.Gaps) == 0 && len(covered) > 0:
		result.State = CompletenessCovered
	case hasKnown || hasPartial:
		result.State = CompletenessPartial
	}
	if len(result.Assertions) == 0 {
		result.Gaps = []Interval{requested}
	}
	return result, nil
}

func intersect(a, b Interval) (Interval, bool) {
	from, to := a.From, a.To
	if b.From.After(from) {
		from = b.From
	}
	if b.To.Before(to) {
		to = b.To
	}
	return Interval{From: from, To: to}, from.Before(to)
}

func subtract(input Interval, occupied []Interval) []Interval {
	remaining := []Interval{input}
	for _, used := range occupied {
		next := make([]Interval, 0, len(remaining)+1)
		for _, part := range remaining {
			if used.To.Before(part.From) || !used.From.Before(part.To) {
				next = append(next, part)
				continue
			}
			if part.From.Before(used.From) {
				next = append(next, Interval{From: part.From, To: used.From})
			}
			if used.To.Before(part.To) {
				next = append(next, Interval{From: used.To, To: part.To})
			}
		}
		remaining = next
	}
	return remaining
}

func mergeIntervals(intervals []Interval) []Interval {
	if len(intervals) < 2 {
		return intervals
	}
	sort.Slice(intervals, func(i, j int) bool { return intervals[i].From.Before(intervals[j].From) })
	merged := make([]Interval, 0, len(intervals))
	for _, interval := range intervals {
		if len(merged) == 0 || merged[len(merged)-1].To.Before(interval.From) {
			merged = append(merged, interval)
			continue
		}
		if interval.To.After(merged[len(merged)-1].To) {
			merged[len(merged)-1].To = interval.To
		}
	}
	return merged
}

func complement(requested Interval, covered []Interval) []Interval {
	if len(covered) == 0 {
		return []Interval{requested}
	}
	gaps := make([]Interval, 0, len(covered)+1)
	cursor := requested.From
	for _, interval := range covered {
		if cursor.Before(interval.From) {
			gaps = append(gaps, Interval{From: cursor, To: interval.From})
		}
		if cursor.Before(interval.To) {
			cursor = interval.To
		}
	}
	if cursor.Before(requested.To) {
		gaps = append(gaps, Interval{From: cursor, To: requested.To})
	}
	return gaps
}
