package imports

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"gorm.io/gorm"
)

const (
	// mediaBridgeUserAgent matches the provider connectors' UA
	// (apps/iroha-providers/bangumi.DefaultUserAgent) -- not imported from
	// there to avoid a provider-package dependency for one string.
	mediaBridgeUserAgent = "iroha/0.1 (+https://github.com/azusachino/iroha)"
	bangumiExtLinkerURL  = "https://rhilip.github.io/BangumiExtLinker/data/anime_map.json"
	fribbAnimeListsURL   = "https://raw.githubusercontent.com/Fribb/anime-lists/master/anime-list-mini.json"
)

type bangumiMapRecord struct {
	BgmID string `json:"bgm_id"`
	MalID string `json:"mal_id"`
}

type fribbMapRecord struct {
	MalID     json.Number `json:"mal_id"`
	AnilistID json.Number `json:"anilist_id"`
}

// RefreshMediaRefBridge re-fetches both community-maintained crosswalk
// datasets (BangumiExtLinker, Fribb/anime-lists -- see
// docs/media-sync-connectors.md §9) and upserts tb_media_ref_bridge. Mirrors
// scripts/build_media_bridge.py's fetch/build logic; kept as the job-queue
// path (triggered from the /to-go inbox) so a refresh runs on the worker
// that's already deployed instead of a separate scheduled job.
func RefreshMediaRefBridge(ctx context.Context, db *gorm.DB) error {
	client := &http.Client{}

	bangumiRecords, err := fetchJSON[[]bangumiMapRecord](ctx, client, bangumiExtLinkerURL)
	if err != nil {
		return fmt.Errorf("fetch BangumiExtLinker map: %w", err)
	}
	bangumiToMAL := make(map[string]string, len(bangumiRecords))
	for _, record := range bangumiRecords {
		if record.BgmID != "" && record.MalID != "" {
			bangumiToMAL[record.BgmID] = record.MalID
		}
	}

	fribbRecords, err := fetchJSON[[]fribbMapRecord](ctx, client, fribbAnimeListsURL)
	if err != nil {
		return fmt.Errorf("fetch Fribb anime-lists map: %w", err)
	}
	malToAniList := make(map[string]string, len(fribbRecords))
	for _, record := range fribbRecords {
		if record.MalID != "" && record.AnilistID != "" {
			malToAniList[record.MalID.String()] = record.AnilistID.String()
		}
	}

	if err := UpsertMediaRefBridge(db, "bangumi_to_mal", bangumiToMAL); err != nil {
		return fmt.Errorf("upsert bangumi_to_mal: %w", err)
	}
	if err := UpsertMediaRefBridge(db, "mal_to_anilist", malToAniList); err != nil {
		return fmt.Errorf("upsert mal_to_anilist: %w", err)
	}
	return nil
}

func fetchJSON[T any](ctx context.Context, client *http.Client, url string) (T, error) {
	var zero T
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return zero, err
	}
	req.Header.Set("User-Agent", mediaBridgeUserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return zero, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	var value T
	if err := json.NewDecoder(resp.Body).Decode(&value); err != nil {
		return zero, err
	}
	return value, nil
}
