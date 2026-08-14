package reports

import (
	"testing"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/sleep"
)

func TestSleepDataKeepsNapsSeparateAndEmptyExplicit(t *testing.T) {
	result := sleep.PeriodReport{SessionCount: 2, MainSleepCount: 1, NapCount: 1, AverageAsleepS: 24000, AverageTimeInBedS: 27000, AverageEfficiency: 0.88}
	result.StageSeconds.Core = 12000
	data := &SleepData{SessionCount: result.SessionCount, MainSleepCount: result.MainSleepCount, NapCount: result.NapCount, AverageAsleepS: result.AverageAsleepS, AverageTimeInBedS: result.AverageTimeInBedS, AverageEfficiency: result.AverageEfficiency, StageSeconds: SleepStageSeconds{Core: result.StageSeconds.Core}}
	if data.SessionCount != 2 || data.MainSleepCount != 1 || data.NapCount != 1 || data.StageSeconds.Core != 12000 {
		t.Fatalf("sleep data = %+v", data)
	}
	if NewSection[SleepData](SleepSchema, nil).State != SectionEmpty {
		t.Fatal("empty sleep result was not marked empty")
	}
}
