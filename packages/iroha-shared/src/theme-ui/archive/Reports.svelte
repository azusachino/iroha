<script lang="ts">
  import BarChart from "../components/BarChart.svelte";
  import ReportCoverage from "../components/ReportCoverage.svelte";
  import ReportReceipt from "../components/ReportReceipt.svelte";
  import type { ReportEvidenceRow } from "../../report";
  import type { DesignLanguage } from "../../themes";
  import ReportFactGrid from "../components/ReportFactGrid.svelte";
  import ReportMetricCard from "../components/ReportMetricCard.svelte";
  import MetricPanel from "../../MetricPanel.svelte";
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
    ...(movement?.by_sport ?? []).map((item) => ({
      label: "movement/" + item.sport,
      value: number(item.distance_m / 1000) + " km",
      detail:
        item.activity_count +
        " activities · " +
        formatDuration(item.duration_s),
    })),
    ...(sleep
      ? [
          {
            label: "sleep/summary",
            value: formatDuration(sleep.average_asleep_s),
            detail:
              sleep.session_count +
              " sessions · " +
              Math.round(sleep.average_efficiency * 100) +
              "% efficiency",
          },
        ]
      : []),
    ...(health?.metric_averages ?? []).map((item) => ({
      label: "health/" + item.metric,
      value: formatMetricValue(item.value, item.unit) + " " + item.unit,
      detail: item.observed_days + " observed days",
    })),
    ...(media
      ? [
          {
            label: "media/events",
            value: media.event_count + " events",
            detail:
              media.completed_count +
              " completed · " +
              media.rated_count +
              " rated",
          },
        ]
      : []),
    ...categories.map((item) => ({
      label: "expenses/" + item.category,
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

<section class="archive-reports" aria-labelledby="archive-reports-title">
  <header class="archive-heading">
    <div>
      <p class="kicker">Archive / monthly report</p>
      <h2 id="archive-reports-title">The report, exactly as generated.</h2>
    </div>
    <span>{report.period.from} → {report.period.to} · {month}</span>
  </header>

  <ReportCoverage {report} />

  <section class="archive-index" aria-label="Canonical domain index">
    <ReportMetricCard
      label="Canonical movement"
      title="Distance by sport"
      summary={movement ? movement.activity_count + " records" : "Empty"}
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
            height={230}
          />
        </MetricPanel>
      {:else}<p class="empty">No movement rows in this envelope.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Canonical rest"
      title="Sleep-stage composition"
      summary={sleep ? sleep.session_count + " sessions" : "Empty"}
    >
      {#if sleep}
        <MetricPanel
          metricId="sleep.stage_seconds"
          label="Sleep-stage composition"
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
              color: "var(--accent-2)",
              formatter: (value) => number(value) + "h",
            }}
            categorical
            height={230}
          />
        </MetricPanel>
      {:else}<p class="empty">No sleep rows in this envelope.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Canonical health"
      title="Observed metric days"
      summary={health ? health.observed_days + " days" : "Empty"}
      tone="quiet"
    >
      {#if health?.metric_averages.length}
        <MetricPanel
          metricId="daily_health.observed_days"
          label="Observed metric days"
          unit="days"
          method={report.sections.daily_health.schema}
          coverage={{
            expected_periods: periodDays,
            observed_periods: health.observed_days,
          }}
          rowHeader="Metric"
          rows={health.metric_averages.map((item) => ({
            label: item.metric,
            value: item.observed_days,
            display: item.observed_days + " d",
            breakdown:
              formatMetricValue(item.value, item.unit) + " " + item.unit,
          }))}
          period={month}
        >
          <BarChart
            categories={health.metric_averages.map((item) => item.metric)}
            primary={{
              name: "Observed days",
              values: health.metric_averages.map((item) => item.observed_days),
              color: "var(--accent)",
            }}
            orientation="horizontal"
            categorical
            height={230}
          />
        </MetricPanel>
      {:else}<p class="empty">No health rows in this envelope.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Canonical media"
      title="Events and completions"
      summary={media ? media.event_count + " events" : "Empty"}
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
            height={230}
          />
        </MetricPanel>
      {:else}<p class="empty">No media rows in this envelope.</p>{/if}
    </ReportMetricCard>

    <ReportMetricCard
      label="Canonical expenses"
      title="Spend by category"
      summary={expenses ? expenses.expense_count + " records" : "Empty"}
      tone="feature"
    >
      {#if categories.length}
        <MetricPanel
          metricId="expenses.amount_minor"
          label="Spend by category"
          unit={primaryCurrency + " minor"}
          method={report.sections.expenses.schema}
          rowHeader="Category"
          rows={categories.map((item) => ({
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
            categories={categories.map((item) => item.category)}
            primary={{
              name: primaryCurrency,
              values: categories.map((item) => item.amount_minor),
              color: "var(--accent)",
              formatter: (value) =>
                formatMoney(value, primaryCurrency, primaryExponent),
            }}
            orientation="horizontal"
            categorical
            height={250}
          />
        </MetricPanel>
      {:else}<p class="empty">No expense rows in this envelope.</p>{/if}
    </ReportMetricCard>
  </section>

  <section class="archive-evidence" aria-labelledby="archive-evidence-title">
    <header>
      <div>
        <p class="kicker">Envelope evidence</p>
        <h3 id="archive-evidence-title">Indexed rows behind the report.</h3>
      </div>
      <span>{evidence.length} rows</span>
    </header>
    <ReportFactGrid
      facts={[
        { label: "Schema", value: report.schema },
        { label: "Period", value: report.period.month },
        {
          label: "Sections",
          value: Object.keys(report.sections).length + " domains",
        },
      ]}
    />
    <ReportReceipt rows={evidence} {theme} />
  </section>
</section>

<style>
  .archive-reports {
    display: grid;
    gap: 1.25rem;
    font-family: var(--font-mono);
  }

  h2,
  h3,
  p {
    margin: 0;
  }

  .archive-heading {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
    border-bottom: 3px double var(--text);
    padding-bottom: 0.8rem;
  }

  h2 {
    font-family: var(--font-sans);
    font-size: clamp(2.4rem, 6vw, 5rem);
    letter-spacing: -0.11em;
    line-height: 0.88;
  }

  .archive-heading > span,
  .archive-evidence > header > span {
    color: var(--text-muted);
    font-size: 0.72rem;
  }

  .kicker {
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }

  .archive-index {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
    border-top: 3px double var(--text);
    padding-top: 1rem;
  }

  .archive-index :global(.report-metric-card:last-child) {
    grid-column: 1 / -1;
  }

  .archive-index :global(.report-metric-card) {
    border-width: 2px;
    border-radius: 0;
  }

  .archive-evidence {
    display: grid;
    gap: 0.8rem;
    border-top: 3px double var(--text);
    padding-top: 1rem;
  }

  .archive-evidence > header {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
  }

  .archive-evidence h3 {
    font-family: var(--font-sans);
    font-size: 1.3rem;
  }

  .empty {
    color: var(--text-muted);
    font-size: 0.78rem;
  }

  @media (max-width: 760px) {
    .archive-heading,
    .archive-evidence > header {
      display: grid;
    }

    .archive-index {
      grid-template-columns: 1fr;
    }

    .archive-index :global(.report-metric-card:last-child) {
      grid-column: auto;
    }
  }
</style>
