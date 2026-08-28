<script lang="ts">
  import BarChart from "../components/BarChart.svelte";
  import CoverageBars from "../../components/CoverageBars.svelte";
  import IconLegend from "../../components/IconLegend.svelte";
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
  import { sportLabel } from "../../domain/sport";
  import { sportIcon } from "../../domain/sport-icons";
  import { expenseCategoryLabel } from "../../view-contracts/expense-view";
  import { expenseCategoryIcon } from "../../domain/expense-icons";
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
  const periodDays = $derived(reportPeriodDays(report));
  const healthRows = $derived<PanelRow[]>(
    (health?.metric_averages ?? []).map((item) => {
      const pct = coveragePercent(item.observed_days, periodDays);
      return {
        label: healthMetricLabel(item.metric),
        icon: healthMetricIcon(item.metric),
        value: pct,
        display: pct + "%",
        breakdown: formatMetricValue(item.value, item.unit) + " " + item.unit,
      };
    }),
  );
  const movementRows = $derived<PanelRow[]>(
    (movement?.by_sport ?? []).map((item) => ({
      label: sportLabel(item.sport),
      icon: sportIcon(item.sport),
      value: item.distance_m / 1000,
      display: number(item.distance_m / 1000) + " km",
    })),
  );
  const sleepRows = $derived<PanelRow[]>(
    sleepStages.map(([stage, seconds]) => ({
      label: stage,
      value: seconds / 3600,
      display: number(seconds / 3600) + "h",
    })),
  );
  const expenseRows = $derived<PanelRow[]>(
    categoryTotals.map((item) => ({
      label: expenseCategoryLabel[item.category],
      icon: expenseCategoryIcon(item.category),
      value: item.amount_minor,
      display: formatMoney(item.amount_minor, primaryCurrency, primaryExponent),
    })),
  );
  const evidence = $derived<ReportEvidenceRow[]>([
    ...(health?.metric_averages ?? []).map((item) => ({
      label: "Health · " + healthMetricLabel(item.metric),
      value: formatMetricValue(item.value, item.unit) + " " + item.unit,
      detail: item.observed_days + " observed days",
    })),
    ...(movement?.by_sport ?? []).map((item) => ({
      label: "Movement · " + sportLabel(item.sport),
      value: number(item.distance_m / 1000) + " km",
      detail: item.activity_count + " activities",
    })),
    ...(sleep
      ? [
          {
            label: "Sleep average",
            value: formatDuration(sleep.average_asleep_s),
            detail:
              Math.round(sleep.average_efficiency * 100) +
              "% efficiency · " +
              sleep.main_sleep_count +
              " main · " +
              sleep.nap_count +
              " naps",
          },
        ]
      : []),
    ...(media
      ? [
          {
            label: "Media events",
            value: media.event_count + " events",
            detail: media.completed_count + " completed",
          },
        ]
      : []),
    ...categoryTotals.map((item) => ({
      label: "Expense · " + expenseCategoryLabel[item.category],
      value: formatMoney(item.amount_minor, primaryCurrency, primaryExponent),
      detail: item.expense_count + " records",
    })),
  ]);

  function number(value: number): string {
    return new Intl.NumberFormat(undefined, {
      maximumFractionDigits: 1,
    }).format(value);
  }
</script>

<section class="grapher-reports" aria-labelledby="grapher-reports-title">
  <header class="grapher-heading">
    <div>
      <p class="kicker">Monthly comparison / {month}</p>
      <h2 id="grapher-reports-title">Compare the canonical signals.</h2>
      <p>
        Every chart exposes its unit, coverage, table parity, and source method.
      </p>
    </div>
    <span class="period">{report.period.from} → {report.period.to}</span>
  </header>

  <ReportCoverage {report} />

  <div class="grapher-primary">
    <ReportMetricCard
      label="Daily health"
      title="Observation coverage"
      summary={health ? health.observed_days + " observed days" : "No records"}
      tone="feature"
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
          <CoverageBars rows={healthRows} max={100} />
        </MetricPanel>
        <ReportFactGrid
          facts={(health.metric_averages ?? []).slice(0, 3).map((item) => ({
            label: healthMetricLabel(item.metric),
            value: formatMetricValue(item.value, item.unit),
            detail: item.unit,
          }))}
        />
      {:else}<p class="empty">No canonical daily-health observations.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Expenses"
      title="Category comparison"
      summary={expenses ? expenses.expense_count + " records" : "No records"}
      tone="feature"
    >
      {#if categoryTotals.length}
        <MetricPanel
          metricId="expenses.amount_minor"
          label="Spend by category"
          unit={primaryCurrency + " minor"}
          method={report.sections.expenses.schema}
          rowHeader="Category"
          rows={expenseRows}
          period={month}
        >
          <IconLegend rows={expenseRows} />
          <BarChart
            categories={expenseRows.map((row) => row.label)}
            primary={{
              name: primaryCurrency,
              values: expenseRows.map((row) => row.value),
              color: "var(--accent-2)",
              formatter: (value) =>
                formatMoney(value, primaryCurrency, primaryExponent),
            }}
            orientation="horizontal"
            height={270}
          />
        </MetricPanel>
      {:else}<p class="empty">
          No canonical expenses in {primaryCurrency}.
        </p>{/if}
    </ReportMetricCard>
  </div>

  <div class="grapher-grid">
    <ReportMetricCard
      label="Movement"
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
          rows={movementRows}
          period={month}
        >
          <IconLegend rows={movementRows} />
          <BarChart
            categories={movementRows.map((row) => row.label)}
            primary={{
              name: "Distance",
              values: movementRows.map((row) => row.value),
              color: "var(--accent)",
              formatter: (value) => number(value) + " km",
            }}
            orientation="horizontal"
            height={220}
          />
        </MetricPanel>
      {:else}<p class="empty">No canonical movement records.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Sleep"
      title="Stage composition"
      summary={sleep ? formatDuration(sleep.average_asleep_s) : "No records"}
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
            height={220}
          />
        </MetricPanel>
      {:else}<p class="empty">No canonical sleep records.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Media"
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
  </div>

  <section class="grapher-evidence" aria-labelledby="grapher-evidence-title">
    <header>
      <div>
        <p class="kicker">Exact evidence</p>
        <h3 id="grapher-evidence-title">The comparison remains auditable.</h3>
      </div>
      <span>{evidence.length} rows</span>
    </header>
    <ReportReceipt rows={evidence} {theme} />
  </section>
</section>

<style>
  .grapher-reports {
    display: grid;
    gap: 1rem;
    font-family: "IBM Plex Mono", "SFMono-Regular", monospace;
    min-width: 0;
  }
  .grapher-reports > * {
    min-width: 0;
  }

  h2,
  h3,
  p {
    margin: 0;
  }

  .grapher-heading {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
    border-bottom: 3px solid var(--text);
    padding-bottom: 1rem;
  }

  h2 {
    max-width: 43rem;
    font-size: clamp(2.8rem, 8vw, 6rem);
    letter-spacing: -0.12em;
    line-height: 0.84;
  }

  .grapher-heading p:last-child {
    max-width: 38rem;
    margin-top: 0.8rem;
    color: var(--text-muted);
    font-family: var(--font-sans);
    line-height: 1.5;
  }

  .kicker {
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .period,
  .grapher-evidence > header > span {
    color: var(--text-muted);
    font-size: 0.72rem;
  }

  .grapher-primary,
  .grapher-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
  }

  .grapher-primary :global(.report-metric-card) {
    border-top: 3px solid var(--text);
  }

  .grapher-evidence {
    display: grid;
    gap: 0.8rem;
    border-top: 3px solid var(--text);
    padding-top: 1rem;
  }

  .grapher-evidence > header {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
  }

  .grapher-evidence h3 {
    font-family: var(--font-sans);
    font-size: 1.3rem;
  }

  .empty {
    color: var(--text-muted);
    font-family: var(--font-sans);
    font-size: 0.78rem;
  }

  @media (max-width: 768px) {
    .grapher-heading,
    .grapher-evidence > header {
      display: grid;
    }

    .grapher-primary,
    .grapher-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
