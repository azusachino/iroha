package imports

import "testing"

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
