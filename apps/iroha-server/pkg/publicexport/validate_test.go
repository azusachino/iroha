package publicexport

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/media"
	"github.com/google/uuid"
)

func ptrTimePublic(t time.Time) *time.Time { return &t }

func TestPublicProjectionOmitsPrivateExpenseAndReportData(t *testing.T) {
	payload, err := json.Marshal(struct {
		Summary    activities.Summary     `json:"summary"`
		Activities []Activity             `json:"activities"`
		Routes     RouteFeatureCollection `json:"routes"`
	}{
		Summary:    validSummary(),
		Activities: []Activity{validActivity()},
		Routes:     RouteFeatureCollection{},
	})
	if err != nil {
		t.Fatalf("marshal public projection: %v", err)
	}
	serialized := string(payload)
	if strings.Contains(serialized, "expense") || strings.Contains(serialized, "report") {
		t.Fatalf("public projection leaked private domain data: %s", serialized)
	}
}

func validSummary() activities.Summary {
	return activities.Summary{
		Totals: activities.SummaryTotals{ActivityCount: 1, DistanceM: 100, DistanceKnownCount: 1, DurationS: 60},
	}
}

func validActivity() Activity {
	return Activity{ID: "act_" + "0198f8f0-0000-7000-8000-000000000000", StartedAt: time.Unix(0, 0)}
}

func TestValidate_OK(t *testing.T) {
	if err := Validate(validSummary(), []Activity{validActivity()}, RouteFeatureCollection{}, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestToActivityPreservesDerivedMetrics(t *testing.T) {
	distance := 1234.5
	duration := 3600
	moving := 3300
	elevation := 42.5
	avgHR := 145
	maxHR := 178
	pace := 330.0

	got := ToActivity(models.Activity{
		ID:             uuid.New(),
		SportType:      "run",
		StartedAt:      time.Unix(0, 0),
		DistanceM:      &distance,
		DurationS:      &duration,
		MovingTimeS:    &moving,
		ElevationGainM: &elevation,
		AvgHR:          &avgHR,
		MaxHR:          &maxHR,
		AvgPaceSPerKM:  &pace,
	})

	if got.MovingTimeS == nil || *got.MovingTimeS != moving {
		t.Fatalf("moving time = %v, want %d", got.MovingTimeS, moving)
	}
	if got.ElevationGainM == nil || *got.ElevationGainM != elevation {
		t.Fatalf("elevation gain = %v, want %.1f", got.ElevationGainM, elevation)
	}
}

func TestValidate_RejectsUnprefixedActivityID(t *testing.T) {
	activity := validActivity()
	activity.ID = "0198f8f0-0000-7000-8000-000000000000"
	if err := Validate(validSummary(), []Activity{activity}, RouteFeatureCollection{}, nil); err == nil {
		t.Fatal("expected error for unprefixed activity id")
	}
}

func TestValidate_RejectsNegativeTotals(t *testing.T) {
	summary := validSummary()
	summary.Totals.DistanceM = -1
	if err := Validate(summary, nil, RouteFeatureCollection{}, nil); err == nil {
		t.Fatal("expected error for negative totals")
	}
}

func TestValidate_RejectsEndedBeforeStarted(t *testing.T) {
	activity := validActivity()
	before := activity.StartedAt.Add(-time.Hour)
	activity.EndedAt = &before
	if err := Validate(validSummary(), []Activity{activity}, RouteFeatureCollection{}, nil); err == nil {
		t.Fatal("expected error for ended_at before started_at")
	}
}

func TestValidate_RejectsOutOfRangeCoordinate(t *testing.T) {
	routes := RouteFeatureCollection{Features: []RouteFeature{{
		Type: "Feature",
		Properties: RouteFeatureProps{
			ActivityID: "act_0198f8f0-0000-7000-8000-000000000000",
		},
		Geometry: RouteGeometry{Type: "LineString", Coordinates: [][2]float64{{200, 0}}},
	}}}
	if err := Validate(validSummary(), []Activity{validActivity()}, routes, nil); err == nil {
		t.Fatal("expected error for out-of-range coordinate")
	}
}

func TestValidate_RejectsUnlinkedRoute(t *testing.T) {
	routes := RouteFeatureCollection{Features: []RouteFeature{{
		Type:     "Feature",
		Geometry: RouteGeometry{Type: "LineString", Coordinates: [][2]float64{{139, 35}, {139.001, 35.001}}},
	}}}
	if err := Validate(validSummary(), []Activity{validActivity()}, routes, nil); err == nil {
		t.Fatal("expected error for route without public activity id")
	}
}

func validBook() Book {
	return Book{
		ID:         "media_0198f8f0-0000-7000-8000-000000000000",
		Title:      "克拉拉与太阳",
		Authors:    []string{"石黑一雄"},
		Status:     "completed",
		FinishedAt: ptrTimePublic(time.Unix(0, 0)),
	}
}

func TestValidate_BookOK(t *testing.T) {
	if err := Validate(validSummary(), []Activity{validActivity()}, RouteFeatureCollection{}, []Book{validBook()}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidate_RejectsUnprefixedBookID(t *testing.T) {
	book := validBook()
	book.ID = "0198f8f0-0000-7000-8000-000000000000"
	if err := Validate(validSummary(), nil, RouteFeatureCollection{}, []Book{book}); err == nil {
		t.Fatal("expected error for unprefixed book id")
	}
}

func TestValidate_RejectsUnknownBookStatus(t *testing.T) {
	book := validBook()
	book.Status = "secretly_reading"
	if err := Validate(validSummary(), nil, RouteFeatureCollection{}, []Book{book}); err == nil {
		t.Fatal("expected error for unknown book status")
	}
}

func TestValidate_RejectsEmptyBookTitle(t *testing.T) {
	book := validBook()
	book.Title = "   "
	if err := Validate(validSummary(), nil, RouteFeatureCollection{}, []Book{book}); err == nil {
		t.Fatal("expected error for empty book title")
	}
}

func TestToBookEncodesPublicIDAndAuthors(t *testing.T) {
	finished := time.Date(2024, 3, 4, 0, 30, 0, 0, time.UTC)
	book, err := ToBook(media.BookExport{
		ID:          uuid.MustParse("0198f8f0-0000-7000-8000-000000000000"),
		Title:       "克拉拉与太阳",
		AuthorsJSON: `["石黑一雄"]`,
		Status:      "completed",
		CompletedOn: &finished,
	})
	if err != nil {
		t.Fatalf("ToBook: %v", err)
	}
	if book.ID != "media_0198f8f0-0000-7000-8000-000000000000" {
		t.Fatalf("id = %q", book.ID)
	}
	if len(book.Authors) != 1 || book.Authors[0] != "石黑一雄" {
		t.Fatalf("authors = %v", book.Authors)
	}
	if book.FinishedAt == nil || !book.FinishedAt.Equal(finished) {
		t.Fatalf("finished_at = %v", book.FinishedAt)
	}
}

func TestToBookFallsBackToEmptyAuthors(t *testing.T) {
	book, err := ToBook(media.BookExport{
		ID:          uuid.New(),
		Title:       "Anonymous",
		AuthorsJSON: "",
		Status:      "in_progress",
	})
	if err != nil {
		t.Fatalf("ToBook: %v", err)
	}
	if book.Authors == nil || len(book.Authors) != 0 {
		t.Fatalf("authors = %v, want empty non-nil", book.Authors)
	}
}
