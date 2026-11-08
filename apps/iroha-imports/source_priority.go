package imports

import (
	"strings"

	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
)

// SourceKind identifies an ingestion source for activities, health metrics, and sleep.
type SourceKind string

const (
	SourceKindAppleHealthExport SourceKind = coreimports.KindAppleHealthExport
	SourceKindHealthAutoExport  SourceKind = coreimports.KindHealthAutoExport
	SourceKindGPX               SourceKind = coreimports.KindGPX
	SourceKindAniList           SourceKind = coreimports.KindAniList
	SourceKindAniListActivity   SourceKind = coreimports.KindAniListActivity
	SourceKindBangumi           SourceKind = coreimports.KindBangumi
	SourceKindManual            SourceKind = "manual"
	SourceKindUnknown           SourceKind = "unknown"
)

// Priority returns the precedence of a source in the reconciliation priority ladder.
// Higher value indicates higher precedence.
//
// Priority ladder:
// 100: Manual / verified owner input
//
//	80: Native Apple Health Export (highest fidelity, granular timeline & samplings)
//	60: Health Auto Export (daily automated sync)
//	40: Third-party GPX / connector imports
//	 0: Unknown / unranked sources
func (k SourceKind) Priority() int {
	switch k {
	case SourceKindManual:
		return 100
	case SourceKindAppleHealthExport:
		return 80
	case SourceKindHealthAutoExport:
		return 60
	case SourceKindGPX:
		return 40
	default:
		return 0
	}
}

// ParseSourceKind normalizes a raw string into a typed SourceKind.
func ParseSourceKind(raw string) SourceKind {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case coreimports.KindAppleHealthExport:
		return SourceKindAppleHealthExport
	case coreimports.KindHealthAutoExport:
		return SourceKindHealthAutoExport
	case coreimports.KindGPX:
		return SourceKindGPX
	case coreimports.KindAniList:
		return SourceKindAniList
	case coreimports.KindAniListActivity:
		return SourceKindAniListActivity
	case coreimports.KindBangumi:
		return SourceKindBangumi
	case "manual", "web", "telegram":
		return SourceKindManual
	default:
		return SourceKind(raw)
	}
}

// SourceWins returns true if the incoming source has higher or equal priority
// compared to the existing source.
func SourceWins(incoming, existing SourceKind) bool {
	if incoming == existing {
		return true
	}
	incPri := incoming.Priority()
	exPri := existing.Priority()
	if incPri != exPri {
		return incPri > exPri
	}
	// Tie-breaker: lexicographical stability
	return incoming < existing
}
