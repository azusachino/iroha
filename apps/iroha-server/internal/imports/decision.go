package imports

// sourceItemDecision captures the outcome of comparing a parsed Apple
// source item's content hash against what we already have on record for
// that source key.
type sourceItemDecision int

const (
	// sourceItemNew means no tb_apple_source_items row exists yet for this
	// source key - this is the first time we've seen this workout.
	sourceItemNew sourceItemDecision = iota
	// sourceItemChanged means a row exists but its content hash differs -
	// the workout's change-relevant fields were edited since we last saw it.
	sourceItemChanged
	// sourceItemUnchanged means a row exists with the same content hash -
	// this is a re-export of the same, unmodified workout.
	sourceItemUnchanged
)

// decideSourceItem is a pure function so it can be unit-tested without a
// database: given the content hash already on record for a source key (nil
// if no row exists yet) and the content hash freshly parsed from the
// current export, it decides whether the source item is new, changed, or
// unchanged.
func decideSourceItem(existingContentHash *string, newContentHash string) sourceItemDecision {
	if existingContentHash == nil {
		return sourceItemNew
	}
	if *existingContentHash == newContentHash {
		return sourceItemUnchanged
	}
	return sourceItemChanged
}

// importDisposition captures how process() should handle a queued import
// job relative to prior completed imports of the same raw file (matched by
// sha256, since raw files themselves are deduped by sha256 at upload time).
type importDisposition int

const (
	// dispositionFresh means no completed import exists for this raw
	// file's sha256 - persist as a brand new import.
	dispositionFresh importDisposition = iota
	// dispositionSkip means a completed import already exists for this
	// sha256 at the SAME parser_version - this is a redundant re-run
	// against already-imported content; short-circuit without re-parsing
	// or re-persisting anything.
	dispositionSkip
	// dispositionReprocess means a completed import exists for this
	// sha256 but at a DIFFERENT parser_version - a parser upgrade (or
	// downgrade). Raw files are canonical snapshots, so this must purge
	// everything derived from the raw file and persist fresh rather than
	// appending alongside the stale rows.
	dispositionReprocess
)

// decideImportDisposition is a pure function so the skip/reprocess/fresh
// three-way branch can be unit-tested without a database. priorSameVersion
// is whether a completed import exists for this raw file's sha256 at the
// same parser_version as the current job; priorAnyVersion is whether a
// completed import exists for this sha256 at any parser_version (including
// the same one). priorSameVersion implies priorAnyVersion.
func decideImportDisposition(priorSameVersion bool, priorAnyVersion bool) importDisposition {
	if priorSameVersion {
		return dispositionSkip
	}
	if priorAnyVersion {
		return dispositionReprocess
	}
	return dispositionFresh
}
