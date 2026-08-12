package reports

import "testing"

func TestDailyHealthDataPreservesSparseMetricCoverage(t *testing.T) {
	data := &DailyHealthData{ObservedDays: 2, MetricAverages: []MetricAverage{{Metric: "steps", Value: 1000, Unit: "count", ObservedDays: 1}}}
	if data.ObservedDays != 2 || len(data.MetricAverages) != 1 || data.MetricAverages[0].ObservedDays != 1 {
		t.Fatalf("daily health data = %+v", data)
	}
	if NewSection[DailyHealthData](DailyHealthSchema, nil).State != SectionEmpty {
		t.Fatal("empty daily health result was not marked empty")
	}
}
