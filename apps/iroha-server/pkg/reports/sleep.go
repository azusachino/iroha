package reports

import (
	"github.com/azusachino/iroha/apps/iroha-server/pkg/sleep"
)

func Sleep(service *sleep.Service, period Period) (*SleepData, error) {
	result, err := service.PeriodReport(sleep.PeriodFilters{From: period.FromDate, To: period.ToDateExclusive})
	if err != nil {
		return nil, err
	}
	if result.SessionCount == 0 {
		return nil, nil
	}
	return &SleepData{
		SessionCount: result.SessionCount, MainSleepCount: result.MainSleepCount, NapCount: result.NapCount,
		AverageAsleepS: result.AverageAsleepS, AverageTimeInBedS: result.AverageTimeInBedS,
		AverageEfficiency: result.AverageEfficiency,
		StageSeconds:      SleepStageSeconds{Core: result.StageSeconds.Core, Deep: result.StageSeconds.Deep, Rem: result.StageSeconds.Rem, Awake: result.StageSeconds.Awake, Unspecified: result.StageSeconds.Unspecified},
	}, nil
}
