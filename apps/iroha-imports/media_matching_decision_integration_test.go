//go:build integration

package imports

import (
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
)

func TestMediaMatchingDecisionSurvivesReplayBridgeRefreshAndUndo(t *testing.T) {
	db := openImportsIntegrationDB(t)
	releaseDate := time.Date(2048, time.January, 1, 0, 0, 0, 0, time.UTC)
	source := seedMediaItem(t, db, "Decision Source", "manga", "series", releaseDate)
	target := seedMediaItem(t, db, "Decision Target", "manga", "series", releaseDate)
	externalID := "integration-matching-decision"
	refID := uuid.New()
	if err := db.Create(&models.MediaExternalRef{
		ID: refID, ScopeType: mediaScopeType, ScopeID: source.itemID,
		Provider: "bangumi", ExternalID: externalID, MatchedBy: mediaMatchProviderID,
		CreatedAt: time.Now().UTC(),
	}).Error; err != nil {
		t.Fatalf("seed provider ref: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Exec("delete from tb_media_matching_decisions where provider = ? and external_id = ?", "bangumi", externalID).Error
		_ = db.Exec("delete from tb_media_external_refs where provider = ? and external_id = ?", "bangumi", externalID).Error
	})

	svc := &Service{db: db}
	if _, err := svc.RecordMediaMatchingDecision(MediaMatchingDecisionInput{
		Provider: "bangumi", ExternalID: externalID, TargetItemID: target.itemID, DecisionKind: MediaDecisionAttach,
	}); err != nil {
		t.Fatalf("record attach decision: %v", err)
	}
	incoming := observations.Media{
		Provider: "bangumi", ExternalID: externalID, Title: "Decision Source",
		MediaType: "manga", ItemRole: "series", ReleaseDate: &releaseDate,
	}
	bridge := StaticMediaRefBridge{"bangumi/" + externalID: {Provider: "anilist", ExternalID: "bridge-target", MatchedBy: mediaMatchBridge}}
	resolution, err := resolveMediaItem(db, incoming, bridge)
	if err != nil {
		t.Fatalf("resolve after attach decision: %v", err)
	}
	if resolution.ItemID != target.itemID || !resolution.Explicit {
		t.Fatalf("resolution after attach = %#v, want explicit target %v", resolution, target.itemID)
	}
	if itemID, err := ensureMediaItem(db, incoming, bridge); err != nil || itemID != target.itemID {
		t.Fatalf("ensure after replay = %v, %v; want target %v", itemID, err, target.itemID)
	}

	if _, err := svc.RecordMediaMatchingDecision(MediaMatchingDecisionInput{
		Provider: "bangumi", ExternalID: externalID, TargetItemID: source.itemID, DecisionKind: MediaDecisionUndo,
	}); err != nil {
		t.Fatalf("record undo decision: %v", err)
	}
	resolution, err = resolveMediaItem(db, incoming, bridge)
	if err != nil {
		t.Fatalf("resolve after undo: %v", err)
	}
	if resolution.ItemID != source.itemID || !resolution.Explicit {
		t.Fatalf("resolution after undo = %#v, want explicit source %v", resolution, source.itemID)
	}
	var ref models.MediaExternalRef
	if err := db.Where("provider = ? and external_id = ?", "bangumi", externalID).First(&ref).Error; err != nil {
		t.Fatalf("load restored provider ref: %v", err)
	}
	if ref.ScopeID != source.itemID || ref.MatchedBy != mediaMatchDecision {
		t.Fatalf("restored ref = scope %v matched_by %q, want source %v and decision", ref.ScopeID, ref.MatchedBy, source.itemID)
	}
	var decisionCount int64
	if err := db.Model(&models.MediaMatchingDecision{}).Where("provider = ? and external_id = ?", "bangumi", externalID).Count(&decisionCount).Error; err != nil {
		t.Fatalf("count matching decisions: %v", err)
	}
	if decisionCount != 2 {
		t.Fatalf("matching decision history count = %d, want 2", decisionCount)
	}
}
