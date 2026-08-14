<script lang="ts">
  import BarChart from "../components/BarChart.svelte";
  import ReportCoverage from "../components/ReportCoverage.svelte";
  import ReportReceipt from "../components/ReportReceipt.svelte";
  import type { ReportEvidenceRow } from "../../report";
  import type { DesignLanguage } from "../../themes";
  import ReportFactGrid from "../components/ReportFactGrid.svelte";
  import ReportMetricCard from "../components/ReportMetricCard.svelte";
  import MetricPanel from "../../MetricPanel.svelte";
  import type { PanelRow } from "../../metric-panel";
  import { formatMetricValue } from "../../format";
  import {
    reportPeriodDays,
    reportSectionData,
    type ReportThemeProps,
  } from "../../report";

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
  const movementRows = $derived<PanelRow[]>(
    (movement?.by_sport ?? []).map((item) => ({
      label: item.sport,
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
      label: item.sport,
      value: number(item.distance_m / 1000) + " km",
      detail:
        item.activity_count + " sessions · " + formatDuration(item.duration_s),
    })),
    ...(sleep
      ? [
          {
            label: "Sleep",
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
      label: item.metric,
      value: formatMetricValue(item.value, item.unit) + " " + item.unit,
      detail: item.observed_days + " observed days",
    })),
    ...(media
      ? [
          {
            label: "Library events",
            value: media.event_count + " events",
            detail: media.completed_count + " completed",
          },
        ]
      : []),
    ...categoryTotals.map((item) => ({
      label: item.category,
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

<section class="journal-reports" aria-labelledby="journal-reports-title">
  <header class="journal-heading">
    <div>
      <p class="date">{month}</p>
      <p class="kicker">Field report · monthly entry</p>
      <h2 id="journal-reports-title">What the month held.</h2>
    </div>
    <p class="note">
      A dated record of observed domains, with no readiness score and no
      invented narrative.
    </p>
  </header>

  <ReportCoverage {report} />

  <section class="journal-observations" aria-label="Observed monthly domains">
    <ReportMetricCard
      label="Movement observation"
      title="Routes and distances"
      summary={movement ? movement.activity_count + " sessions" : "No record"}
      tone="feature"
    >
      {#if movement?.by_sport.length}
        <MetricPanel
          metricId="movement.distance_m"
          label="Routes and distances"
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
            height={230}
          />
        </MetricPanel>
      {:else}<p class="empty">No movement was recorded.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Rest observation"
      title="The shape of sleep"
      summary={sleep ? formatDuration(sleep.average_asleep_s) : "No record"}
    >
      {#if sleep}
        <MetricPanel
          metricId="sleep.stage_seconds"
          label="The shape of sleep"
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
            height={230}
          />
        </MetricPanel>
        <ReportFactGrid
          facts={[
            { label: "Average", value: formatDuration(sleep.average_asleep_s) },
            {
              label: "Efficiency",
              value: Math.round(sleep.average_efficiency * 100) + "%",
            },
            {
              label: "Structure",
              value:
                sleep.main_sleep_count + " main / " + sleep.nap_count + " naps",
            },
          ]}
        />
      {:else}<p class="empty">No sleep was recorded.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Body observation"
      title="Days with evidence"
      summary={health ? health.observed_days + " observed days" : "No record"}
      tone="quiet"
    >
      {#if health?.metric_averages.length}
        <MetricPanel
          metricId="daily_health.observed_days"
          label="Days with evidence"
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
            height={230}
          />
        </MetricPanel>
      {:else}<p class="empty">No body metrics were recorded.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Library observation"
      title="Things that moved"
      summary={media ? media.event_count + " events" : "No record"}
      tone="quiet"
    >
      {#if media?.by_kind.length}
        <MetricPanel
          metricId="media.event_count"
          label="Things that moved"
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
            height={230}
          />
        </MetricPanel>
      {:else}<p class="empty">No library events were recorded.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Ledger observation"
      title="What spending repeated"
      summary={expenses ? expenses.expense_count + " entries" : "No record"}
      tone="feature"
    >
      {#if categoryTotals.length}
        <MetricPanel
          metricId="expenses.amount_minor"
          label="What spending repeated"
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
            height={240}
          />
        </MetricPanel>
      {:else}<p class="empty">No spending entries were recorded.</p>{/if}
    </ReportMetricCard>
  </section>

  <section class="journal-record" aria-labelledby="journal-record-title">
    <header>
      <div>
        <p class="kicker">Field notes</p>
        <h3 id="journal-record-title">The observed record, in order.</h3>
      </div>
      <span>{evidence.length} entries</span>
    </header>
    <ReportReceipt rows={evidence} {theme} />
  </section>
</section>

<style>
  .journal-reports {
    display: grid;
    gap: 1.25rem;
    font-family: Georgia, "Times New Roman", serif;
  }

  h2,
  h3,
  p {
    margin: 0;
  }

  .journal-heading {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 15rem;
    align-items: end;
    gap: 1.5rem;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.25rem;
  }

  h2 {
    max-width: 40rem;
    font-size: clamp(2.5rem, 7vw, 5.5rem);
    font-weight: 500;
    letter-spacing: -0.1em;
    line-height: 0.88;
  }

  .date,
  .kicker {
    color: var(--accent);
    font-size: 0.68rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .note {
    color: var(--text-muted);
    font-size: 0.85rem;
    line-height: 1.5;
  }

  .journal-observations {
    display: grid;
    gap: 1rem;
  }

  .journal-observations :global(.report-metric-card) {
    border-style: dashed;
    background: color-mix(in srgb, var(--surface) 88%, var(--accent) 12%);
  }

  .journal-record {
    display: grid;
    gap: 0.8rem;
    border-top: 1px solid var(--border);
    padding-top: 1rem;
  }

  .journal-record > header {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
  }

  .journal-record h3 {
    font-size: 1.35rem;
    font-weight: 500;
  }

  .journal-record > header > span {
    color: var(--text-muted);
    font-size: 0.76rem;
  }

  .empty {
    color: var(--text-muted);
    font-size: 0.78rem;
  }

  @media (max-width: 768px) {
    .journal-heading,
    .journal-record > header {
      grid-template-columns: 1fr;
      display: grid;
    }
  }
</style>
