package drifter

import (
	"fmt"
	"strings"
	"time"

	connector "github.com/azusachino/iroha/apps/iroha-core/connector/v1"
)

// SupportedMajorVersion is the one feed major version this package reads.
// Drifter publishes the contract as schemas/reading-snapshot-v1.schema.json.
const SupportedMajorVersion = "1"

// checkVersion refuses a page whose major version is not supported. It is the
// single check shared by the connector (live pages) and the adapter (stored
// evidence), so neither reads a page the other would reject.
func checkVersion(version string) error {
	major, _, _ := strings.Cut(version, ".")
	if major != SupportedMajorVersion {
		return fmt.Errorf("unsupported drifter feed version %q, want major %s", version, SupportedMajorVersion)
	}
	return nil
}

// ReadingSnapshot is one page of drifter-server's reading feed, served by both
// GET /api/v1/reading/snapshot (a whole shelf) and GET /api/v1/reading/changes
// (books touched since a watermark). The page is stored verbatim as raw
// evidence; these fields exist to read the envelope, the pagination and the
// ordering facts. Nothing the gateway did not state is inferred here.
type ReadingSnapshot struct {
	Version        string                `json:"version"`
	GeneratedAt    time.Time             `json:"generated_at"`
	SourceIdentity string                `json:"source_identity"`
	TargetIdentity string                `json:"target_identity,omitempty"`
	Watermark      string                `json:"watermark"`
	Cursor         string                `json:"cursor"`
	HasMore        bool                  `json:"has_more"`
	Books          []readingSnapshotBook `json:"books"`
	// SnapshotRequired marks a body that refuses the requested watermark even
	// without the matching HTTP status.
	SnapshotRequired bool `json:"snapshot_required,omitempty"`
	// SourceAccount is the pre-replication account field. The gateway now
	// names its accounts through SourceIdentity/TargetIdentity; this stays
	// readable so an older gateway's pages still decode.
	SourceAccount string `json:"source_account,omitempty"`

	// evidence is the exact response this page was decoded from, ready to be
	// stored. Only the connector sets it; the adapter keeps parsing stored
	// bytes so a replay reads the same evidence.
	evidence connector.Snapshot
}

// readingSnapshotBook is Drifter's provider-neutral view of one shelf book.
// Facts the gateway does not know stay absent; the adapter must not infer
// them (see ADR 0011).
type readingSnapshotBook struct {
	SourceProvider     string                `json:"source_provider"`
	SourceBookID       string                `json:"source_book_id"`
	Title              string                `json:"title"`
	Author             string                `json:"author"`
	Translator         string                `json:"translator"`
	Publisher          string                `json:"publisher"`
	ISBN               string                `json:"isbn"`
	CoverURL           string                `json:"cover_url"`
	ReaderURL          string                `json:"reader_url"`
	Status             string                `json:"status"`
	ProgressPercent    int                   `json:"progress_percent"`
	ReadingTimeSeconds int                   `json:"reading_time_seconds"`
	LastReadAt         *time.Time            `json:"last_read_at"`
	FinishedAt         *time.Time            `json:"finished_at"`
	StartedOn          *string               `json:"started_on"`
	IsCustom           bool                  `json:"is_custom"`
	MatchedTarget      *readingMatchedTarget `json:"matched_target"`
}

// readingMatchedTarget is a mapping Drifter already confirmed against another
// catalog. Iroha records it as an external ref rather than re-deciding it.
type readingMatchedTarget struct {
	TargetProvider string `json:"target_provider"`
	WorkID         int64  `json:"work_id"`
	EditionID      int64  `json:"edition_id"`
	OriginalTitle  string `json:"original_title"`
	OriginalAuthor string `json:"original_author"`
}
