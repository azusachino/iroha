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
	if !servicesConfigured(services) {
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

func GenerateMonthlySeries(endMonth, timezone string, months int, services Services, generatedAt time.Time) (MonthlyReportSeries, error) {
	if months <= 0 || months > MaxSeriesMonths {
		return MonthlyReportSeries{}, ErrInvalidSeriesMonths
	}
	endPeriod, err := ParseMonth(endMonth, timezone)
	if err != nil {
		return MonthlyReportSeries{}, err
	}
	if !servicesConfigured(services) {
		return MonthlyReportSeries{}, ErrMissingService
	}
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	location, err := LoadTimezone(endPeriod.Timezone)
	if err != nil {
		return MonthlyReportSeries{}, err
	}
	fromMonth := endPeriod.FromDate.AddDate(0, 1-months, 0)
	series := MonthlyReportSeries{
		Schema:          MonthlyReportSeriesSchema,
		EndMonth:        endPeriod.Month,
		RequestedMonths: months,
		FromMonth:       fromMonth.Format("2006-01"),
		ToMonth:         endPeriod.Month,
		GeneratedAt:     generatedAt.UTC(),
		CurrentReport:   nil,
		Reports:         make([]MonthlyReportSeriesPoint, 0, months),
		EmptyMonths:     make([]string, 0),
	}
	for index := 0; index < months; index++ {
		periodMonth := fromMonth.AddDate(0, index, 0).Format("2006-01")
		report, err := GenerateMonthly(periodMonth, endPeriod.Timezone, services, generatedAt)
		if err != nil {
			return MonthlyReportSeries{}, err
		}
		if !reportHasData(report) {
			if periodMonth == endPeriod.Month {
				series.CurrentReport = &report
			}
			series.EmptyMonths = append(series.EmptyMonths, periodMonth)
			continue
		}
		if periodMonth == endPeriod.Month {
			series.CurrentReport = &report
		}
		series.Reports = append(series.Reports, MonthlyReportSeriesPoint{
			Month:        periodMonth,
			Completeness: monthCompleteness(report.Period, generatedAt, location),
			Movement:     monthlyMovementTrend(report),
			Sleep:        monthlySleepTrend(report),
			DailyHealth:  monthlyDailyHealthTrend(report),
			Media:        monthlyMediaTrend(report),
			Expenses:     monthlyExpensesTrend(report),
		})
	}
	return series, nil
}

func monthlyMovementTrend(report MonthlyReport) *MonthlyReportMovementTrend {
	if report.Sections.Movement.Data == nil {
		return nil
	}
	return &MonthlyReportMovementTrend{DistanceM: report.Sections.Movement.Data.DistanceM}
}

func monthlySleepTrend(report MonthlyReport) *MonthlyReportSleepTrend {
	if report.Sections.Sleep.Data == nil {
		return nil
	}
	return &MonthlyReportSleepTrend{AverageAsleepS: report.Sections.Sleep.Data.AverageAsleepS}
}

func monthlyDailyHealthTrend(report MonthlyReport) *MonthlyReportDailyHealthTrend {
	if report.Sections.DailyHealth.Data == nil {
		return nil
	}
	return &MonthlyReportDailyHealthTrend{ObservedDays: report.Sections.DailyHealth.Data.ObservedDays}
}

func monthlyMediaTrend(report MonthlyReport) *MonthlyReportMediaTrend {
	if report.Sections.Media.Data == nil {
		return nil
	}
	return &MonthlyReportMediaTrend{
		EventCount:     report.Sections.Media.Data.EventCount,
		CompletedCount: report.Sections.Media.Data.CompletedCount,
	}
}

func monthlyExpensesTrend(report MonthlyReport) *MonthlyReportExpensesTrend {
	if report.Sections.Expenses.Data == nil {
		return nil
	}
	return &MonthlyReportExpensesTrend{TotalsByCurrency: report.Sections.Expenses.Data.TotalsByCurrency}
}

func servicesConfigured(services Services) bool {
	return services.Activities != nil && services.Sleep != nil && services.Daily != nil && services.Media != nil && services.Expenses != nil
}

func reportHasData(report MonthlyReport) bool {
	return report.Sections.Movement.Data != nil || report.Sections.Sleep.Data != nil || report.Sections.DailyHealth.Data != nil || report.Sections.Media.Data != nil || report.Sections.Expenses.Data != nil
}

func monthCompleteness(period ReportMonth, generatedAt time.Time, location *time.Location) string {
	to, err := time.ParseInLocation("2006-01-02", period.To, location)
	if err != nil || generatedAt.In(location).Before(to) {
		return CompletenessPartial
	}
	return CompletenessComplete
}
