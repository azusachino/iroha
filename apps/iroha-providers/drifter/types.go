package drifter

import "time"

// readingSnapshotResponse is one page of drifter-server's
// GET /api/v1/reading/snapshot response. The page is stored verbatim as raw
// evidence; the fields here exist only to read the pagination envelope.
type readingSnapshotResponse struct {
	Version       string                `json:"version"`
	GeneratedAt   time.Time             `json:"generated_at"`
	SourceAccount string                `json:"source_account"`
	Cursor        string                `json:"cursor"`
	HasMore       bool                  `json:"has_more"`
	Books         []readingSnapshotBook `json:"books"`
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
