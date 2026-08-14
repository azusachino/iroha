//go:build integration

package httpapi

import (
	"reflect"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/expenses"
	"github.com/google/uuid"
)

func TestIntegrationMonthlyReportCrossDomainBoundariesAndStability(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)
	resetMediaTables(t, db)
	for _, statement := range []string{
		"delete from tb_sleep_sessions where wake_date >= '2099-03-01' and wake_date < '2099-04-01'",
		"delete from tb_daily_metrics where day >= '2099-03-01' and day < '2099-04-01'",
		"delete from tb_daily_summaries where day >= '2099-03-01' and day < '2099-04-01'",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("reset cross-domain fixture: %v", err)
		}
	}
	server := newIntegrationServer(t, db)

	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load timezone: %v", err)
	}
	insideInstant := time.Date(2099, time.March, 1, 0, 30, 0, 0, location)
	boundaryInstant := time.Date(2099, time.April, 1, 0, 0, 0, 0, location)
	insideDate := time.Date(2099, time.March, 1, 0, 0, 0, 0, time.UTC)
	boundaryDate := time.Date(2099, time.April, 1, 0, 0, 0, 0, time.UTC)
	now := time.Now().UTC()

	rawID := uuid.New()
	if err := db.Create(&models.RawFile{
		ID: rawID, SHA256: "cross-domain-report-" + rawID.String(), OriginalFilename: "report-test.xml",
		StoragePath: "/tmp/report-test.xml", SourceKind: "test", UploadedVia: "test", CreatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create raw file: %v", err)
	}

	distance := 1000.0
	duration := 600
	insideActivity := models.Activity{ID: uuid.New(), SportType: "run", Title: "inside", StartedAt: insideInstant, DistanceM: &distance, DurationS: &duration, SourceKind: "test", FirstRawFileID: rawID, CreatedAt: now, UpdatedAt: now}
	boundaryActivity := models.Activity{ID: uuid.New(), SportType: "bike", Title: "boundary", StartedAt: boundaryInstant, DistanceM: &distance, DurationS: &duration, SourceKind: "test", FirstRawFileID: rawID, CreatedAt: now, UpdatedAt: now}
	for _, row := range []models.Activity{insideActivity, boundaryActivity} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create activity: %v", err)
		}
	}

	insideSleep := models.SleepSession{ID: uuid.New(), WakeDate: insideDate, StartedAt: insideInstant.Add(-8 * time.Hour), EndedAt: insideInstant, TimeInBedS: 28800, AsleepS: 25200, Efficiency: 0.875, IsMainSleep: true, CoreS: 12000, DeepS: 6000, RemS: 7200, AwakeS: 3600, Source: "test", FirstRawFileID: rawID, CreatedAt: now, UpdatedAt: now}
	boundarySleep := models.SleepSession{ID: uuid.New(), WakeDate: boundaryDate, StartedAt: boundaryInstant.Add(-8 * time.Hour), EndedAt: boundaryInstant, TimeInBedS: 28800, AsleepS: 25200, Efficiency: 0.875, IsMainSleep: true, Source: "test", FirstRawFileID: rawID, CreatedAt: now, UpdatedAt: now}
	for _, row := range []models.SleepSession{insideSleep, boundarySleep} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create sleep: %v", err)
		}
	}

	insideSummary := models.DailySummary{ID: uuid.New(), Day: insideDate, MoveKcal: 500, MoveGoalKcal: 600, Source: "test", FirstRawFileID: rawID, CreatedAt: now, UpdatedAt: now}
	boundarySummary := models.DailySummary{ID: uuid.New(), Day: boundaryDate, MoveKcal: 900, MoveGoalKcal: 600, Source: "test", FirstRawFileID: rawID, CreatedAt: now, UpdatedAt: now}
	for _, row := range []models.DailySummary{insideSummary, boundarySummary} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create daily summary: %v", err)
		}
	}
	metrics := []models.DailyMetric{
		{ID: uuid.New(), Day: insideDate, Metric: "steps", Value: 1000, Unit: "count", Source: "test", FirstRawFileID: rawID, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Day: time.Date(2099, time.March, 2, 0, 0, 0, 0, time.UTC), Metric: "steps", Value: 1.5, Unit: "km", Source: "test", FirstRawFileID: rawID, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Day: boundaryDate, Metric: "steps", Value: 9999, Unit: "count", Source: "test", FirstRawFileID: rawID, CreatedAt: now, UpdatedAt: now},
	}
	for _, row := range metrics {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create daily metric: %v", err)
		}
	}

	workID := seedWork(t, db, "cross-domain")
	itemID := seedItem(t, db, workID, "book", "inside book")
	seedProgress(t, db, itemID, "completed", ptrTime(time.Date(2099, time.March, 2, 0, 0, 0, 0, location)))
	seedFinishEvent(t, db, itemID, insideInstant)
	boundaryMediaItem := seedItem(t, db, workID, "anime_season", "boundary media")
	if err := db.Exec(`insert into tb_media_consumption_events
		(id, media_item_id, event_type, event_at, source_kind, created_at)
		values (?, ?, 'finished', ?, 'test', ?)`, uuid.New(), boundaryMediaItem, boundaryInstant, now).Error; err != nil {
		t.Fatalf("create boundary media event: %v", err)
	}

	expenseService := expenses.NewService(db)
	createExpense := func(ref, currency, category string, amount int64, date time.Time) {
		t.Helper()
		if _, err := expenseService.Create(expenses.CreateInput{
			OccurredOn: date, Currency: currency, AmountMinor: amount, Category: category,
			Source: expenses.Source{Kind: "cross-domain-test", Ref: ref},
		}); err != nil {
			t.Fatalf("create expense %s: %v", ref, err)
		}
	}
	createExpense("inside-jpy", "JPY", "food", 1300, insideDate)
	createExpense("inside-usd", "USD", "food", 2500, insideDate)
	createExpense("boundary-jpy", "JPY", "food", 9999, boundaryDate)

	path := "/api/v1/reports/monthly?month=2099-03&timezone=America%2FNew_York"
	first := requestJSON(t, server, "GET", path, "", 200, nil)
	second := requestJSON(t, server, "GET", path, "", 200, nil)
	delete(first, "generated_at")
	delete(second, "generated_at")
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("report changed apart from generated_at\nfirst=%#v\nsecond=%#v", first, second)
	}

	sections := first["sections"].(map[string]any)
	movement := sections["movement"].(map[string]any)["data"].(map[string]any)
	if movement["activity_count"] != float64(1) || movement["distance_m"] != float64(1000) {
		t.Fatalf("movement = %#v", movement)
	}
	sleep := sections["sleep"].(map[string]any)["data"].(map[string]any)
	if sleep["session_count"] != float64(1) {
		t.Fatalf("sleep = %#v", sleep)
	}
	daily := sections["daily_health"].(map[string]any)["data"].(map[string]any)
	reportMetrics := daily["metric_averages"].([]any)
	if daily["observed_days"] != float64(2) || len(reportMetrics) != 2 {
		t.Fatalf("daily = %#v", daily)
	}
	if reportMetrics[0].(map[string]any)["unit"] != "count" || reportMetrics[1].(map[string]any)["unit"] != "km" {
		t.Fatalf("daily metric units = %#v", reportMetrics)
	}
	media := sections["media"].(map[string]any)["data"].(map[string]any)
	if media["event_count"] != float64(1) || media["completed_count"] != float64(1) {
		t.Fatalf("media = %#v", media)
	}
	expense := sections["expenses"].(map[string]any)["data"].(map[string]any)
	currencyTotals := expense["totals_by_currency"].([]any)
	if expense["expense_count"] != float64(2) || len(currencyTotals) != 2 {
		t.Fatalf("expenses = %#v", expense)
	}
	if currencyTotals[0].(map[string]any)["currency"] != "JPY" || currencyTotals[0].(map[string]any)["amount_minor"] != float64(1300) || currencyTotals[1].(map[string]any)["currency"] != "USD" || currencyTotals[1].(map[string]any)["amount_minor"] != float64(2500) {
		t.Fatalf("currency totals = %#v", currencyTotals)
	}
}
