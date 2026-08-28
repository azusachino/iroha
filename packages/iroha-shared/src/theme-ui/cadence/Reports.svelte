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
  import { healthMetricColorVar } from "../../domain/health-metric-colors";
  import { sportLabel, sportColorVar } from "../../domain/sport";
  import { sportIcon } from "../../domain/sport-icons";
  import { expenseCategoryLabel } from "../../view-contracts/expense-view";
  import { expenseCategoryIcon } from "../../domain/expense-icons";
  import { categoryColorVar } from "../../domain/category-color";
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
  const categories = $derived(
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
  const evidence = $derived<ReportEvidenceRow[]>([
    ...(media
      ? [
          {
            label: "Library signal",
            value: media.event_count + " events",
            detail:
              media.completed_count +
              " completed · " +
              media.rated_count +
              " rated",
          },
        ]
      : []),
    ...(movement?.by_sport ?? []).map((item) => ({
      label: "Route · " + sportLabel(item.sport),
      value: number(item.distance_m / 1000) + " km",
      detail:
        item.activity_count + " sessions · " + formatDuration(item.duration_s),
    })),
    ...(sleep
      ? [
          {
            label: "Rest signal",
            value: formatDuration(sleep.average_asleep_s),
            detail: Math.round(sleep.average_efficiency * 100) + "% efficiency",
          },
        ]
      : []),
    ...(health?.metric_averages ?? []).map((item) => ({
      label: "Body · " + healthMetricLabel(item.metric),
      value: formatMetricValue(item.value, item.unit) + " " + item.unit,
      detail: item.observed_days + " observed days",
    })),
    ...(categories ?? []).map((item) => ({
      label: "Spend band · " + expenseCategoryLabel[item.category],
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
  const movementRows = $derived<PanelRow[]>(
    (movement?.by_sport ?? []).map((item) => ({
      label: sportLabel(item.sport),
      icon: sportIcon(item.sport),
      colorVar: sportColorVar(item.sport),
      value: item.distance_m / 1000,
      display: number(item.distance_m / 1000) + " km",
    })),
  );
  const expenseRows = $derived<PanelRow[]>(
    categories.map((item) => ({
      label: expenseCategoryLabel[item.category],
      icon: expenseCategoryIcon(item.category),
      colorVar: categoryColorVar(item.category),
      value: item.amount_minor,
      display: formatMoney(item.amount_minor, primaryCurrency, primaryExponent),
    })),
  );

  function number(value: number): string {
    return new Intl.NumberFormat(undefined, {
      maximumFractionDigits: 1,
    }).format(value);
  }
</script>

<section class="sound-reports" aria-labelledby="sound-reports-title">
  <header class="sound-heading">
    <div>
      <p class="kicker">Signal report / {month}</p>
      <h2 id="sound-reports-title">A month with a pulse.</h2>
      <p>
        Intensity, quiet, and recurring bands stay visible as separate signals.
      </p>
    </div>
    <span>{report.period.from} → {report.period.to}</span>
  </header>

  <ReportCoverage {report} />

  <div class="signal-stack">
    <ReportMetricCard
      label="Library signal"
      title="Events and completions"
      summary={media ? media.event_count + " events" : "Silence"}
      tone="feature"
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
            primaryType="line"
            categorical
            height={280}
          />
        </MetricPanel>
      {:else}<p class="empty">No canonical media events.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Movement signal"
      title="Distance intensity"
      summary={movement ? movement.activity_count + " sessions" : "Silence"}
    >
      {#if movement?.by_sport.length}
        <MetricPanel
          metricId="movement.distance_m"
          label="Distance intensity"
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
              colors: movementRows.map((row) => `var(${row.colorVar})`),
              formatter: (value) => number(value) + " km",
            }}
            orientation="horizontal"
            height={250}
          />
        </MetricPanel>
      {:else}<p class="empty">No canonical movement records.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Recovery signal"
      title="Stage bands"
      summary={sleep ? formatDuration(sleep.average_asleep_s) : "Silence"}
      tone="quiet"
    >
      {#if sleep}
        <MetricPanel
          metricId="sleep.stage_seconds"
          label="Stage bands"
          unit="hours"
          method={report.sections.sleep.schema}
          rowHeader="Stage"
          rows={sleepStages.map(([stage, seconds]) => ({
            label: stage,
            value: seconds / 3600,
            display: number(seconds / 3600) + "h",
          }))}
          period={month}
        >
          <BarChart
            categories={sleepStages.map(([stage]) => stage)}
            primary={{
              name: "Hours",
              values: sleepStages.map(([, seconds]) => seconds / 3600),
              color: "var(--accent)",
              formatter: (value) => number(value) + "h",
            }}
            categorical
            height={250}
          />
        </MetricPanel>
        <ReportFactGrid
          facts={[
            {
              label: "Efficiency",
              value: Math.round(sleep.average_efficiency * 100) + "%",
            },
            { label: "Main", value: String(sleep.main_sleep_count) },
            { label: "Naps", value: String(sleep.nap_count) },
          ]}
        />
      {:else}<p class="empty">No canonical sleep records.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Body signal"
      title="Observed coverage"
      summary={health ? health.observed_days + " days" : "Silence"}
      tone="quiet"
    >
      {#if health?.metric_averages.length}
        <MetricPanel
          metricId="daily_health.observed_days"
          label="Observed coverage"
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
      {:else}<p class="empty">No canonical daily-health observations.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Ledger signal"
      title="Spend bands"
      summary={expenses ? expenses.expense_count + " records" : "Silence"}
    >
      {#if categories.length}
        <MetricPanel
          metricId="expenses.amount_minor"
          label="Spend bands"
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
              colors: expenseRows.map((row) => `var(${row.colorVar})`),
              formatter: (value) =>
                formatMoney(value, primaryCurrency, primaryExponent),
            }}
            orientation="horizontal"
            height={260}
          />
        </MetricPanel>
      {:else}<p class="empty">No canonical expenses.</p>{/if}
    </ReportMetricCard>
  </div>

  <section class="signal-evidence" aria-labelledby="signal-evidence-title">
    <header>
      <div>
        <p class="kicker">Signal index</p>
        <h3 id="signal-evidence-title">The quiet details remain available.</h3>
      </div>
      <span>{evidence.length} observations</span>
    </header>
    <ReportReceipt rows={evidence} {theme} />
  </section>
</section>

<style>
  .sound-reports {
    display: grid;
    gap: 1.1rem;
    min-width: 0;
  }
  .sound-reports > * {
    min-width: 0;
  }

  h2,
  h3,
  p {
    margin: 0;
  }

  .sound-heading {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
  }

  h2 {
    font-size: clamp(2.8rem, 9vw, 7rem);
    letter-spacing: -0.13em;
    line-height: 0.82;
    text-shadow: 0 0 1.5rem color-mix(in srgb, var(--accent) 25%, transparent);
  }

  .sound-heading p:last-child {
    margin-top: 0.8rem;
    color: var(--text-muted);
  }

  .kicker {
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.66rem;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }

  .sound-heading > span,
  .signal-evidence > header > span {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.72rem;
  }

  .signal-stack {
    display: grid;
    gap: 1rem;
    padding: 0.8rem;
    border: 1px solid var(--border);
    border-radius: 1rem;
    background:
      radial-gradient(
        circle at 50% 0,
        color-mix(in srgb, var(--accent) 18%, transparent),
        transparent 58%
      ),
      var(--surface);
    box-shadow: 0 0 2rem color-mix(in srgb, var(--accent) 10%, transparent);
  }

  .signal-stack :global(.report-metric-card) {
    border-inline-width: 3px;
    border-color: color-mix(in srgb, var(--accent-2) 40%, var(--border));
  }

  .signal-evidence {
    display: grid;
    gap: 0.8rem;
    border-top: 1px solid var(--border);
    padding-top: 1rem;
  }

  .signal-evidence > header {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
  }

  .signal-evidence h3 {
    font-size: 1.25rem;
  }

  .empty {
    color: var(--text-muted);
    font-size: 0.78rem;
  }

  @media (max-width: 768px) {
    .sound-heading,
    .signal-evidence > header {
      display: grid;
    }
  }
</style>
