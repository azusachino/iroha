package reports

import "time"

const (
	SectionAvailable = "available"
	SectionEmpty     = "empty"
)

type ReportSection[T any] struct {
	Schema string `json:"schema"`
	State  string `json:"state"`
	Data   *T     `json:"data"`
}

type MonthlyReport struct {
	Schema      string         `json:"schema"`
	Period      ReportMonth    `json:"period"`
	GeneratedAt time.Time      `json:"generated_at"`
	Sections    ReportSections `json:"sections"`
}

type MonthlyReportSeries struct {
	Schema          string                     `json:"schema"`
	EndMonth        string                     `json:"end_month"`
	RequestedMonths int                        `json:"requested_months"`
	FromMonth       string                     `json:"from_month"`
	ToMonth         string                     `json:"to_month"`
	GeneratedAt     time.Time                  `json:"generated_at"`
	CurrentReport   *MonthlyReport             `json:"current_report"`
	Reports         []MonthlyReportSeriesPoint `json:"reports"`
	EmptyMonths     []string                   `json:"empty_months"`
}

type MonthlyReportSeriesPoint struct {
	Month        string                         `json:"month"`
	Completeness string                         `json:"completeness"`
	Movement     *MonthlyReportMovementTrend    `json:"movement"`
	Sleep        *MonthlyReportSleepTrend       `json:"sleep"`
	DailyHealth  *MonthlyReportDailyHealthTrend `json:"daily_health"`
	Media        *MonthlyReportMediaTrend       `json:"media"`
	Expenses     *MonthlyReportExpensesTrend    `json:"expenses"`
}

type MonthlyReportMovementTrend struct {
	DistanceM float64 `json:"distance_m"`
}

type MonthlyReportSleepTrend struct {
	AverageAsleepS float64 `json:"average_asleep_s"`
}

type MonthlyReportDailyHealthTrend struct {
	ObservedDays   int             `json:"observed_days"`
	MetricAverages []MetricAverage `json:"metric_averages"`
}

type MonthlyReportMediaTrend struct {
	EventCount     int `json:"event_count"`
	CompletedCount int `json:"completed_count"`
}

type MonthlyReportExpensesTrend struct {
	TotalsByCurrency []ExpenseCurrencyTotal `json:"totals_by_currency"`
}

type ReportMonth struct {
	Kind     string `json:"kind"`
	Month    string `json:"month"`
	From     string `json:"from"`
	To       string `json:"to"`
	Timezone string `json:"timezone"`
}

type ReportSections struct {
	Movement    ReportSection[MovementData]    `json:"movement"`
	Sleep       ReportSection[SleepData]       `json:"sleep"`
	DailyHealth ReportSection[DailyHealthData] `json:"daily_health"`
	Media       ReportSection[MediaData]       `json:"media"`
	Expenses    ReportSection[ExpensesData]    `json:"expenses"`
}

type MovementData struct {
	ActivityCount         int                  `json:"activity_count"`
	DistanceM             float64              `json:"distance_m"`
	DistanceActivityCount int                  `json:"distance_activity_count"`
	DurationS             int                  `json:"duration_s"`
	BySport               []MovementSportTotal `json:"by_sport"`
}

type SleepData struct {
	SessionCount      int               `json:"session_count"`
	MainSleepCount    int               `json:"main_sleep_count"`
	NapCount          int               `json:"nap_count"`
	AverageAsleepS    float64           `json:"average_asleep_s"`
	AverageTimeInBedS float64           `json:"average_time_in_bed_s"`
	AverageEfficiency float64           `json:"average_efficiency"`
	StageSeconds      SleepStageSeconds `json:"stage_seconds"`
}

type DailyHealthData struct {
	ObservedDays   int             `json:"observed_days"`
	MetricAverages []MetricAverage `json:"metric_averages"`
}

type MediaData struct {
	EventCount     int              `json:"event_count"`
	CompletedCount int              `json:"completed_count"`
	RatedCount     int              `json:"rated_count"`
	AverageRating  *float64         `json:"average_rating"`
	ByKind         []MediaKindTotal `json:"by_kind"`
	CompletedItems []MediaCompleted `json:"completed_items"`
}

type ExpensesData struct {
	ExpenseCount     int                    `json:"expense_count"`
	TotalsByCurrency []ExpenseCurrencyTotal `json:"totals_by_currency"`
	ByCategory       []ExpenseCategoryTotal `json:"by_category"`
}

type MetricAverage struct {
	Metric       string  `json:"metric"`
	Value        float64 `json:"value"`
	Unit         string  `json:"unit"`
	ObservedDays int     `json:"observed_days"`
}

type MovementSportTotal struct {
	Sport                 string  `json:"sport"`
	ActivityCount         int     `json:"activity_count"`
	DistanceM             float64 `json:"distance_m"`
	DistanceActivityCount int     `json:"distance_activity_count"`
	DurationS             int     `json:"duration_s"`
}

type SleepStageSeconds struct {
	Core        int `json:"core"`
	Deep        int `json:"deep"`
	Rem         int `json:"rem"`
	Awake       int `json:"awake"`
	Unspecified int `json:"unspecified"`
}

type MediaKindTotal struct {
	Kind           string `json:"kind"`
	EventCount     int    `json:"event_count"`
	CompletedCount int    `json:"completed_count"`
}

type MediaCompleted struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	MediaType   string    `json:"media_type"`
	CompletedAt time.Time `json:"completed_at"`
}

type ExpenseCurrencyTotal struct {
	Currency         string `json:"currency"`
	CurrencyExponent int    `json:"currency_exponent"`
	AmountMinor      int64  `json:"amount_minor"`
	ExpenseCount     int    `json:"expense_count"`
}

type ExpenseCategoryTotal struct {
	Category         string `json:"category"`
	Currency         string `json:"currency"`
	CurrencyExponent int    `json:"currency_exponent"`
	AmountMinor      int64  `json:"amount_minor"`
	ExpenseCount     int    `json:"expense_count"`
}

func NewSection[T any](schema string, data *T) ReportSection[T] {
	state := SectionAvailable
	if data == nil {
		state = SectionEmpty
	}
	return ReportSection[T]{Schema: schema, State: state, Data: data}
}
