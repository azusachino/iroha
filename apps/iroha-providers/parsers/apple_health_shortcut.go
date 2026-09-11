package parsers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
)

const AppleHealthShortcutSchema = "iroha.health.shortcut.v1"

type AppleHealthShortcutMetadata struct {
	SourceInstanceKey string
	CapturedAt        time.Time
}

type shortcutEnvelope struct {
	Schema            string                 `json:"schema"`
	SourceInstanceKey string                 `json:"source_instance_key"`
	CapturedAt        string                 `json:"captured_at"`
	Coverage          []shortcutCoverage     `json:"coverage"`
	Activities        []shortcutActivity     `json:"activities"`
	Sleep             []shortcutSleep        `json:"sleep"`
	DailySummaries    []shortcutDailySummary `json:"daily_summaries"`
	DailyMetrics      []shortcutDailyMetric  `json:"daily_metrics"`
}

type shortcutCoverage struct {
	Category     string          `json:"category"`
	From         string          `json:"from"`
	To           string          `json:"to"`
	Timezone     string          `json:"timezone"`
	Completeness string          `json:"completeness"`
	Scope        json.RawMessage `json:"scope"`
}

type shortcutActivity struct {
	ID           string   `json:"id"`
	SportType    string   `json:"sport_type"`
	Title        string   `json:"title"`
	StartedAt    string   `json:"started_at"`
	EndedAt      string   `json:"ended_at"`
	DistanceM    *float64 `json:"distance_m"`
	DurationS    *int     `json:"duration_s"`
	AvgHR        *int     `json:"avg_hr"`
	MaxHR        *int     `json:"max_hr"`
	CaloriesKcal *float64 `json:"calories_kcal"`
}

type shortcutSleep struct {
	WakeDate     string                 `json:"wake_date"`
	StartedAt    string                 `json:"started_at"`
	EndedAt      string                 `json:"ended_at"`
	TimeInBedS   int                    `json:"time_in_bed_s"`
	AsleepS      int                    `json:"asleep_s"`
	Efficiency   float64                `json:"efficiency"`
	IsMainSleep  bool                   `json:"is_main_sleep"`
	CoreS        int                    `json:"core_s"`
	DeepS        int                    `json:"deep_s"`
	RemS         int                    `json:"rem_s"`
	AwakeS       int                    `json:"awake_s"`
	UnspecifiedS int                    `json:"unspecified_s"`
	Segments     []shortcutSleepSegment `json:"segments"`
}

type shortcutSleepSegment struct {
	Stage     string `json:"stage"`
	StartedAt string `json:"started_at"`
	EndedAt   string `json:"ended_at"`
}

type shortcutDailySummary struct {
	Day             string  `json:"day"`
	MoveKcal        float64 `json:"move_kcal"`
	MoveGoalKcal    float64 `json:"move_goal_kcal"`
	ExerciseMin     float64 `json:"exercise_min"`
	ExerciseGoalMin float64 `json:"exercise_goal_min"`
	StandHours      float64 `json:"stand_hours"`
	StandGoalHours  float64 `json:"stand_goal_hours"`
}

type shortcutDailyMetric struct {
	Day    string  `json:"day"`
	Metric string  `json:"metric"`
	Value  float64 `json:"value"`
	Unit   string  `json:"unit"`
}

func ParseAppleHealthShortcut(path, rawHash string) (provider.ImportBatch, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return provider.ImportBatch{}, err
	}
	envelope, _, err := decodeAppleHealthShortcut(body)
	if err != nil {
		return provider.ImportBatch{}, err
	}

	batch := provider.ImportBatch{
		Activities: make([]observations.Activity, 0, len(envelope.Activities)),
		Sleep:      make([]observations.Sleep, 0, len(envelope.Sleep)),
		Daily: provider.DailyObservations{
			Summaries: make([]observations.DailySummary, 0, len(envelope.DailySummaries)),
			Metrics:   make([]observations.DailyMetric, 0, len(envelope.DailyMetrics)),
		},
		Coverage: make([]provider.CoverageAssertion, 0, len(envelope.Coverage)),
	}

	for _, item := range envelope.Coverage {
		if item.Category == "" || item.Timezone == "" || item.Completeness == "" {
			return provider.ImportBatch{}, errors.New("shortcut coverage must identify a bounded result")
		}
		location, err := time.LoadLocation(item.Timezone)
		if err != nil {
			return provider.ImportBatch{}, fmt.Errorf("coverage timezone: %w", err)
		}
		from, err := parseInstant(item.From, location)
		if err != nil {
			return provider.ImportBatch{}, fmt.Errorf("coverage from: %w", err)
		}
		to, err := parseInstant(item.To, location)
		if err != nil || !from.Before(to) {
			return provider.ImportBatch{}, errors.New("coverage interval must be half-open and non-empty")
		}
		if item.Completeness != "unknown" && item.Completeness != "partial" && item.Completeness != "covered" && item.Completeness != "covered_empty" {
			return provider.ImportBatch{}, fmt.Errorf("unsupported shortcut completeness %q", item.Completeness)
		}
		scope := item.Scope
		if len(scope) == 0 {
			scope = json.RawMessage(`{}`)
		}
		var object map[string]any
		if err := json.Unmarshal(scope, &object); err != nil || object == nil {
			return provider.ImportBatch{}, errors.New("shortcut coverage scope must be an object")
		}
		batch.Coverage = append(batch.Coverage, provider.CoverageAssertion{
			Category: item.Category, ScopeJSON: scope, From: from, To: to,
			Timezone: item.Timezone, IngestionMode: "bounded_replacement", Completeness: item.Completeness,
		})
	}

	for _, item := range envelope.Activities {
		if item.ID == "" || item.SportType == "" || item.StartedAt == "" || item.EndedAt == "" {
			return provider.ImportBatch{}, errors.New("shortcut activity requires id, sport_type, and time bounds")
		}
		startedAt, err := parseInstant(item.StartedAt, time.UTC)
		if err != nil {
			return provider.ImportBatch{}, err
		}
		endedAt, err := parseInstant(item.EndedAt, time.UTC)
		if err != nil || !startedAt.Before(endedAt) {
			return provider.ImportBatch{}, errors.New("shortcut activity interval is invalid")
		}
		batch.Activities = append(batch.Activities, observations.Activity{
			Provider: "apple_health", ExternalID: item.ID, SourceActivityID: item.ID,
			SportType: item.SportType, Title: item.Title, StartedAt: startedAt, EndedAt: &endedAt,
			DistanceM: item.DistanceM, DurationS: item.DurationS, AvgHR: item.AvgHR,
			MaxHR: item.MaxHR, CaloriesKcal: item.CaloriesKcal, SourceKind: KindAppleHealthShortcut,
			ContentHash: rawHash,
		})
	}

	for _, item := range envelope.Sleep {
		location := time.UTC
		wakeDate, err := parseInstant(item.WakeDate, location)
		if err != nil {
			return provider.ImportBatch{}, err
		}
		startedAt, err := parseInstant(item.StartedAt, location)
		if err != nil {
			return provider.ImportBatch{}, err
		}
		endedAt, err := parseInstant(item.EndedAt, location)
		if err != nil || !startedAt.Before(endedAt) {
			return provider.ImportBatch{}, errors.New("shortcut sleep interval is invalid")
		}
		session := observations.Sleep{WakeDate: wakeDate, StartedAt: startedAt, EndedAt: endedAt, TimeInBedS: item.TimeInBedS, AsleepS: item.AsleepS, Efficiency: item.Efficiency, IsMainSleep: item.IsMainSleep, CoreS: item.CoreS, DeepS: item.DeepS, RemS: item.RemS, AwakeS: item.AwakeS, UnspecifiedS: item.UnspecifiedS, Source: KindAppleHealthShortcut}
		for _, segment := range item.Segments {
			segmentStart, segmentErr := parseInstant(segment.StartedAt, time.UTC)
			segmentEnd, endErr := parseInstant(segment.EndedAt, time.UTC)
			if segmentErr != nil || endErr != nil || segment.Stage == "" || !segmentStart.Before(segmentEnd) {
				return provider.ImportBatch{}, errors.New("shortcut sleep segment is invalid")
			}
			session.Segments = append(session.Segments, observations.SleepSegment{Stage: segment.Stage, StartedAt: segmentStart, EndedAt: segmentEnd, Source: KindAppleHealthShortcut})
		}
		batch.Sleep = append(batch.Sleep, session)
	}

	for _, item := range envelope.DailySummaries {
		day, err := parseInstant(item.Day, time.UTC)
		if err != nil {
			return provider.ImportBatch{}, err
		}
		batch.Daily.Summaries = append(batch.Daily.Summaries, observations.DailySummary{Day: day, MoveKcal: item.MoveKcal, MoveGoalKcal: item.MoveGoalKcal, ExerciseMin: item.ExerciseMin, ExerciseGoalMin: item.ExerciseGoalMin, StandHours: item.StandHours, StandGoalHours: item.StandGoalHours, Source: KindAppleHealthShortcut})
	}
	for _, item := range envelope.DailyMetrics {
		day, err := parseInstant(item.Day, time.UTC)
		if err != nil || item.Metric == "" || item.Unit == "" {
			return provider.ImportBatch{}, errors.New("shortcut daily metric is invalid")
		}
		batch.Daily.Metrics = append(batch.Daily.Metrics, observations.DailyMetric{Day: day, Metric: item.Metric, Value: item.Value, Unit: item.Unit, Source: KindAppleHealthShortcut})
	}
	return batch, nil
}

func ValidateAppleHealthShortcut(body []byte) (AppleHealthShortcutMetadata, error) {
	_, metadata, err := decodeAppleHealthShortcut(body)
	return metadata, err
}

func decodeAppleHealthShortcut(body []byte) (shortcutEnvelope, AppleHealthShortcutMetadata, error) {
	var envelope shortcutEnvelope
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return shortcutEnvelope{}, AppleHealthShortcutMetadata{}, fmt.Errorf("decode shortcut payload: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return shortcutEnvelope{}, AppleHealthShortcutMetadata{}, errors.New("shortcut payload has trailing data")
		}
		return shortcutEnvelope{}, AppleHealthShortcutMetadata{}, fmt.Errorf("decode shortcut payload: %w", err)
	}
	if envelope.Schema != AppleHealthShortcutSchema || envelope.SourceInstanceKey == "" || envelope.CapturedAt == "" || len(envelope.Coverage) == 0 {
		return shortcutEnvelope{}, AppleHealthShortcutMetadata{}, errors.New("invalid shortcut payload envelope")
	}
	if err := validateShortcutCoverage(envelope.Coverage); err != nil {
		return shortcutEnvelope{}, AppleHealthShortcutMetadata{}, err
	}
	capturedAt, err := parseInstant(envelope.CapturedAt, time.UTC)
	if err != nil {
		return shortcutEnvelope{}, AppleHealthShortcutMetadata{}, fmt.Errorf("invalid captured_at: %w", err)
	}
	return envelope, AppleHealthShortcutMetadata{SourceInstanceKey: envelope.SourceInstanceKey, CapturedAt: capturedAt}, nil
}

func validateShortcutCoverage(items []shortcutCoverage) error {
	for _, item := range items {
		if item.Category == "" || item.Timezone == "" || item.Completeness == "" {
			return errors.New("shortcut coverage must identify a bounded result")
		}
		location, err := time.LoadLocation(item.Timezone)
		if err != nil {
			return fmt.Errorf("coverage timezone: %w", err)
		}
		from, err := parseInstant(item.From, location)
		if err != nil {
			return fmt.Errorf("coverage from: %w", err)
		}
		to, err := parseInstant(item.To, location)
		if err != nil || !from.Before(to) {
			return errors.New("coverage interval must be half-open and non-empty")
		}
		if item.Completeness != "unknown" && item.Completeness != "partial" && item.Completeness != "covered" && item.Completeness != "covered_empty" {
			return fmt.Errorf("unsupported shortcut completeness %q", item.Completeness)
		}
		scope := item.Scope
		if len(scope) == 0 {
			scope = json.RawMessage(`{}`)
		}
		var object map[string]any
		if err := json.Unmarshal(scope, &object); err != nil || object == nil {
			return errors.New("shortcut coverage scope must be an object")
		}
	}
	return nil
}

func parseInstant(value string, location *time.Location) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, location)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time %q", value)
	}
	return parsed.UTC(), nil
}
