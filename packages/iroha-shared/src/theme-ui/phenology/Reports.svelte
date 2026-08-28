<script lang="ts">
  import BarChart from "../components/BarChart.svelte";
  import CoverageBars from "../../components/CoverageBars.svelte";
  import ReportCoverage from "../components/ReportCoverage.svelte";
  import ReportReceipt from "../components/ReportReceipt.svelte";
  import type { ReportEvidenceRow } from "../../domain/report";
  import type { DesignLanguage } from "../../theme/themes";
  import ReportFactGrid from "../components/ReportFactGrid.svelte";
  import ReportMetricCard from "../components/ReportMetricCard.svelte";
  import MetricPanel from "../../components/MetricPanel.svelte";
  import type { PanelRow } from "../../components/metric-panel";
  import { formatMetricValue } from "../../format/format";
  import { healthMetricLabel } from "../../domain/health-metric-labels";
  import { healthMetricIcon } from "../../domain/health-metric-icons";
  import { healthMetricColorVar } from "../../domain/health-metric-colors";
  import {
    coveragePercent,
    reportPeriodDays,
    reportSectionData,
    type ReportThemeProps,
  } from "../../domain/report";

  let {
    month,
    report,
    primaryCurrency,
    primaryExponent,
    formatMoney,
    formatDuration,
    theme,
  }: ReportThemeProps & { theme: DesignLanguage } = $props();

  const movement = $derived(
    reportSectionData<
      ReportThemeProps["report"]["sections"]["movement"]["data"]
    >(report, "movement"),
  );
  const sleep = $derived(
    reportSectionData<ReportThemeProps["report"]["sections"]["sleep"]["data"]>(
      report,
      "sleep",
    ),
  );
  const health = $derived(
    reportSectionData<
      ReportThemeProps["report"]["sections"]["daily_health"]["data"]
    >(report, "daily_health"),
  );
  const media = $derived(
    reportSectionData<ReportThemeProps["report"]["sections"]["media"]["data"]>(
      report,
      "media",
    ),
  );
  const expenses = $derived(
    reportSectionData<
      ReportThemeProps["report"]["sections"]["expenses"]["data"]
    >(report, "expenses"),
  );
  const categoryTotals = $derived(
    [...(expenses?.by_category ?? [])]
      .filter((item) => item.currency === primaryCurrency)
      .sort((left, right) => right.amount_minor - left.amount_minor),
  );
  const sleepStages = $derived<[string, number][]>(
    sleep
      ? [
          ["Core", sleep.stage_seconds.core],
          ["Deep", sleep.stage_seconds.deep],
          ["REM", sleep.stage_seconds.rem],
          ["Awake", sleep.stage_seconds.awake],
          ["Unspecified", sleep.stage_seconds.unspecified],
        ]
      : [],
  );
  const sleepRows = $derived<PanelRow[]>(
    sleepStages.map(([stage, seconds]) => ({
      label: stage,
      value: seconds / 3600,
      display: number(seconds / 3600) + "h",
    })),
  );
  const evidence = $derived<ReportEvidenceRow[]>([
    ...(sleep
      ? [
          {
            label: "Recovery",
            value: formatDuration(sleep.average_asleep_s),
            detail:
              Math.round(sleep.average_efficiency * 100) +
              "% efficiency · " +
              sleep.main_sleep_count +
              " main sleeps · " +
              sleep.nap_count +
              " naps",
          },
        ]
      : []),
    ...(movement?.by_sport ?? []).map((item) => ({
      label: "Movement · " + item.sport,
      value: number(item.distance_m / 1000) + " km",
      detail: item.activity_count + " sessions",
    })),
    ...(health?.metric_averages ?? []).map((item) => ({
      label: "Health · " + healthMetricLabel(item.metric),
      value: formatMetricValue(item.value, item.unit) + " " + item.unit,
      detail: item.observed_days + " observed days",
    })),
    ...(media
      ? [
          {
            label: "Media rhythm",
            value: media.event_count + " events",
            detail: media.completed_count + " completed",
          },
        ]
      : []),
    ...categoryTotals.map((item) => ({
      label: "Spend · " + item.category,
      value: formatMoney(item.amount_minor, primaryCurrency, primaryExponent),
      detail: item.expense_count + " records",
    })),
  ]);
  const periodDays = $derived(reportPeriodDays(report));
  const healthRows = $derived<PanelRow[]>(
    (health?.metric_averages ?? []).map((item) => {
      const pct = coveragePercent(item.observed_days, periodDays);
      return {
        label: healthMetricLabel(item.metric),
        icon: healthMetricIcon(item.metric),
        colorVar: healthMetricColorVar(item.metric),
        value: pct,
        display: pct + "%",
        breakdown: formatMetricValue(item.value, item.unit) + " " + item.unit,
      };
    }),
  );

  function number(value: number): string {
    return new Intl.NumberFormat(undefined, {
      maximumFractionDigits: 1,
    }).format(value);
  }
</script>

<section class="phenology-reports" aria-labelledby="phenology-reports-title">
  <header class="phenology-heading">
    <div>
      <p class="kicker">Monthly cycle · {month}</p>
      <h2 id="phenology-reports-title">Observe the season of the data.</h2>
      <p>Recovery is the center; the other domains orbit around the month.</p>
    </div>
    <span>{report.period.from} → {report.period.to}</span>
  </header>

  <ReportCoverage {report} />

  <div class="cycle-stage">
    <ReportMetricCard
      label="Recovery center"
      title="Sleep-stage composition"
      summary={sleep ? formatDuration(sleep.average_asleep_s) : "No records"}
      tone="feature"
    >
      {#if sleep}
        <MetricPanel
          metricId="sleep.stage_seconds"
          label="Sleep-stage composition"
          unit="hours"
          method={report.sections.sleep.schema}
          rowHeader="Stage"
          rows={sleepRows}
          period={month}
        >
          <BarChart
            categories={sleepStages.map(([stage]) => stage)}
            primary={{
              name: "Hours",
              values: sleepStages.map(([, seconds]) => seconds / 3600),
              color: "var(--accent-2)",
              formatter: (value) => number(value) + "h",
            }}
            categorical
            height={270}
          />
        </MetricPanel>
        <ReportFactGrid
          facts={[
            {
              label: "Efficiency",
              value: Math.round(sleep.average_efficiency * 100) + "%",
            },
            { label: "Main sleep", value: String(sleep.main_sleep_count) },
            { label: "Naps", value: String(sleep.nap_count) },
          ]}
        />
      {:else}<p class="empty">No canonical sleep records.</p>{/if}
    </ReportMetricCard>
  </div>

  <div class="orbit-grid">
    <ReportMetricCard
      label="Movement orbit"
      title="Distance by sport"
      summary={movement ? movement.activity_count + " sessions" : "No records"}
    >
      {#if movement?.by_sport.length}
        <MetricPanel
          metricId="movement.distance_m"
          label="Distance by sport"
          unit="km"
          method={report.sections.movement.schema}
          rowHeader="Sport"
          rows={movement.by_sport.map((item) => ({
            label: item.sport,
            value: item.distance_m / 1000,
            display: number(item.distance_m / 1000) + " km",
          }))}
          period={month}
        >
          <BarChart
            categories={movement.by_sport.map((item) => item.sport)}
            primary={{
              name: "Distance",
              values: movement.by_sport.map((item) => item.distance_m / 1000),
              color: "var(--accent)",
              formatter: (value) => number(value) + " km",
            }}
            orientation="horizontal"
            categorical
            height={220}
          />
        </MetricPanel>
      {:else}<p class="empty">No canonical movement records.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Body orbit"
      title="Observation coverage"
      summary={health ? health.observed_days + " days" : "No records"}
      tone="quiet"
    >
      {#if health?.metric_averages.length}
        <MetricPanel
          metricId="daily_health.observed_days"
          label="Observation coverage"
          unit="%"
          method={report.sections.daily_health.schema}
          coverage={{
            expected_periods: periodDays,
            observed_periods: health.observed_days,
          }}
          rowHeader="Metric"
          rows={healthRows}
          period={month}
        >
          <CoverageBars rows={healthRows} />
        </MetricPanel>
      {:else}<p class="empty">No canonical daily-health observations.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Library orbit"
      title="Events and completions"
      summary={media ? media.event_count + " events" : "No records"}
    >
      {#if media?.by_kind.length}
        <MetricPanel
          metricId="media.event_count"
          label="Events and completions"
          unit="count"
          method={report.sections.media.schema}
          rowHeader="Kind"
          rows={media.by_kind.map((item) => ({
            label: item.kind,
            value: item.event_count,
            display:
              item.event_count +
              " events · " +
              item.completed_count +
              " completed",
          }))}
          period={month}
        >
          <BarChart
            categories={media.by_kind.map((item) => item.kind)}
            primary={{
              name: "Events",
              values: media.by_kind.map((item) => item.event_count),
              color: "var(--accent)",
            }}
            secondary={{
              name: "Completed",
              values: media.by_kind.map((item) => item.completed_count),
              color: "var(--accent-2)",
            }}
            categorical
            height={220}
          />
        </MetricPanel>
      {:else}<p class="empty">No canonical media events.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Ledger orbit"
      title="Spend by category"
      summary={expenses ? expenses.expense_count + " records" : "No records"}
      tone="quiet"
    >
      {#if categoryTotals.length}
        <MetricPanel
          metricId="expenses.amount_minor"
          label="Spend by category"
          unit={primaryCurrency + " minor"}
          method={report.sections.expenses.schema}
          rowHeader="Category"
          rows={categoryTotals.map((item) => ({
            label: item.category,
            value: item.amount_minor,
            display: formatMoney(
              item.amount_minor,
              primaryCurrency,
              primaryExponent,
            ),
          }))}
          period={month}
        >
          <BarChart
            categories={categoryTotals.map((item) => item.category)}
            primary={{
              name: primaryCurrency,
              values: categoryTotals.map((item) => item.amount_minor),
              color: "var(--accent)",
              formatter: (value) =>
                formatMoney(value, primaryCurrency, primaryExponent),
            }}
            orientation="horizontal"
            categorical
            height={230}
          />
        </MetricPanel>
      {:else}<p class="empty">No canonical expenses.</p>{/if}
    </ReportMetricCard>
  </div>

  <section class="cycle-evidence" aria-labelledby="cycle-evidence-title">
    <header>
      <div>
        <p class="kicker">Seasonal notes</p>
        <h3 id="cycle-evidence-title">The month in evidence.</h3>
      </div>
      <span>{evidence.length} observations</span>
    </header>
    <ReportReceipt rows={evidence} {theme} />
  </section>
</section>

<style>
  .phenology-reports {
    display: grid;
    gap: 1.25rem;
    min-width: 0;
  }
  .phenology-reports > * {
    min-width: 0;
  }

  h2,
  h3,
  p {
    margin: 0;
  }

  .phenology-heading {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
  }

  h2 {
    max-width: 43rem;
    font-size: clamp(2.4rem, 7vw, 5.5rem);
    letter-spacing: -0.11em;
    line-height: 0.88;
  }

  .phenology-heading p:last-child {
    max-width: 36rem;
    margin-top: 0.8rem;
    color: var(--text-muted);
    line-height: 1.5;
  }

  .kicker {
    color: var(--accent);
    font-size: 0.66rem;
    font-weight: 750;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }

  .phenology-heading > span,
  .cycle-evidence > header > span {
    color: var(--text-muted);
    font-size: 0.72rem;
  }

  .cycle-stage {
    display: grid;
    padding: 1rem;
    border: 1px solid var(--border);
    border-radius: 1.4rem;
    background:
      radial-gradient(
        circle at 50% 20%,
        color-mix(in srgb, var(--accent) 20%, transparent),
        transparent 48%
      ),
      var(--surface);
  }

  .orbit-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
  }

  .orbit-grid :global(.report-metric-card) {
    border-radius: 1rem;
  }

  .cycle-evidence {
    display: grid;
    gap: 0.8rem;
    border-top: 1px solid var(--border);
    padding-top: 1rem;
  }

  .cycle-evidence > header {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
  }

  .cycle-evidence h3 {
    font-size: 1.3rem;
    letter-spacing: -0.04em;
  }

  .empty {
    color: var(--text-muted);
    font-size: 0.78rem;
  }

  @media (max-width: 768px) {
    .phenology-heading,
    .cycle-evidence > header {
      display: grid;
    }

    .orbit-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
