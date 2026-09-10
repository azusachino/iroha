package imports

// importDisposition captures how process() should handle a queued import
// job relative to prior completed imports of the same raw file.
type importDisposition int

const (
	// dispositionFresh means no completed import exists for this raw file.
	dispositionFresh importDisposition = iota
	// dispositionSkip means the same raw file was already completed at this parser version.
	dispositionSkip
	// dispositionReprocess means a completed import exists at another parser version.
	dispositionReprocess
)

// decideImportDisposition is a pure function so the skip/reprocess/fresh
// three-way branch can be tested without a database. priorSameVersion implies
// priorAnyVersion, but the exact match wins even for inconsistent inputs.
func decideImportDisposition(priorSameVersion bool, priorAnyVersion bool) importDisposition {
	if priorSameVersion {
		return dispositionSkip
	}
	if priorAnyVersion {
		return dispositionReprocess
	}
	return dispositionFresh
}
