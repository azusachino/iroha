package imports

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTwoHopMediaRefBridge(t *testing.T) {
	dir := t.TempDir()
	bgmPath := filepath.Join(dir, "bangumi_to_mal.json")
	malPath := filepath.Join(dir, "mal_to_anilist.json")
	if err := os.WriteFile(bgmPath, []byte(`{"8":"2904"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(malPath, []byte(`{"2904":"99423"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	bridge, err := LoadTwoHopMediaRefBridge(bgmPath, malPath)
	if err != nil {
		t.Fatalf("LoadTwoHopMediaRefBridge: %v", err)
	}
	ref, ok := bridge.Lookup("bangumi", "8")
	if !ok || ref.Provider != "anilist" || ref.ExternalID != "99423" {
		t.Fatalf("loaded bridge lookup = %#v, %v; want anilist/99423", ref, ok)
	}
}

func TestLoadTwoHopMediaRefBridge_EmptyPathsSkipped(t *testing.T) {
	bridge, err := LoadTwoHopMediaRefBridge("", "")
	if err != nil {
		t.Fatalf("LoadTwoHopMediaRefBridge with empty paths: %v", err)
	}
	if _, ok := bridge.Lookup("bangumi", "8"); ok {
		t.Fatal("expected no lookup result from an empty bridge")
	}
}

// TestNormalizeMediaTitle_CanonicalKeyCollisionSafety guards the other side
// of the dedup fix: normalizeMediaTitle strips whitespace and bracketed
// annotations entirely to bridge real cross-provider formatting
// differences, and that's only safe if it doesn't also start conflating
// genuinely different titles into the same canonical key. The "same" cases
// are the exact real prod pairs the fix targets; the "different" cases are
// real, distinct, well-known titles chosen to adversarially probe the same
// mechanism (shared prefixes, sequel/season naming, punctuation-adjacent
// text) without a bracket/date/media_type/item_role guard to save it here --
// those guards live in titleYearCandidates, not in this function, so this
// test intentionally checks normalizeMediaTitle alone.
func TestNormalizeMediaTitle_CanonicalKeyCollisionSafety(t *testing.T) {
	sameCases := []struct{ name, a, b string }{
		{"trailing bracketed gloss", "拝啓、天国の姉さん、勇者になった姪がエロすぎてーー 叔父さん、保護者とかそろそろ無理です＋（ぷらす）", "拝啓、天国の姉さん、勇者になった姪がエロすぎてーー 叔父さん、保護者とかそろそろ無理です＋"},
		{"differing bracket style", "追放された転生王子、『自動製作《オートクラフト》』", "追放された転生王子、『自動製作（オートクラフト）』"},
		{"fullwidth vs ASCII tilde plus spacing", "史上最強の魔法剣士、Fランク冒険者に転生する  ~剣聖と魔帝、2つの前世を持った男の英雄譚~", "史上最強の魔法剣士、Fランク冒険者に転生する ～剣聖と魔帝、2つの前世を持った男の英雄譚～"},
		{"inserted separator space", "その悪役貴族、ママヒロインが好きすぎる～真摯な努力で最強となり不遇な推しキャラ助けまくる～", "その悪役貴族、ママヒロインが好きすぎる ～真摯な努力で最強となり不遇な推しキャラ助けまくる～"},
	}
	for _, tc := range sameCases {
		t.Run("same/"+tc.name, func(t *testing.T) {
			normA, normB := normalizeMediaTitle(tc.a), normalizeMediaTitle(tc.b)
			if normA != normB {
				t.Fatalf("normalizeMediaTitle(%q) = %q, normalizeMediaTitle(%q) = %q -- want equal", tc.a, normA, tc.b, normB)
			}
		})
	}

	differentCases := []struct{ name, a, b string }{
		{"different season/cour of the same franchise", "SPY×FAMILY 第2クール", "SPY×FAMILY"},
		{"sequel with a bracketed disambiguator dropped", "薬屋のひとりごと（第二期）", "薬屋のひとりごと"},
		{"shared prefix, different subtitle content", "史上最強の魔法剣士、Fランク冒険者に転生する ～剣聖と魔帝の物語～", "史上最強の魔法剣士、Fランク冒険者に転生する ～勇者と魔王の物語～"},
		{"shared English words, different franchise", "The Villainous Noble is Way Too Fond of Cats", "The Villainous Noble is Way Too Fond of MILF Heroines"},
		{"remake vs original sharing a bracketed year", "Fullmetal Alchemist (2003)", "Fullmetal Alchemist (2009)"},
	}
	for _, tc := range differentCases {
		t.Run("different/"+tc.name, func(t *testing.T) {
			normA, normB := normalizeMediaTitle(tc.a), normalizeMediaTitle(tc.b)
			if normA == normB {
				t.Fatalf("normalizeMediaTitle(%q) and normalizeMediaTitle(%q) both = %q -- want different canonical keys for genuinely different titles", tc.a, tc.b, normA)
			}
		})
	}
}

func TestTwoHopMediaRefBridge(t *testing.T) {
	bridge := TwoHopMediaRefBridge{
		BangumiToMAL: map[string]string{"bg-1": "mal-1", "bg-2": "mal-2"},
		MALToAniList: map[string]string{"mal-1": "ani-1"},
	}

	ref, ok := bridge.Lookup("bangumi", "bg-1")
	if !ok || ref.Provider != "anilist" || ref.ExternalID != "ani-1" {
		t.Fatalf("full bridge = %#v, %v; want anilist/ani-1", ref, ok)
	}
	ref, ok = bridge.Lookup("bangumi", "bg-2")
	if !ok || ref.Provider != "mal" || ref.ExternalID != "mal-2" {
		t.Fatalf("partial bridge = %#v, %v; want mal/mal-2", ref, ok)
	}
	if _, ok := bridge.Lookup("bangumi", "missing"); ok {
		t.Fatal("missing Bangumi mapping unexpectedly resolved")
	}
}
