package i18n

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// activityTitleMigrationRow matches one ('provider label', 'canonical title',
// 'sport slug') row of migration 00027's title map.
var activityTitleMigrationRow = regexp.MustCompile(`\('([^']+)', '([^']+)', '([^']+)'\)`)

func readActivityTitleMigration(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "apps", "iroha-server", "db", "migrations", "00027_canonicalize_activity_titles.up.sql")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration 00027: %v", err)
	}
	return string(body)
}

// readActivityTitleMigrationMap returns only the title-map INSERT statement, so
// a check-constraint value list elsewhere in the file cannot be mistaken for a
// mapping row.
func readActivityTitleMigrationMap(t *testing.T) string {
	t.Helper()
	body := readActivityTitleMigration(t)
	start := strings.Index(body, "insert into activity_title_map")
	if start < 0 {
		t.Fatal("migration 00027 has no activity_title_map insert")
	}
	statement := body[start:]
	if end := strings.Index(statement, ";"); end >= 0 {
		statement = statement[:end]
	}
	return statement
}

// TestActivityTitleMigrationAgreesWithTaxonomy guards the data rewrite in
// migration 00027 against drifting away from this package. The migration is an
// immutable snapshot, so this asserts agreement where it overlaps rather than
// demanding every future synonym appear there.
func TestActivityTitleMigrationAgreesWithTaxonomy(t *testing.T) {
	rows := activityTitleMigrationRow.FindAllStringSubmatch(readActivityTitleMigrationMap(t), -1)
	if len(rows) < 50 {
		t.Fatalf("found %d title-map rows, want the taxonomy snapshot", len(rows))
	}

	for _, row := range rows {
		oldTitle, wantTitle, wantSport := row[1], row[2], row[3]
		got := ResolveActivity(oldTitle, ActivityContext{})
		if !got.IsSystemLabel {
			t.Errorf("migration rewrites %q, which the taxonomy does not treat as a system label", oldTitle)
			continue
		}
		if got.CanonicalTitle != wantTitle || got.SportType != wantSport {
			t.Errorf("migration maps %q to %q/%q, taxonomy says %q/%q",
				oldTitle, wantTitle, wantSport, got.CanonicalTitle, got.SportType)
		}
	}
}

// TestActivityTitleMigrationCoversRequiredLabels pins the labels ADR-0012 and
// the Slice 3 brief require the remediation to rewrite.
func TestActivityTitleMigrationCoversRequiredLabels(t *testing.T) {
	required := map[string]string{
		"ラン":          ActivityTitleRun,
		"屋外ラン":        ActivityTitleOutdoorRun,
		"屋外 ラン":       ActivityTitleOutdoorRun,
		"屋外ランニング":     ActivityTitleOutdoorRun,
		"户外跑步":        ActivityTitleOutdoorRun,
		"屋内ラン":        ActivityTitleIndoorRun,
		"屋内 ラン":       ActivityTitleIndoorRun,
		"トレッドミル":      ActivityTitleIndoorRun,
		"室内跑步":        ActivityTitleIndoorRun,
		"跑步机":         ActivityTitleIndoorRun,
		"ウォーキング":      ActivityTitleWalk,
		"步行":          ActivityTitleWalk,
		"屋外ウォーキング":    ActivityTitleOutdoorWalk,
		"屋外 ウォーキング":   ActivityTitleOutdoorWalk,
		"户外步行":        ActivityTitleOutdoorWalk,
		"屋内ウォーキング":    ActivityTitleIndoorWalk,
		"屋内 ウォーキング":   ActivityTitleIndoorWalk,
		"室内步行":        ActivityTitleIndoorWalk,
		"サイクリング":      ActivityTitleOutdoorCycling,
		"屋外サイクリング":    ActivityTitleOutdoorCycling,
		"屋外 サイクリング":   ActivityTitleOutdoorCycling,
		"户外骑行":        ActivityTitleOutdoorCycling,
		"屋内サイクリング":    ActivityTitleIndoorCycling,
		"屋内 サイクリング":   ActivityTitleIndoorCycling,
		"室内骑行":        ActivityTitleIndoorCycling,
		"プールスイミング":    ActivityTitlePoolSwim,
		"スイム":         ActivityTitlePoolSwim,
		"泳池游泳":        ActivityTitlePoolSwim,
		"ハイキング":       ActivityTitleHiking,
		"徒步":          ActivityTitleHiking,
		"機能的筋力トレーニング": ActivityTitleStrengthTraining,
		"筋トレ":         ActivityTitleStrengthTraining,
		"功能性力量训练":     ActivityTitleStrengthTraining,
		"高強度インターバルトレーニング": ActivityTitleHIIT,
		"高强度间歇训练":         ActivityTitleHIIT,
		"ヨガ":              ActivityTitleYoga,
		"瑜伽":              ActivityTitleYoga,
		"フィットネスゲーム":       ActivityTitleFitnessGaming,
		"健身游戏":            ActivityTitleFitnessGaming,
	}

	rewritten := make(map[string]string)
	for _, row := range activityTitleMigrationRow.FindAllStringSubmatch(readActivityTitleMigrationMap(t), -1) {
		rewritten[row[1]] = row[2]
	}
	for label, wantTitle := range required {
		got, ok := rewritten[label]
		if !ok {
			t.Errorf("migration 00027 does not rewrite %q", label)
			continue
		}
		if got != wantTitle {
			t.Errorf("migration 00027 rewrites %q to %q, want %q", label, got, wantTitle)
		}
	}
}
