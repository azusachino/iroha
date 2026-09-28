package parsers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
)

const (
	DefaultHealthAutoExportInstance = "iphone-hae:primary"
	// haeTimeLayout is the only timestamp form Format v2 emits; it always
	// carries the device's UTC offset.
	haeTimeLayout = "2006-01-02 15:04:05 -0700"
	// kilojoulesPerKilocalorie converts energy reported in kJ.
	kilojoulesPerKilocalorie = 4.184
)

type HealthAutoExportMetadata struct {
	SourceInstanceKey string
	CapturedAt        time.Time
}

type haeExport struct {
	Data haeData `json:"data"`
}

type haeData struct {
	Metrics  []haeMetric  `json:"metrics"`
	Workouts []haeWorkout `json:"workouts"`
}

type haeMetric struct {
	Name  string           `json:"name"`
	Units string           `json:"units"`
	Data  []map[string]any `json:"data"`
}

type haeQuantity struct {
	Qty   float64 `json:"qty"`
	Units string  `json:"units"`
}

type haeHeartRateSummary struct {
	Min *haeQuantity `json:"min"`
	Avg *haeQuantity `json:"avg"`
	Max *haeQuantity `json:"max"`
}

type haeHeartRatePoint struct {
	Date   string  `json:"date"`
	Min    float64 `json:"Min"`
	Avg    float64 `json:"Avg"`
	Max    float64 `json:"Max"`
	Units  string  `json:"units"`
	Source string  `json:"source"`
}

type haeRoutePoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
	Speed     float64 `json:"speed"`
	Timestamp string  `json:"timestamp"`
}

type haeWorkout struct {
	ID                 string               `json:"id"`
	Name               string               `json:"name"`
	Start              string               `json:"start"`
	End                string               `json:"end"`
	Duration           float64              `json:"duration"`
	Distance           *haeQuantity         `json:"distance"`
	ActiveEnergyBurned *haeQuantity         `json:"activeEnergyBurned"`
	TotalEnergy        *haeQuantity         `json:"totalEnergy"`
	HeartRate          *haeHeartRateSummary `json:"heartRate"`
	HeartRateData      []haeHeartRatePoint  `json:"heartRateData"`
	Route              []haeRoutePoint      `json:"route"`
}

func parseHaeTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty timestamp")
	}
	t, err := time.Parse(haeTimeLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("timestamp %q is not Format v2 (yyyy-MM-dd HH:mm:ss Z)", s)
	}
	return t, nil
}

func ValidateHealthAutoExport(body []byte) (HealthAutoExportMetadata, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	var export haeExport
	if err := decoder.Decode(&export); err != nil {
		return HealthAutoExportMetadata{}, fmt.Errorf("invalid Health Auto Export JSON: %w", err)
	}

	if export.Data.Metrics == nil && export.Data.Workouts == nil {
		return HealthAutoExportMetadata{}, errors.New("health auto export payload must contain data.metrics or data.workouts")
	}
	for i, metric := range export.Data.Metrics {
		if metric.Name == "" {
			return HealthAutoExportMetadata{}, fmt.Errorf("data.metrics[%d] has no name", i)
		}
	}
	// Format v2 always gives a workout an id; Version 1 does not.
	for i, workout := range export.Data.Workouts {
		if workout.ID == "" {
			return HealthAutoExportMetadata{}, fmt.Errorf("data.workouts[%d] has no id; only Health Auto Export Format v2 is supported", i)
		}
		if _, err := parseHaeTime(workout.Start); err != nil {
			return HealthAutoExportMetadata{}, fmt.Errorf("data.workouts[%d].start: %w", i, err)
		}
	}

	return HealthAutoExportMetadata{
		SourceInstanceKey: DefaultHealthAutoExportInstance,
		CapturedAt:        time.Now().UTC(),
	}, nil
}

// ParseHealthAutoExport parses a Format v2 payload. loc is the effective
// timezone that labels the coverage window; daily facts keep their device-local day.
func ParseHealthAutoExport(path, rawHash string, loc *time.Location) (provider.ImportBatch, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return provider.ImportBatch{}, err
	}

	var export haeExport
	if err := json.Unmarshal(body, &export); err != nil {
		return provider.ImportBatch{}, fmt.Errorf("decode Health Auto Export: %w", err)
	}

	batch := provider.ImportBatch{
		Activities: make([]observations.Activity, 0, len(export.Data.Workouts)),
		Sleep:      make([]observations.Sleep, 0),
		Daily: provider.DailyObservations{
			Summaries: make([]observations.DailySummary, 0),
			Metrics:   make([]observations.DailyMetric, 0),
		},
		Coverage: make([]provider.CoverageAssertion, 0),
	}

	var (
		minObserved time.Time
		maxObserved time.Time
		hasObserved bool
	)

	// markObserved widens the window to the whole device-local day containing t,
	// so a day summary stamped at its device midnight covers all 24 hours.
	markObserved := func(t time.Time) {
		if t.IsZero() {
			return
		}
		dayStart, dayEnd := dayBounds(t)
		if !hasObserved {
			minObserved = dayStart
			maxObserved = dayEnd
			hasObserved = true
			return
		}
		if dayStart.Before(minObserved) {
			minObserved = dayStart
		}
		if dayEnd.After(maxObserved) {
			maxObserved = dayEnd
		}
	}

	// 1. Process Metrics
	for _, metric := range export.Data.Metrics {
		if metric.Name == "sleep_analysis" {
			sleepSessions, sleepMin, sleepMax, err := parseHaeSleep(metric)
			if err != nil {
				return provider.ImportBatch{}, err
			}
			batch.Sleep = append(batch.Sleep, sleepSessions...)
			markObserved(sleepMin)
			markObserved(sleepMax)
			continue
		}

		dailyMetrics, metricMin, metricMax, err := parseHaeDailyMetric(metric)
		if err != nil {
			return provider.ImportBatch{}, err
		}
		batch.Daily.Metrics = append(batch.Daily.Metrics, dailyMetrics...)
		markObserved(metricMin)
		markObserved(metricMax)
	}

	// 2. Process Workouts
	for _, workout := range export.Data.Workouts {
		activity, err := parseHaeWorkout(workout, rawHash)
		if err != nil {
			return provider.ImportBatch{}, err
		}
		batch.Activities = append(batch.Activities, activity)
		markObserved(activity.StartedAt)
		if activity.EndedAt != nil {
			markObserved(*activity.EndedAt)
		}
	}

	// 3. Compute Bounded Coverage Assertion
	if hasObserved {
		fromDay, _ := dayBounds(minObserved.In(loc))
		toDay := maxObserved.In(loc)
		if start, end := dayBounds(toDay); !toDay.Equal(start) {
			toDay = end
		}

		batch.Coverage = append(batch.Coverage, provider.CoverageAssertion{
			Category:      "health",
			ScopeJSON:     json.RawMessage(`{}`),
			From:          fromDay,
			To:            toDay,
			Timezone:      loc.String(),
			IngestionMode: "bounded_replacement",
			Completeness:  "unknown",
		})
	}

	return batch, nil
}

// dayBounds returns the calendar day containing t in t's own location.
func dayBounds(t time.Time) (time.Time, time.Time) {
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return start, start.AddDate(0, 0, 1)
}

func parseHaeSleep(metric haeMetric) ([]observations.Sleep, time.Time, time.Time, error) {
	var (
		sessions []observations.Sleep
		segments []observations.SleepSegment
		minTime  time.Time
		maxTime  time.Time
	)

	trackTime := func(t time.Time) {
		if t.IsZero() {
			return
		}
		if minTime.IsZero() || t.Before(minTime) {
			minTime = t
		}
		if maxTime.IsZero() || t.After(maxTime) {
			maxTime = t
		}
	}

	for _, pt := range metric.Data {
		// Stage segment (startDate, endDate, value)
		if startStr, ok := pt["startDate"].(string); ok {
			endStr, _ := pt["endDate"].(string)
			valStr, _ := pt["value"].(string)
			start, err := parseHaeTime(startStr)
			if err != nil {
				return nil, minTime, maxTime, fmt.Errorf("sleep_analysis startDate: %w", err)
			}
			end, err := parseHaeTime(endStr)
			if err != nil {
				return nil, minTime, maxTime, fmt.Errorf("sleep_analysis endDate: %w", err)
			}
			trackTime(start)
			trackTime(end)
			stage := normalizeHaeSleepStage(valStr)
			segments = append(segments, observations.SleepSegment{
				Stage:     stage,
				StartedAt: start,
				EndedAt:   end,
				Source:    KindHealthAutoExport,
			})
			continue
		}

		// Night aggregate (Summarize Data ON)
		startStr, _ := pt["sleepStart"].(string)
		endStr, _ := pt["sleepEnd"].(string)
		startedAt, err := parseHaeTime(startStr)
		if err != nil {
			return nil, minTime, maxTime, fmt.Errorf("sleep_analysis sleepStart: %w", err)
		}
		endedAt, err := parseHaeTime(endStr)
		if err != nil {
			return nil, minTime, maxTime, fmt.Errorf("sleep_analysis sleepEnd: %w", err)
		}

		trackTime(startedAt)
		trackTime(endedAt)

		// WakeDate derived from morning ending
		wakeDate := time.Date(endedAt.Year(), endedAt.Month(), endedAt.Day(), 0, 0, 0, 0, time.UTC)

		inBedH, _ := getFloat(pt, "inBed")
		asleepH, _ := getFloat(pt, "asleep")
		if asleepH == 0 {
			asleepH, _ = getFloat(pt, "totalSleep")
		}
		coreH, _ := getFloat(pt, "core")
		deepH, _ := getFloat(pt, "deep")
		remH, _ := getFloat(pt, "rem")
		awakeH, _ := getFloat(pt, "awake")

		timeInBedS := int(math.Round(inBedH * 3600))
		asleepS := int(math.Round(asleepH * 3600))
		if timeInBedS == 0 {
			timeInBedS = int(endedAt.Sub(startedAt).Seconds())
		}
		if asleepS == 0 {
			asleepS = timeInBedS
		}

		efficiency := 0.0
		if timeInBedS > 0 {
			efficiency = float64(asleepS) / float64(timeInBedS)
		}

		sessions = append(sessions, observations.Sleep{
			WakeDate:     wakeDate,
			StartedAt:    startedAt,
			EndedAt:      endedAt,
			TimeInBedS:   timeInBedS,
			AsleepS:      asleepS,
			Efficiency:   efficiency,
			IsMainSleep:  asleepS >= 3*3600,
			CoreS:        int(math.Round(coreH * 3600)),
			DeepS:        int(math.Round(deepH * 3600)),
			RemS:         int(math.Round(remH * 3600)),
			AwakeS:       int(math.Round(awakeH * 3600)),
			UnspecifiedS: 0,
			Source:       KindHealthAutoExport,
		})
	}

	// Summarize Data OFF sends only stage intervals, which cannot form a night.
	if len(sessions) == 0 && len(segments) > 0 {
		return nil, minTime, maxTime, errors.New("sleep_analysis has stage intervals but no nightly summary; enable Summarize Data in Health Auto Export")
	}

	// Attach segments to corresponding night session if inside [StartedAt, EndedAt]
	for i := range sessions {
		for _, seg := range segments {
			if !seg.StartedAt.Before(sessions[i].StartedAt.Add(-time.Hour)) && !seg.EndedAt.After(sessions[i].EndedAt.Add(time.Hour)) {
				sessions[i].Segments = append(sessions[i].Segments, seg)
			}
		}
	}

	return sessions, minTime, maxTime, nil
}

func parseHaeDailyMetric(metric haeMetric) ([]observations.DailyMetric, time.Time, time.Time, error) {
	var (
		results []observations.DailyMetric
		minTime time.Time
		maxTime time.Time
	)

	canonicalMetric, defaultUnit, isCumulative := mapHaeMetricName(metric.Name)
	if canonicalMetric == "" {
		return nil, minTime, maxTime, nil
	}

	unit := metric.Units
	if unit == "" {
		unit = defaultUnit
	}
	if unit == "" {
		unit = "count"
	}
	scale := 1.0
	if canonicalMetric == DailyMetricDistanceKM {
		switch strings.ToLower(unit) {
		case "mi", "miles":
			scale = 1.60934
		case "m":
			scale = 0.001
		}
		unit = "km"
	}

	trackTime := func(t time.Time) {
		if t.IsZero() {
			return
		}
		if minTime.IsZero() || t.Before(minTime) {
			minTime = t
		}
		if maxTime.IsZero() || t.After(maxTime) {
			maxTime = t
		}
	}

	// Group points by calendar day (YYYY-MM-DD)
	dailyValues := make(map[string]float64)
	dailyCounts := make(map[string]int)
	dayDates := make(map[string]time.Time)

	for _, pt := range metric.Data {
		dateStr, _ := pt["date"].(string)
		t, err := parseHaeTime(dateStr)
		if err != nil {
			return nil, minTime, maxTime, fmt.Errorf("metric %s: %w", metric.Name, err)
		}
		trackTime(t)

		dayKey := t.Format("2006-01-02")
		dayDates[dayKey] = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)

		val, ok := getFloat(pt, "qty")
		if !ok {
			val, ok = getFloat(pt, "Avg")
		}
		if !ok {
			val, _ = getFloat(pt, "value")
		}

		val *= scale

		if isCumulative {
			dailyValues[dayKey] += val
		} else {
			// gauge/point: running average
			dailyValues[dayKey] += val
			dailyCounts[dayKey]++
		}
	}

	for dayKey, val := range dailyValues {
		finalVal := val
		if !isCumulative && dailyCounts[dayKey] > 1 {
			finalVal = val / float64(dailyCounts[dayKey])
		}
		results = append(results, observations.DailyMetric{
			Day:    dayDates[dayKey],
			Metric: canonicalMetric,
			Value:  finalVal,
			Unit:   unit,
			Source: KindHealthAutoExport,
		})
	}

	return results, minTime, maxTime, nil
}

func parseHaeWorkout(w haeWorkout, rawHash string) (observations.Activity, error) {
	if w.ID == "" {
		return observations.Activity{}, errors.New("workout missing id")
	}
	startedAt, err := parseHaeTime(w.Start)
	if err != nil {
		return observations.Activity{}, fmt.Errorf("workout %s invalid start: %w", w.ID, err)
	}

	var endedAt *time.Time
	if w.End != "" {
		if e, err := parseHaeTime(w.End); err == nil {
			endedAt = &e
		}
	}

	sportType := normalizeHaeSport(w.Name)
	title := w.Name
	if title == "" {
		title = "Workout"
	}

	durationS := int(math.Round(w.Duration))
	var distanceM *float64
	if w.Distance != nil {
		d := w.Distance.Qty
		switch strings.ToLower(w.Distance.Units) {
		case "km":
			d *= 1000
		case "mi", "miles":
			d *= 1609.34
		}
		distanceM = &d
	}

	var caloriesKcal *float64
	if w.ActiveEnergyBurned != nil {
		c := w.ActiveEnergyBurned.Qty
		if strings.EqualFold(w.ActiveEnergyBurned.Units, "kJ") {
			c /= kilojoulesPerKilocalorie
		}
		caloriesKcal = &c
	}

	var avgHR *int
	var maxHR *int
	if w.HeartRate != nil {
		if w.HeartRate.Avg != nil {
			avg := int(math.Round(w.HeartRate.Avg.Qty))
			avgHR = &avg
		}
		if w.HeartRate.Max != nil {
			max := int(math.Round(w.HeartRate.Max.Qty))
			maxHR = &max
		}
	}

	// Parse GPS Route
	routePoints := make([]observations.RoutePoint, 0, len(w.Route))
	for _, rp := range w.Route {
		t, err := parseHaeTime(rp.Timestamp)
		if err != nil {
			t = startedAt
		}
		elev := rp.Altitude
		routePoints = append(routePoints, observations.RoutePoint{
			Ts:         &t,
			Lat:        rp.Latitude,
			Lon:        rp.Longitude,
			ElevationM: &elev,
		})
	}

	// Parse Heart Rate Samplings
	samplings := make([]observations.Sampling, 0, len(w.HeartRateData))
	for _, hr := range w.HeartRateData {
		t, err := parseHaeTime(hr.Date)
		if err != nil {
			continue
		}
		val := hr.Avg
		if val == 0 {
			val = hr.Min
		}
		samplings = append(samplings, observations.Sampling{
			SamplingType: "heart_rate",
			Ts:           t,
			Value:        val,
			Unit:         "bpm",
		})
	}

	return observations.Activity{
		Provider:         "apple_health",
		ExternalID:       w.ID,
		SourceActivityID: w.ID,
		SportType:        sportType,
		Title:            title,
		StartedAt:        startedAt,
		EndedAt:          endedAt,
		DistanceM:        distanceM,
		DurationS:        &durationS,
		AvgHR:            avgHR,
		MaxHR:            maxHR,
		CaloriesKcal:     caloriesKcal,
		RoutePoints:      routePoints,
		Samplings:        samplings,
		SourceKind:       KindHealthAutoExport,
		ContentHash:      rawHash,
	}, nil
}

func mapHaeMetricName(name string) (metric string, defaultUnit string, isCumulative bool) {
	switch name {
	case "step_count", "steps":
		return DailyMetricSteps, "count", true
	case "walking_running_distance", "distance":
		return DailyMetricDistanceKM, "km", true
	case "flights_climbed":
		return DailyMetricFlights, "count", true
	case "resting_heart_rate":
		return DailyMetricRestingHR, "bpm", false
	case "walking_heart_rate_average":
		return DailyMetricWalkingHR, "bpm", false
	case "heart_rate_variability_sdnn":
		return DailyMetricHRVSDNN, "ms", false
	case "vo2_max":
		return DailyMetricVO2Max, "ml/kg_min", false
	case "body_mass":
		return DailyMetricBodyMassKG, "kg", false
	case "oxygen_saturation":
		return DailyMetricSpO2Avg, "percent", false
	case "respiratory_rate":
		return DailyMetricRespiratoryRate, "count/min", false
	default:
		return "", "", false
	}
}

func normalizeHaeSleepStage(val string) string {
	switch strings.ToLower(val) {
	case "core":
		return SleepStageCore
	case "deep":
		return SleepStageDeep
	case "rem":
		return SleepStageREM
	case "awake":
		return SleepStageAwake
	case "inbed", "in_bed":
		return SleepStageInBed
	default:
		return SleepStageAsleepUnspecified
	}
}

func normalizeHaeSport(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "run"):
		return "running"
	case strings.Contains(n, "walk"), strings.Contains(n, "hike"):
		return "walking"
	case strings.Contains(n, "cycle"), strings.Contains(n, "bike"):
		return "cycling"
	case strings.Contains(n, "swim"):
		return "swimming"
	default:
		slug := strings.ReplaceAll(n, " ", "_")
		return strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
				return r
			}
			return -1
		}, slug)
	}
}

func getFloat(m map[string]any, key string) (float64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
