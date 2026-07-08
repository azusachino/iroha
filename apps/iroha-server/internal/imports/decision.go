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
