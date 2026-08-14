<script lang="ts">
  import BarChart from "$lib/components/BarChart.svelte";
  import ReportCoverage from "$lib/components/ReportCoverage.svelte";
  import ReportEvidenceList, {
    type ReportEvidenceRow,
  } from "$lib/components/ReportEvidenceList.svelte";
  import ReportFactGrid from "$lib/components/ReportFactGrid.svelte";
  import ReportMetricCard from "$lib/components/ReportMetricCard.svelte";
  import MetricPanel from "@iroha/shared/MetricPanel.svelte";
  import type { PanelRow } from "@iroha/shared/metric-panel";
  import { formatMetricValue } from "$lib/format";
  import {
    reportPeriodDays,
    reportSectionData,
    type ReportThemeProps,
  } from "$lib/report-view";

  let {
    month,
    report,
    primaryCurrency,
    primaryExponent,
    formatMoney,
    formatDuration,
  }: ReportThemeProps = $props();

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
  const movementRows = $derived<PanelRow[]>(
    (movement?.by_sport ?? []).map((item) => ({
      label: item.sport,
      value: item.distance_m / 1000,
      display: number(item.distance_m / 1000) + " km",
    })),
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
  const healthRows = $derived<PanelRow[]>(
    (health?.metric_averages ?? []).map((item) => ({
      label: item.metric,
      value: item.observed_days,
      display: item.observed_days + " d",
      breakdown: formatMetricValue(item.value, item.unit) + " " + item.unit,
    })),
  );
  const expenseRows = $derived<PanelRow[]>(
    categoryTotals.map((item) => ({
      label: item.category,
      value: item.amount_minor,
      display: formatMoney(item.amount_minor, primaryCurrency, primaryExponent),
    })),
  );
  const evidence = $derived<ReportEvidenceRow[]>([
    ...(movement?.by_sport ?? []).map((item) => ({
      label: "Movement · " + item.sport,
      value: number(item.distance_m / 1000) + " km",
      detail:
        item.activity_count +
        " activities · " +
        formatDuration(item.duration_s),
    })),
    ...(sleep
      ? [
          {
            label: "Sleep architecture",
            value: formatDuration(sleep.average_asleep_s),
            detail:
              sleep.main_sleep_count +
              " main sleeps · " +
              sleep.nap_count +
              " naps · " +
              Math.round(sleep.average_efficiency * 100) +
              "% efficiency",
          },
        ]
      : []),
    ...(health?.metric_averages ?? []).map((item) => ({
      label: "Health · " + item.metric,
      value: formatMetricValue(item.value, item.unit) + " " + item.unit,
      detail: item.observed_days + " observed days",
    })),
    ...(media
      ? [
          {
            label: "Media events",
            value: media.event_count + " events",
            detail:
              media.completed_count +
              " completed · " +
              media.rated_count +
              " rated",
          },
        ]
      : []),
    ...categoryTotals.map((item) => ({
      label: "Expense · " + item.category,
      value: formatMoney(item.amount_minor, primaryCurrency, primaryExponent),
      detail: item.expense_count + " records",
    })),
  ]);
  const periodDays = $derived(reportPeriodDays(report));

  function number(value: number): string {
    return new Intl.NumberFormat(undefined, {
      maximumFractionDigits: 1,
    }).format(value);
  }
</script>

<section class="atlas-reports" aria-labelledby="atlas-reports-title">
  <header class="atlas-heading">
    <div>
      <p class="kicker">Monthly survey · {month}</p>
      <h2 id="atlas-reports-title">Read the month as a territory.</h2>
      <p>
        The map is the hierarchy: plotted domains first, canonical evidence
        below.
      </p>
    </div>
    <span class="coordinate">{report.period.from} → {report.period.to}</span>
  </header>

  <ReportCoverage {report} />

  <div class="atlas-grid">
    <ReportMetricCard
      label="Movement"
      title="Distance by sport"
      summary={movement ? movement.activity_count + " sessions" : "No records"}
      tone="feature"
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
            height={240}
          />
        </MetricPanel>
        <ReportFactGrid
          facts={[
            {
              label: "Distance",
              value: number(movement.distance_m / 1000) + " km",
            },
            { label: "Duration", value: formatDuration(movement.duration_s) },
            {
              label: "Coverage",
              value: movement.distance_activity_count + " with distance",
            },
          ]}
        />
      {:else}<p class="empty">No canonical movement records.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Sleep"
      title="Sleep-stage composition"
      summary={sleep
        ? sleep.main_sleep_count + " main · " + sleep.nap_count + " naps"
        : "No records"}
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
            height={240}
          />
        </MetricPanel>
        <ReportFactGrid
          facts={[
            { label: "Average", value: formatDuration(sleep.average_asleep_s) },
            {
              label: "Efficiency",
              value: Math.round(sleep.average_efficiency * 100) + "%",
            },
            { label: "Sessions", value: String(sleep.session_count) },
          ]}
        />
      {:else}<p class="empty">No canonical sleep records.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Daily health"
      title="Observation coverage"
      summary={health ? health.observed_days + " observed days" : "No records"}
      tone="quiet"
    >
      {#if health?.metric_averages.length}
        <MetricPanel
          metricId="daily_health.observed_days"
          label="Observation coverage"
          unit="days"
          method={report.sections.daily_health.schema}
          coverage={{
            expected_periods: periodDays,
            observed_periods: health.observed_days,
          }}
          rowHeader="Metric"
          rows={healthRows}
          period={month}
        >
          <BarChart
            categories={health.metric_averages.map((item) => item.metric)}
            primary={{
              name: "Observed days",
              values: health.metric_averages.map((item) => item.observed_days),
              color: "var(--accent)",
              formatter: (value) => value + "d",
            }}
            orientation="horizontal"
            categorical
            height={240}
          />
        </MetricPanel>
      {:else}<p class="empty">No canonical daily-health observations.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Media"
      title="Events and completions"
      summary={media ? media.event_count + " events" : "No records"}
      tone="quiet"
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
            height={240}
          />
        </MetricPanel>
      {:else}<p class="empty">No canonical media events.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Expenses"
      title="Spend by category"
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
            height={260}
          />
        </MetricPanel>
      {:else}<p class="empty">
          No canonical expenses in {primaryCurrency}.
        </p>{/if}
    </ReportMetricCard>
  </div>

  <section class="atlas-evidence" aria-labelledby="atlas-evidence-title">
    <header>
      <div>
        <p class="kicker">Evidence ledger</p>
        <h3 id="atlas-evidence-title">The plotted values, grounded.</h3>
      </div>
      <span>{evidence.length} rows</span>
    </header>
    <ReportEvidenceList rows={evidence} />
  </section>
</section>

<style>
  .atlas-reports {
    display: grid;
    gap: 1.25rem;
  }

  .atlas-heading {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
    border-bottom: 2px solid var(--text);
    padding-bottom: 1rem;
  }

  h2,
  h3,
  p {
    margin: 0;
  }

  h2 {
    max-width: 40rem;
    font-size: clamp(2.2rem, 6vw, 5rem);
    letter-spacing: -0.1em;
    line-height: 0.88;
  }

  .atlas-heading p:last-child {
    max-width: 34rem;
    margin-top: 0.8rem;
    color: var(--text-muted);
    line-height: 1.5;
  }

  .kicker {
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.66rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .coordinate,
  .atlas-evidence > header > span {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.72rem;
  }

  .atlas-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
    padding: 1rem;
    background-image:
      linear-gradient(
        color-mix(in srgb, var(--accent) 8%, transparent) 1px,
        transparent 1px
      ),
      linear-gradient(
        90deg,
        color-mix(in srgb, var(--accent) 8%, transparent) 1px,
        transparent 1px
      );
    background-size: 18px 18px;
  }

  .atlas-grid :global(.report-metric-card:last-child) {
    grid-column: 1 / -1;
  }

  .atlas-evidence {
    display: grid;
    gap: 0.8rem;
    border-top: 2px solid var(--text);
    padding-top: 1rem;
  }

  .atlas-evidence > header {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
  }

  .atlas-evidence h3 {
    font-size: 1.25rem;
    letter-spacing: -0.04em;
  }

  .empty {
    color: var(--text-muted);
    font-size: 0.78rem;
  }

  @media (max-width: 760px) {
    .atlas-heading,
    .atlas-evidence > header {
      display: grid;
    }

    .atlas-grid {
      grid-template-columns: 1fr;
    }

    .atlas-grid :global(.report-metric-card:last-child) {
      grid-column: auto;
    }
  }
</style>
