package reports

import (
	"errors"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/daily"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/expenses"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/media"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/sleep"
)

var ErrMissingService = errors.New("monthly report service is not configured")

type Services struct {
	Activities *activities.Service
	Sleep      *sleep.Service
	Daily      *daily.Service
	Media      *media.Service
	Expenses   *expenses.Service
}

func GenerateMonthly(month, timezone string, services Services, generatedAt time.Time) (MonthlyReport, error) {
	period, err := ParseMonth(month, timezone)
	if err != nil {
		return MonthlyReport{}, err
	}
	if generatedAt.IsZero() {
		generatedAt = time.Now()
	}
	if services.Activities == nil || services.Sleep == nil || services.Daily == nil || services.Media == nil || services.Expenses == nil {
		return MonthlyReport{}, ErrMissingService
	}

	movement, err := Movement(services.Activities, period)
	if err != nil {
		return MonthlyReport{}, err
	}
	sleepData, err := Sleep(services.Sleep, period)
	if err != nil {
		return MonthlyReport{}, err
	}
	dailyHealth, err := DailyHealth(services.Daily, period)
	if err != nil {
		return MonthlyReport{}, err
	}
	mediaData, err := Media(services.Media, period)
	if err != nil {
		return MonthlyReport{}, err
	}
	expenseData, err := Expenses(services.Expenses, period)
	if err != nil {
		return MonthlyReport{}, err
	}

	return MonthlyReport{
		Schema: MonthlyReportSchema, Period: period.Wire(), GeneratedAt: generatedAt.UTC(),
		Sections: ReportSections{
			Movement: NewSection(MovementSchema, movement), Sleep: NewSection(SleepSchema, sleepData),
			DailyHealth: NewSection(DailyHealthSchema, dailyHealth), Media: NewSection(MediaSchema, mediaData),
			Expenses: NewSection(ExpensesSchema, expenseData),
		},
	}, nil
}
