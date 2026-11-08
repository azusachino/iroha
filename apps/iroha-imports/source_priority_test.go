package imports

import (
	"testing"
)

func TestSourcePriorityLadder(t *testing.T) {
	tests := []struct {
		name     string
		incoming SourceKind
		existing SourceKind
		wantWins bool
	}{
		{
			name:     "apple health export beats health auto export",
			incoming: SourceKindAppleHealthExport,
			existing: SourceKindHealthAutoExport,
			wantWins: true,
		},
		{
			name:     "health auto export does not overwrite apple health export",
			incoming: SourceKindHealthAutoExport,
			existing: SourceKindAppleHealthExport,
			wantWins: false,
		},
		{
			name:     "manual beats apple health export",
			incoming: SourceKindManual,
			existing: SourceKindAppleHealthExport,
			wantWins: true,
		},
		{
			name:     "health auto export beats gpx",
			incoming: SourceKindHealthAutoExport,
			existing: SourceKindGPX,
			wantWins: true,
		},
		{
			name:     "same source always wins (refresh)",
			incoming: SourceKindAppleHealthExport,
			existing: SourceKindAppleHealthExport,
			wantWins: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SourceWins(tt.incoming, tt.existing)
			if got != tt.wantWins {
				t.Errorf("SourceWins(%v, %v) = %v, want %v", tt.incoming, tt.existing, got, tt.wantWins)
			}
		})
	}
}
