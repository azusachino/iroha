package parsers

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type appleWorkout struct {
	WorkoutActivityType string `xml:"workoutActivityType,attr"`
	StartDate           string `xml:"startDate,attr"`
	EndDate             string `xml:"endDate,attr"`
	Duration            string `xml:"duration,attr"`
	DurationUnit        string `xml:"durationUnit,attr"`
	TotalDistance       string `xml:"totalDistance,attr"`
	TotalDistanceUnit   string `xml:"totalDistanceUnit,attr"`
	SourceName          string `xml:"sourceName,attr"`
	SourceVersion       string `xml:"sourceVersion,attr"`
	Device              string `xml:"device,attr"`

	WorkoutRoute    *appleWorkoutRoute      `xml:"WorkoutRoute"`
	Statistics      []appleWorkoutStatistic `xml:"WorkoutStatistics"`
	Events          []appleWorkoutEvent     `xml:"WorkoutEvent"`
	MetadataEntries []appleMetadataEntry    `xml:"MetadataEntry"`
}

type appleWorkoutRoute struct {
	FileReference   *appleFileReference  `xml:"FileReference"`
	MetadataEntries []appleMetadataEntry `xml:"MetadataEntry"`
}

// syncIdentifier returns the WorkoutRoute's HKMetadataKeySyncIdentifier
// metadata value, if present. This is a stable identity for the route
// distinct from the workout's own identity, used to detect route
// additions/changes/removals via the workout content hash.
func (route *appleWorkoutRoute) syncIdentifier() string {
	if route == nil {
		return ""
	}
	for _, entry := range route.MetadataEntries {
		if entry.Key == "HKMetadataKeySyncIdentifier" {
			return entry.Value
		}
	}
	return ""
}

type appleFileReference struct {
	Path string `xml:"path,attr"`
}

type appleWorkoutStatistic struct {
	Type      string `xml:"type,attr"`
	Sum       string `xml:"sum,attr"`
	Average   string `xml:"average,attr"`
	Maximum   string `xml:"maximum,attr"`
	Minimum   string `xml:"minimum,attr"`
	Unit      string `xml:"unit,attr"`
	StartDate string `xml:"startDate,attr"`
	EndDate   string `xml:"endDate,attr"`
}

type appleWorkoutEvent struct {
	Type         string `xml:"type,attr"`
	Date         string `xml:"date,attr"`
	Duration     string `xml:"duration,attr"`
	DurationUnit string `xml:"durationUnit,attr"`
}

type appleMetadataEntry struct {
	Key   string `xml:"key,attr"`
	Value string `xml:"value,attr"`
}

func ParseAppleHealthExport(path string, rawHash string) ([]ParsedActivity, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	var activities []ParsedActivity
	for _, file := range reader.File {
		if !strings.HasSuffix(file.Name, "export.xml") {
			continue
		}

		workouts, err := parseAppleWorkoutsRaw(file)
		if err != nil {
			return nil, err
		}

		fileActivities := make([]ParsedActivity, 0, len(workouts))
		for _, workout := range workouts {
			activity, ok := workoutToActivity(workout)
			if !ok {
				continue
			}
			attachWorkoutRoute(&activity, workout, reader)
			fileActivities = append(fileActivities, activity)
		}

		// Pass 2: export.xml's document order is ExportDate, Me, the huge
		// <Record> block, THEN <Workout> elements - so records are seen
		// before workout windows exist. Re-open the same zip entry (do not
		// buffer the ~900MB file) and stream <Record> elements now that
		// fileActivities (and their windows) are known.
		//
		// buildWorkoutWindows takes pointers into fileActivities, so
		// nothing may be appended to fileActivities between here and where
		// it's copied into the outer activities slice below - an append
		// that grows past capacity would reallocate fileActivities' backing
		// array and silently orphan those pointers, losing any samplings
		// written through them.
		windows := buildWorkoutWindows(fileActivities)
		if len(windows) > 0 {
			samplingsFile, err := file.Open()
			if err != nil {
				return nil, err
			}
			err = decodeAppleSamplings(samplingsFile, windows)
			_ = samplingsFile.Close()
			if err != nil {
				return nil, err
			}
		}

		activities = append(activities, fileActivities...)
	}
	return activities, nil
}

// ParseAppleHealthSleep streams the sleep-analysis records from an Apple
// Health export and groups them into sessions. Sleep is intentionally parsed
// through its own path rather than being folded into ParsedActivity.
func ParseAppleHealthSleep(path string) ([]ParsedSleepSession, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	for _, file := range reader.File {
		if !strings.HasSuffix(file.Name, "export.xml") {
			continue
		}
		opened, err := file.Open()
		if err != nil {
			return nil, err
		}
		segments, decodeErr := decodeAppleSleepSegments(opened)
		_ = opened.Close()
		if decodeErr != nil {
			return nil, decodeErr
		}
		return sessionizeSleepSegments(segments, DefaultSleepSessionGap), nil
	}
	return nil, fmt.Errorf("apple health export.xml not found")
}

// ParseAppleHealthDailyActivity streams ActivitySummary rings and daily
// quantity records from an Apple Health export. Quantity records are retained
// only for the supported metrics and are deduplicated by source-priority
// interval union before being emitted as daily totals.
func ParseAppleHealthDailyActivity(path string) ([]ParsedDailySummary, []ParsedDailyMetric, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = reader.Close() }()

	for _, file := range reader.File {
		if !strings.HasSuffix(file.Name, "export.xml") {
			continue
		}
		opened, err := file.Open()
		if err != nil {
			return nil, nil, err
		}
		summaries, records, decodeErr := decodeAppleDailyActivity(opened)
		_ = opened.Close()
		if decodeErr != nil {
			return nil, nil, decodeErr
		}
		return summaries, rollupDailyMetrics(records), nil
	}
	return nil, nil, fmt.Errorf("apple health export.xml not found")
}

type dailyActivityRecord struct {
	day       string
	metric    string
	unit      string
	value     float64
	startedAt time.Time
	endedAt   time.Time
	source    string
}

type dailyInterval struct {
	start  time.Time
	end    time.Time
	value  float64
	source string
}

var selectedDailyRecordTypes = map[string]string{
	"HKQuantityTypeIdentifierStepCount":              DailyMetricSteps,
	"HKQuantityTypeIdentifierDistanceWalkingRunning": DailyMetricDistanceKM,
	"HKQuantityTypeIdentifierFlightsClimbed":         DailyMetricFlights,
}

func decodeAppleDailyActivity(r io.Reader) ([]ParsedDailySummary, []dailyActivityRecord, error) {
	decoder := xml.NewDecoder(r)
	summariesByDay := make(map[string]ParsedDailySummary)
	records := make([]dailyActivityRecord, 0)
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}

		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "ActivitySummary":
			var summary appleActivitySummary
			if err := decoder.DecodeElement(&summary, &start); err != nil {
				return nil, nil, err
			}
			parsed, ok := parseDailySummary(summary)
			if ok {
				summariesByDay[parsed.Day.Format("2006-01-02")] = parsed
			}
		case "Record":
			metric, selected := selectedDailyMetric(start)
			if !selected {
				if err := decoder.Skip(); err != nil {
					return nil, nil, err
				}
				continue
			}
			var record appleRecord
			if err := decoder.DecodeElement(&record, &start); err != nil {
				return nil, nil, err
			}
			parsed, ok := parseDailyRecord(record, metric)
			if ok {
				records = append(records, parsed)
			}
		}
	}

	days := make([]string, 0, len(summariesByDay))
	for day := range summariesByDay {
		days = append(days, day)
	}
	sort.Strings(days)
	summaries := make([]ParsedDailySummary, 0, len(days))
	for _, day := range days {
		summaries = append(summaries, summariesByDay[day])
	}
	return summaries, records, nil
}

type appleActivitySummary struct {
	DateComponents  string `xml:"dateComponents,attr"`
	MoveKcal        string `xml:"activeEnergyBurned,attr"`
	MoveGoalKcal    string `xml:"activeEnergyBurnedGoal,attr"`
	ExerciseMin     string `xml:"appleExerciseTime,attr"`
	ExerciseGoalMin string `xml:"appleExerciseTimeGoal,attr"`
	StandHours      string `xml:"appleStandHours,attr"`
	StandGoalHours  string `xml:"appleStandHoursGoal,attr"`
	Source          string `xml:"sourceName,attr"`
}

func parseDailySummary(summary appleActivitySummary) (ParsedDailySummary, bool) {
	day, err := parseAppleDate(summary.DateComponents)
	if err != nil {
		return ParsedDailySummary{}, false
	}
	parse := func(value string) float64 {
		parsed, _ := strconv.ParseFloat(value, 64)
		return parsed
	}
	source := summary.Source
	if source == "" {
		source = KindAppleHealthExport
	}
	return ParsedDailySummary{
		Day:             day,
		MoveKcal:        parse(summary.MoveKcal),
		MoveGoalKcal:    parse(summary.MoveGoalKcal),
		ExerciseMin:     parse(summary.ExerciseMin),
		ExerciseGoalMin: parse(summary.ExerciseGoalMin),
		StandHours:      parse(summary.StandHours),
		StandGoalHours:  parse(summary.StandGoalHours),
		Source:          source,
	}, true
}

func parseDailyRecord(record appleRecord, metric string) (dailyActivityRecord, bool) {
	startedAt, err := parseAppleTime(record.StartDate)
	if err != nil {
		return dailyActivityRecord{}, false
	}
	endedAt, err := parseAppleTime(record.EndDate)
	if err != nil || endedAt.Before(startedAt) {
		return dailyActivityRecord{}, false
	}
	value, err := strconv.ParseFloat(record.Value, 64)
	if err != nil {
		return dailyActivityRecord{}, false
	}
	if metric == DailyMetricDistanceKM && record.Unit == "mi" {
		value *= 1.609344
	}
	day, err := parseAppleDate(record.StartDate)
	if err != nil {
		return dailyActivityRecord{}, false
	}
	unit := record.Unit
	switch metric {
	case DailyMetricDistanceKM:
		unit = "km"
	case DailyMetricSteps, DailyMetricFlights:
		unit = "count"
	}
	return dailyActivityRecord{
		day:       day.Format("2006-01-02"),
		metric:    metric,
		unit:      unit,
		value:     value,
		startedAt: startedAt,
		endedAt:   endedAt,
		source:    record.SourceName,
	}, true
}

func selectedDailyMetric(start xml.StartElement) (string, bool) {
	for _, attr := range start.Attr {
		if attr.Name.Local == "type" {
			metric, ok := selectedDailyRecordTypes[attr.Value]
			return metric, ok
		}
	}
	return "", false
}

func rollupDailyMetrics(records []dailyActivityRecord) []ParsedDailyMetric {
	byDayMetric := make(map[string][]dailyActivityRecord)
	for _, record := range records {
		key := record.day + "\x00" + record.metric
		byDayMetric[key] = append(byDayMetric[key], record)
	}
	keys := make([]string, 0, len(byDayMetric))
	for key := range byDayMetric {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	metrics := make([]ParsedDailyMetric, 0, len(keys))
	for _, key := range keys {
		group := byDayMetric[key]
		sort.SliceStable(group, func(i, j int) bool {
			left, right := dailySourcePriority(group[i].source), dailySourcePriority(group[j].source)
			if left != right {
				return left > right
			}
			if !group[i].startedAt.Equal(group[j].startedAt) {
				return group[i].startedAt.Before(group[j].startedAt)
			}
			return group[i].endedAt.Before(group[j].endedAt)
		})
		accepted := make([]dailyInterval, 0, len(group))
		for _, record := range group {
			interval := dailyInterval{start: record.startedAt, end: record.endedAt, value: record.value, source: record.source}
			overlaps := false
			for _, existing := range accepted {
				if interval.start.Before(existing.end) && interval.end.After(existing.start) {
					overlaps = true
					break
				}
			}
			if !overlaps {
				accepted = append(accepted, interval)
			}
		}
		if len(accepted) == 0 {
			continue
		}
		day, _ := parseAppleDate(group[0].day)
		value := 0.0
		for _, interval := range accepted {
			value += interval.value
		}
		metrics = append(metrics, ParsedDailyMetric{
			Day:    day,
			Metric: group[0].metric,
			Value:  value,
			Unit:   group[0].unit,
			Source: accepted[0].source,
		})
	}
	return metrics
}

func dailySourcePriority(source string) int {
	source = strings.ToLower(source)
	switch {
	case strings.Contains(source, "watch"):
		return 2
	case strings.Contains(source, "iphone"):
		return 1
	default:
		return 0
	}
}

func parseAppleDate(value string) (time.Time, error) {
	if len(value) < len("2006-01-02") {
		return time.Time{}, fmt.Errorf("invalid Apple date %q", value)
	}
	return time.ParseInLocation("2006-01-02", value[:len("2006-01-02")], time.UTC)
}

func decodeAppleSleepSegments(r io.Reader) ([]ParsedSleepSegment, error) {
	decoder := xml.NewDecoder(r)
	segments := make([]ParsedSleepSegment, 0)
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "Record" {
			continue
		}
		stage, selected := selectedAppleSleepStage(start)
		if !selected {
			if err := decoder.Skip(); err != nil {
				return nil, err
			}
			continue
		}

		var record appleRecord
		if err := decoder.DecodeElement(&record, &start); err != nil {
			return nil, err
		}
		startedAt, err := parseAppleTime(record.StartDate)
		if err != nil {
			continue
		}
		endedAt, err := parseAppleTime(record.EndDate)
		if err != nil || endedAt.Before(startedAt) {
			continue
		}
		segments = append(segments, ParsedSleepSegment{
			Stage:     stage,
			StartedAt: startedAt,
			EndedAt:   endedAt,
			Source:    record.SourceName,
		})
	}

	sort.SliceStable(segments, func(i, j int) bool {
		return segments[i].StartedAt.Before(segments[j].StartedAt)
	})
	return segments, nil
}

func selectedAppleSleepStage(start xml.StartElement) (string, bool) {
	var recordType, value string
	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "type":
			recordType = attr.Value
		case "value":
			value = attr.Value
		}
	}
	if recordType != "HKCategoryTypeIdentifierSleepAnalysis" {
		return "", false
	}
	stage, ok := appleSleepStages[value]
	return stage, ok
}

func sessionizeSleepSegments(segments []ParsedSleepSegment, gap time.Duration) []ParsedSleepSession {
	if len(segments) == 0 {
		return nil
	}
	sorted := append([]ParsedSleepSegment(nil), segments...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].StartedAt.Before(sorted[j].StartedAt)
	})

	sessions := make([]ParsedSleepSession, 0)
	current := []ParsedSleepSegment{sorted[0]}
	currentEnd := sorted[0].EndedAt
	for _, segment := range sorted[1:] {
		if segment.StartedAt.Sub(currentEnd) <= gap {
			current = append(current, segment)
			if segment.EndedAt.After(currentEnd) {
				currentEnd = segment.EndedAt
			}
			continue
		}
		sessions = append(sessions, buildSleepSession(current))
		current = []ParsedSleepSegment{segment}
		currentEnd = segment.EndedAt
	}
	sessions = append(sessions, buildSleepSession(current))
	return sessions
}

func buildSleepSession(segments []ParsedSleepSegment) ParsedSleepSession {
	byStage := make(map[string][]sleepInterval)
	sources := make(map[string]struct{})
	for _, segment := range segments {
		byStage[segment.Stage] = append(byStage[segment.Stage], sleepInterval{start: segment.StartedAt, end: segment.EndedAt})
		if segment.Source != "" {
			sources[segment.Source] = struct{}{}
		}
	}

	startedAt := segments[0].StartedAt
	endedAt := segments[0].EndedAt
	for _, segment := range segments[1:] {
		if segment.EndedAt.After(endedAt) {
			endedAt = segment.EndedAt
		}
	}
	timeInBed := unionSleepSeconds(byStage[SleepStageInBed])
	if timeInBed == 0 {
		timeInBed = endedAt.Sub(startedAt).Seconds()
	}
	asleepIntervals := make([]sleepInterval, 0)
	for _, stage := range []string{SleepStageCore, SleepStageDeep, SleepStageREM, SleepStageAsleepUnspecified} {
		asleepIntervals = append(asleepIntervals, byStage[stage]...)
	}
	asleep := unionSleepSeconds(asleepIntervals)

	stageSeconds := func(stage string) int {
		return int(unionSleepSeconds(byStage[stage]))
	}
	timeInBedS := int(timeInBed)
	asleepS := int(asleep)
	efficiency := 0.0
	if timeInBed > 0 {
		efficiency = asleep / timeInBed
	}
	wakeDate := time.Date(endedAt.Year(), endedAt.Month(), endedAt.Day(), 0, 0, 0, 0, endedAt.Location())
	return ParsedSleepSession{
		WakeDate:     wakeDate,
		StartedAt:    startedAt,
		EndedAt:      endedAt,
		TimeInBedS:   timeInBedS,
		AsleepS:      asleepS,
		Efficiency:   efficiency,
		IsMainSleep:  asleep >= MainSleepThreshold.Seconds(),
		CoreS:        stageSeconds(SleepStageCore),
		DeepS:        stageSeconds(SleepStageDeep),
		RemS:         stageSeconds(SleepStageREM),
		AwakeS:       stageSeconds(SleepStageAwake),
		UnspecifiedS: stageSeconds(SleepStageAsleepUnspecified),
		Source:       strings.Join(sortedSleepSources(sources), ","),
		Segments:     append([]ParsedSleepSegment(nil), segments...),
	}
}

type sleepInterval struct {
	start time.Time
	end   time.Time
}

func unionSleepSeconds(intervals []sleepInterval) float64 {
	if len(intervals) == 0 {
		return 0
	}
	sorted := append([]sleepInterval(nil), intervals...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].start.Before(sorted[j].start) })
	start, end := sorted[0].start, sorted[0].end
	total := 0.0
	for _, interval := range sorted[1:] {
		if interval.start.After(end) {
			total += end.Sub(start).Seconds()
			start, end = interval.start, interval.end
			continue
		}
		if interval.end.After(end) {
			end = interval.end
		}
	}
	return total + end.Sub(start).Seconds()
}

func sortedSleepSources(sources map[string]struct{}) []string {
	values := make([]string, 0, len(sources))
	for source := range sources {
		values = append(values, source)
	}
	sort.Strings(values)
	return values
}

func parseAppleWorkoutsRaw(file *zip.File) ([]appleWorkout, error) {
	opened, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = opened.Close() }()

	return decodeAppleWorkoutsRaw(opened)
}

// attachWorkoutRoute locates the zip entry referenced by the workout's
// WorkoutRoute/FileReference (if any), parses its track points, and attaches
// them to activity.RoutePoints. Workouts without a route, or whose
// referenced gpx entry is missing from the zip, are left with empty
// RoutePoints - this must never fail the overall import.
func attachWorkoutRoute(activity *ParsedActivity, workout appleWorkout, reader *zip.ReadCloser) {
	if workout.WorkoutRoute == nil || workout.WorkoutRoute.FileReference == nil {
		return
	}
	refPath := workout.WorkoutRoute.FileReference.Path
	if refPath == "" {
		return
	}

	routeFile := findRouteZipFile(reader.File, refPath)
	if routeFile == nil {
		return
	}

	opened, err := routeFile.Open()
	if err != nil {
		return
	}
	defer func() { _ = opened.Close() }()

	points, err := parseGPXPoints(opened)
	if err != nil {
		return
	}
	activity.RoutePoints = points
}

// findRouteZipFile matches a WorkoutRoute FileReference path (which carries
// a leading slash, e.g. "/workout-routes/route_x.gpx") against the zip's
// entries by suffix, so it works regardless of the export directory prefix
// (e.g. "apple_health_export/workout-routes/route_x.gpx").
func findRouteZipFile(files []*zip.File, refPath string) *zip.File {
	trimmed := strings.TrimPrefix(refPath, "/")
	if trimmed == "" {
		return nil
	}
	for _, file := range files {
		if strings.HasSuffix(file.Name, trimmed) {
			return file
		}
	}
	return nil
}

// decodeAppleWorkouts streams through an export.xml document, decoding each
// <Workout> element's full subtree while skipping everything else (notably
// the potentially millions of sibling <Record> elements). It must remain
// streaming rather than reading the whole document into memory.
func decodeAppleWorkouts(r io.Reader) ([]ParsedActivity, error) {
	workouts, err := decodeAppleWorkoutsRaw(r)
	if err != nil {
		return nil, err
	}

	var activities []ParsedActivity
	for _, workout := range workouts {
		activity, ok := workoutToActivity(workout)
		if ok {
			activities = append(activities, activity)
		}
	}
	return activities, nil
}

// decodeAppleWorkoutsRaw performs the streaming token walk over an
// export.xml document and returns the decoded appleWorkout structs
// (including their nested subtrees) without converting them to
// ParsedActivity. Exposed separately so tests can assert on the captured
// nested data directly.
func decodeAppleWorkoutsRaw(r io.Reader) ([]appleWorkout, error) {
	decoder := xml.NewDecoder(r)
	var workouts []appleWorkout
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "Workout" {
			continue
		}

		var workout appleWorkout
		if err := decoder.DecodeElement(&workout, &start); err != nil {
			return nil, err
		}
		workouts = append(workouts, workout)
	}
	return workouts, nil
}

// selectedRecordTypes maps the HealthKit <Record type="..."> identifiers we
// capture as activity samplings to their sampling_type slug. Every other
// record type (steps, sleep, resting HR outside a workout, ...) is skipped
// without being decoded - this is the common case over ~2M records.
var selectedRecordTypes = map[string]string{
	"HKQuantityTypeIdentifierHeartRate":              "heart_rate",
	"HKQuantityTypeIdentifierRunningSpeed":           "running_speed",
	"HKQuantityTypeIdentifierRunningPower":           "running_power",
	"HKQuantityTypeIdentifierRunningStrideLength":    "stride_length",
	"HKQuantityTypeIdentifierDistanceWalkingRunning": "distance",
	"HKQuantityTypeIdentifierActiveEnergyBurned":     "active_energy",
}

// appleRecord decodes a single selected-type <Record> element. Only decoded
// once selectedRecordSamplingType has already confirmed the record's type
// attr is one we care about - unselected records are skipped via
// decoder.Skip() without ever being unmarshaled into this struct.
type appleRecord struct {
	Type       string `xml:"type,attr"`
	SourceName string `xml:"sourceName,attr"`
	Unit       string `xml:"unit,attr"`
	StartDate  string `xml:"startDate,attr"`
	EndDate    string `xml:"endDate,attr"`
	Value      string `xml:"value,attr"`
}

var appleSleepStages = map[string]string{
	"HKCategoryValueSleepAnalysisInBed":             SleepStageInBed,
	"HKCategoryValueSleepAnalysisAwake":             SleepStageAwake,
	"HKCategoryValueSleepAnalysisAsleepCore":        SleepStageCore,
	"HKCategoryValueSleepAnalysisAsleepDeep":        SleepStageDeep,
	"HKCategoryValueSleepAnalysisAsleepREM":         SleepStageREM,
	"HKCategoryValueSleepAnalysisAsleepUnspecified": SleepStageAsleepUnspecified,
}

// workoutWindow is a workout's [start, end] time span paired with a pointer
// into the activities slice it came from, used to associate pass-2 records
// back to the activity that will be persisted.
type workoutWindow struct {
	start    time.Time
	end      time.Time
	activity *ParsedActivity
}

// buildWorkoutWindows builds the sorted-by-start window list used to
// associate <Record> samples with their owning workout. Only activities with
// a valid EndedAt are included (an open-ended workout has no window to test
// against). Callers must not append to activities after calling this - see
// the caution at ParseAppleHealthExport's call site.
func buildWorkoutWindows(activities []ParsedActivity) []workoutWindow {
	windows := make([]workoutWindow, 0, len(activities))
	for i := range activities {
		if activities[i].EndedAt == nil {
			continue
		}
		windows = append(windows, workoutWindow{
			start:    activities[i].StartedAt,
			end:      *activities[i].EndedAt,
			activity: &activities[i],
		})
	}
	sort.Slice(windows, func(i, j int) bool { return windows[i].start.Before(windows[j].start) })
	return windows
}

// findOwningActivity binary-searches windows (which must be sorted by
// start) for the rightmost window whose start <= ts, then checks whether ts
// also falls at-or-before that window's end. Both endpoints are inclusive,
// so a record exactly on a workout's start or end boundary counts as
// inside. Returns nil if no window contains ts.
//
// If workout windows overlap (rare in practice), the rightmost
// start-matching window is used rather than searching earlier windows too -
// an accepted approximation per the task spec.
func findOwningActivity(windows []workoutWindow, ts time.Time) *ParsedActivity {
	idx := sort.Search(len(windows), func(i int) bool {
		return windows[i].start.After(ts)
	}) - 1
	if idx < 0 {
		return nil
	}
	w := windows[idx]
	if ts.Before(w.start) || ts.After(w.end) {
		return nil
	}
	return w.activity
}

// decodeAppleSamplings is pass 2 over export.xml: it streams top-level
// <Record> elements and, for each whose type is in selectedRecordTypes and
// whose startDate falls inside a workout window, appends a ParsedSampling
// to that window's activity (via the activity pointer captured in
// workoutWindow).
//
// This must stay cheap over ~2M records: the type attr is read directly off
// the raw xml.StartElement (no allocation), and any non-selected record is
// discarded via decoder.Skip() - a cheap subtree skip - without ever being
// unmarshaled. Only selected-type records pay for DecodeElement, and only
// those that also land inside a workout window are retained.
func decodeAppleSamplings(r io.Reader, windows []workoutWindow) error {
	decoder := xml.NewDecoder(r)
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "Record" {
			continue
		}

		samplingType, selected := selectedRecordSamplingType(start)
		if !selected {
			if err := decoder.Skip(); err != nil {
				return err
			}
			continue
		}

		var rec appleRecord
		if err := decoder.DecodeElement(&rec, &start); err != nil {
			return err
		}

		ts, err := parseAppleTime(rec.StartDate)
		if err != nil {
			continue
		}
		value, err := strconv.ParseFloat(rec.Value, 64)
		if err != nil {
			continue
		}

		activity := findOwningActivity(windows, ts)
		if activity == nil {
			continue
		}
		activity.Samplings = append(activity.Samplings, ParsedSampling{
			SamplingType: samplingType,
			Ts:           ts,
			Value:        value,
			Unit:         rec.Unit,
		})
	}
	return nil
}

// selectedRecordSamplingType reads the "type" attr straight off a raw
// <Record> StartElement, without decoding the element. This keeps the
// common (unselected) case to an attribute scan followed by decoder.Skip().
func selectedRecordSamplingType(start xml.StartElement) (string, bool) {
	for _, attr := range start.Attr {
		if attr.Name.Local == "type" {
			samplingType, ok := selectedRecordTypes[attr.Value]
			return samplingType, ok
		}
	}
	return "", false
}

func workoutToActivity(workout appleWorkout) (ParsedActivity, bool) {
	startedAt, err := parseAppleTime(workout.StartDate)
	if err != nil {
		return ParsedActivity{}, false
	}
	var endedAt *time.Time
	if parsed, err := parseAppleTime(workout.EndDate); err == nil {
		endedAt = &parsed
	}

	durationS := parseDurationSeconds(workout.Duration, workout.DurationUnit)
	distanceM := parseDistanceMeters(workout.TotalDistance, workout.TotalDistanceUnit)
	externalID := stableWorkoutSourceKey(workout)
	contentHash := workoutContentHash(workout)

	activity := ParsedActivity{
		Provider:         "apple_health",
		ExternalID:       externalID,
		SportType:        normalizeAppleSport(workout.WorkoutActivityType),
		Title:            strings.TrimPrefix(workout.WorkoutActivityType, "HKWorkoutActivityType"),
		StartedAt:        startedAt,
		EndedAt:          endedAt,
		DistanceM:        distanceM,
		DurationS:        durationS,
		SourceKind:       "apple_health_export",
		SourceActivityID: externalID,
		ContentHash:      contentHash,
	}

	applyWorkoutStatistics(&activity, workout.Statistics)
	activity.Laps = deriveWorkoutLaps(workout, startedAt)

	return activity, true
}

// applyWorkoutStatistics populates the workout-level summary fields
// (AvgHR/MaxHR/AvgPaceSPerKM, CaloriesKcal, and DistanceM when a more authoritative
// per-stat value is available) from the workout's WorkoutStatistics
// entries.
func applyWorkoutStatistics(activity *ParsedActivity, stats []appleWorkoutStatistic) {
	for _, stat := range stats {
		switch stat.Type {
		case "HKQuantityTypeIdentifierHeartRate":
			if avg, err := strconv.ParseFloat(stat.Average, 64); err == nil {
				rounded := int(math.Round(avg))
				activity.AvgHR = &rounded
			}
			if maxVal, err := strconv.ParseFloat(stat.Maximum, 64); err == nil {
				rounded := int(math.Round(maxVal))
				activity.MaxHR = &rounded
			}

		case "HKQuantityTypeIdentifierDistanceWalkingRunning":
			// The per-statistic distance is authoritative over the
			// workout-level totalDistance attribute when present.
			if distanceM := parseDistanceMeters(stat.Sum, stat.Unit); distanceM != nil {
				activity.DistanceM = distanceM
			}

		case "HKQuantityTypeIdentifierActiveEnergyBurned":
			if sumVal, err := strconv.ParseFloat(stat.Sum, 64); err == nil {
				activity.CaloriesKcal = &sumVal
			}
		}
	}

	if activity.DurationS != nil && activity.DistanceM != nil && *activity.DistanceM > 0 {
		pace := float64(*activity.DurationS) / (*activity.DistanceM / 1000.0)
		activity.AvgPaceSPerKM = &pace
	}
}

// deriveWorkoutLaps derives timing-only laps from the workout's
// lap-delimiting WorkoutEvents.
//
// Interpretation (a judgment call - flagged for review): HealthKit's
// HKWorkoutEventTypeLap/Segment events mark the *end* of a lap, not a
// standalone boundary. So for N such events, in chronological (document)
// order, this produces exactly N laps: lap K spans
// [boundary(K-1), event[K].Date], where boundary(0) is the workout's own
// StartDate. The workout's EndDate is NOT appended as a trailing boundary,
// so any tail after the last lap event (if the workout continued past its
// last recorded lap) is not represented as a lap. This mirrors how fitness
// devices commonly report "lap N completed at T" markers rather than
// separate start/end markers per lap.
//
// Only HKWorkoutEventTypeLap events are used when present; if the workout
// has none of those but has HKWorkoutEventTypeSegment events, those are
// used instead. Marker/Pause/Resume events never contribute lap
// boundaries. Workouts with no qualifying events produce no laps (nil).
func deriveWorkoutLaps(workout appleWorkout, startedAt time.Time) []ParsedLap {
	boundaryEvents := lapBoundaryEvents(workout.Events)
	if len(boundaryEvents) == 0 {
		return nil
	}

	var laps []ParsedLap
	prevBoundary := startedAt
	lapNo := 1
	for _, event := range boundaryEvents {
		eventDate, err := parseAppleTime(event.Date)
		if err != nil {
			// Guard against an unparseable date: skip this boundary
			// entirely rather than producing a bogus lap.
			continue
		}

		start := prevBoundary
		end := eventDate
		durationS := int(end.Sub(start).Seconds())
		laps = append(laps, ParsedLap{
			LapNo:     lapNo,
			StartTs:   &start,
			EndTs:     &end,
			DurationS: &durationS,
		})
		lapNo++
		prevBoundary = eventDate
	}

	return laps
}

// lapBoundaryEvents selects the WorkoutEvents that delimit laps:
// HKWorkoutEventTypeLap events if any are present, otherwise
// HKWorkoutEventTypeSegment events as a fallback. All other event types
// (Marker, Pause, Resume, ...) are ignored for lap purposes.
func lapBoundaryEvents(events []appleWorkoutEvent) []appleWorkoutEvent {
	var laps []appleWorkoutEvent
	var segments []appleWorkoutEvent
	for _, event := range events {
		switch event.Type {
		case "HKWorkoutEventTypeLap":
			laps = append(laps, event)
		case "HKWorkoutEventTypeSegment":
			segments = append(segments, event)
		}
	}
	if len(laps) > 0 {
		return laps
	}
	return segments
}

// stableWorkoutSourceKey builds an identity for an Apple Health workout that
// is stable ACROSS full-export re-exports of the same underlying workout.
// It deliberately does NOT include the export zip's sha256 (that changes on
// every export even when the workout itself hasn't). It combines the
// workout's source name, a normalized device fingerprint (see
// stableDeviceKey), activity type, start/end timestamps and duration -
// fields that together uniquely and durably identify a single workout
// recorded by a single device.
func stableWorkoutSourceKey(workout appleWorkout) string {
	return strings.Join([]string{
		workout.SourceName,
		stableDeviceKey(workout.Device),
		workout.WorkoutActivityType,
		workout.StartDate,
		workout.EndDate,
		workout.Duration,
	}, "|")
}

// deviceFieldRe extracts "name:" and "hardware:" values out of Apple's
// HKDevice description string, e.g.:
//
//	<<HKDevice: 0x8dcef18c0>, name:Apple Watch, manufacturer:Apple Inc.,
//	  model:Watch, hardware:Watch7,1, software:10.2, creation date:...>>
//
// The leading "0x..." token is a runtime pointer and "creation date:" is
// wall-clock capture time - both are volatile and differ between exports of
// the SAME physical device. Only "name" and "hardware" are stable, so those
// are the only fields pulled out.
var deviceFieldRe = regexp.MustCompile(`(?:^|,\s*)(name|hardware):([^,>]*)`)

// stableDeviceKey normalizes a raw Apple HKDevice description string into a
// fingerprint that is stable across exports of the same device, by
// extracting only the "name" and "hardware" fields and dropping the
// volatile pointer address and creation-date segment.
func stableDeviceKey(device string) string {
	if device == "" {
		return ""
	}
	matches := deviceFieldRe.FindAllStringSubmatch(device, -1)
	if len(matches) == 0 {
		return ""
	}
	values := make(map[string]string, len(matches))
	for _, m := range matches {
		values[m[1]] = strings.TrimSpace(m[2])
	}
	return fmt.Sprintf("name=%s;hardware=%s", values["name"], values["hardware"])
}

// workoutContentHash computes a sha256 hex digest over the change-relevant
// fields of a workout: its own attrs (with device normalized via
// stableDeviceKey, since the raw device string carries a volatile pointer
// and creation date that would otherwise make every re-export look
// "changed"), its WorkoutStatistics entries in document order, its
// WorkoutEvent entries in document order (laps are derived from these -
// see deriveWorkoutLaps), and its workout-level MetadataEntry values
// deduped by key (real exports can repeat metadata keys under one Workout)
// and sorted for determinism.
func workoutContentHash(workout appleWorkout) string {
	var b strings.Builder

	fmt.Fprintf(&b, "type=%s\n", workout.WorkoutActivityType)
	fmt.Fprintf(&b, "start=%s\n", workout.StartDate)
	fmt.Fprintf(&b, "end=%s\n", workout.EndDate)
	fmt.Fprintf(&b, "duration=%s\n", workout.Duration)
	fmt.Fprintf(&b, "durationUnit=%s\n", workout.DurationUnit)
	fmt.Fprintf(&b, "totalDistance=%s\n", workout.TotalDistance)
	fmt.Fprintf(&b, "totalDistanceUnit=%s\n", workout.TotalDistanceUnit)
	fmt.Fprintf(&b, "sourceName=%s\n", workout.SourceName)
	fmt.Fprintf(&b, "sourceVersion=%s\n", workout.SourceVersion)
	fmt.Fprintf(&b, "device=%s\n", stableDeviceKey(workout.Device))

	if workout.WorkoutRoute != nil && workout.WorkoutRoute.FileReference != nil {
		fmt.Fprintf(&b, "routePath=%s\n", workout.WorkoutRoute.FileReference.Path)
	}
	fmt.Fprintf(&b, "routeSyncId=%s\n", workout.WorkoutRoute.syncIdentifier())

	for _, stat := range workout.Statistics {
		fmt.Fprintf(&b, "stat:type=%s|sum=%s|avg=%s|max=%s|min=%s|unit=%s|start=%s|end=%s\n",
			stat.Type, stat.Sum, stat.Average, stat.Maximum, stat.Minimum, stat.Unit, stat.StartDate, stat.EndDate)
	}

	// Events are not otherwise reflected in the hashed workout attrs, but
	// laps are now derived from them (see deriveWorkoutLaps), so a change
	// in events must be treated as a content change even though the
	// workout-level attrs and statistics are unchanged.
	for _, event := range workout.Events {
		fmt.Fprintf(&b, "event:type=%s|date=%s|duration=%s|durationUnit=%s\n",
			event.Type, event.Date, event.Duration, event.DurationUnit)
	}

	metadata := make(map[string]string, len(workout.MetadataEntries))
	for _, entry := range workout.MetadataEntries {
		metadata[entry.Key] = entry.Value
	}
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(&b, "meta:%s=%s\n", key, metadata[key])
	}

	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func parseAppleTime(value string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05 -0700", value)
}

func parseDurationSeconds(value string, unit string) *int {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	switch unit {
	case "min":
		parsed *= 60
	case "hr":
		parsed *= 3600
	}
	seconds := int(parsed)
	return &seconds
}

func parseDistanceMeters(value string, unit string) *float64 {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	switch unit {
	case "km":
		parsed *= 1000
	case "mi":
		parsed *= 1609.344
	}
	return &parsed
}

func normalizeAppleSport(value string) string {
	switch value {
	case "HKWorkoutActivityTypeRunning":
		return "run"
	case "HKWorkoutActivityTypeWalking":
		return "walk"
	case "HKWorkoutActivityTypeCycling":
		return "ride"
	case "HKWorkoutActivityTypeHiking":
		return "hike"
	default:
		return strings.TrimPrefix(value, "HKWorkoutActivityType")
	}
}
