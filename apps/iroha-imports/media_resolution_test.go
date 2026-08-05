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
