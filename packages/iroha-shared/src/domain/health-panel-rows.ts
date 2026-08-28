import type { PanelRow } from "../components/metric-panel";
import { formatMetricValue } from "../format/format";
import { coveragePercent, type DailyHealthReportData } from "./report";
import { healthMetricColorVar } from "./health-metric-colors";
import { healthMetricIcon } from "./health-metric-icons";
import { healthMetricLabel } from "./health-metric-labels";
import { healthMetricRange } from "./health-metric-ranges";

// Every theme's Reports.svelte built this exact PanelRow[] from the same
// DailyHealthReportData shape via a near-identical block -- a real fix (the
// round-11 switch from coverage% to each metric's actual value) had to be
// applied six times by hand. One shared builder now means one place to fix.
// `colored` exists only for Grapher, which stays uncolored on purpose,
// continuing its established identity as the plainest of the six languages.
export function healthPanelRows(
  metricAverages: DailyHealthReportData["metric_averages"],
  periodDays: number,
  options: { colored?: boolean } = {},
): PanelRow[] {
  const colored = options.colored ?? true;
  return metricAverages.map((item) => {
    const pct = coveragePercent(item.observed_days, periodDays);
    return {
      label: healthMetricLabel(item.metric),
      icon: healthMetricIcon(item.metric),
      colorVar: colored ? healthMetricColorVar(item.metric) : undefined,
      value: item.value,
      display: formatMetricValue(item.value, item.unit) + " " + item.unit,
      range: healthMetricRange(item.metric),
      breakdown: `${pct}% coverage (${item.observed_days}/${periodDays} days)`,
    };
  });
}
