<script lang="ts">
  import BarChart from "$lib/components/BarChart.svelte";
  import MetricPanel from "@iroha/shared/MetricPanel.svelte";
  import type { PanelRow } from "@iroha/shared/metric-panel";
  import type { MonthlyReport } from "$lib/api";
  import { formatMetricValue } from "$lib/format";

  let {
    report,
    primaryCurrency,
    primaryExponent,
    formatMoney,
    formatDuration,
  }: {
    report: MonthlyReport;
    primaryCurrency: string;
    primaryExponent: number;
    formatMoney: (
      amountMinor: number,
      currency: string,
      exponent: number,
    ) => string;
    formatDuration: (seconds: number) => string;
  } = $props();

  const movement = $derived(report.sections.movement.data);
  const sleep = $derived(report.sections.sleep.data);
  const health = $derived(report.sections.daily_health.data);
  const media = $derived(report.sections.media.data);
  const expenses = $derived(report.sections.expenses.data);
  const categoryTotals = $derived(
    [...(expenses?.by_category ?? [])]
      .filter((item) => item.currency === primaryCurrency)
      .sort((a, b) => b.amount_minor - a.amount_minor),
  );

  const hours = (seconds: number) => seconds / 3600;
  const number = (value: number) =>
    new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 }).format(
      value,
    );

  const month = $derived(report.period.month);
  // Exact rows behind each chart. They mirror what the chart plots, so table
  // parity and CSV export cannot disagree with the visual.
  const movementRows = $derived<PanelRow[]>(
    (movement?.by_sport ?? []).map((item) => ({
      label: item.sport,
      value: item.distance_m / 1000,
      display: `${number(item.distance_m / 1000)} km`,
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
      value: hours(seconds),
      display: `${number(hours(seconds))}h`,
    })),
  );
  const healthRows = $derived<PanelRow[]>(
    (health?.metric_averages ?? []).map((item) => ({
      label: item.metric,
      breakdown: `${formatMetricValue(item.value, item.unit)} ${item.unit}`,
      value: item.observed_days,
      display: `${item.observed_days} d`,
    })),
  );
  const mediaRows = $derived<PanelRow[]>(
    (media?.by_kind ?? []).flatMap((item) => [
      {
        label: item.kind,
        breakdown: "events",
        value: item.event_count,
        display: String(item.event_count),
      },
      {
        label: item.kind,
        breakdown: "completed",
        value: item.completed_count,
        display: String(item.completed_count),
      },
    ]),
  );
  const expenseRows = $derived<PanelRow[]>(
    categoryTotals.map((item) => ({
      label: item.category,
      value: item.amount_minor,
      display: formatMoney(item.amount_minor, primaryCurrency, primaryExponent),
    })),
  );
  const periodDays = $derived(
    Math.round(
      (Date.parse(`${report.period.to}T00:00:00Z`) -
        Date.parse(`${report.period.from}T00:00:00Z`)) /
        86_400_000,
    ),
  );
</script>

<section class="domain-charts" aria-label="Monthly domain charts">
  <div class="coverage" aria-label="Domain availability">
    <span>Canonical coverage</span>
    {#each Object.entries(report.sections) as [domain, section]}
      <b class:available={section.state === "available"}
        >{domain.replace("daily_health", "health").replace("_", " ")} · {section.state}</b
      >
    {/each}
  </div>
  <article class="domain movement">
    <header>
      <div>
        <p>Movement</p>
        <h3>Distance by sport</h3>
      </div>
      {#if movement}<dl>
          <div>
            <dt>Activities</dt>
            <dd>{movement.activity_count}</dd>
          </div>
          <div>
            <dt>Distance</dt>
            <dd>{number(movement.distance_m / 1000)} km</dd>
          </div>
          <div>
            <dt>Duration</dt>
            <dd>{formatDuration(movement.duration_s)}</dd>
          </div>
        </dl>{/if}
    </header>
    {#if movement?.by_sport.length}<MetricPanel
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
            formatter: (value) => `${number(value)} km`,
          }}
          orientation="horizontal"
          categorical
          height={240}
        />
      </MetricPanel>{:else}<p class="empty">
        No canonical movement records.
      </p>{/if}
  </article>

  <article class="domain sleep">
    <header>
      <div>
        <p>Sleep</p>
        <h3>Sleep-stage composition</h3>
      </div>
      {#if sleep}<dl>
          <div>
            <dt>Average</dt>
            <dd>{formatDuration(sleep.average_asleep_s)}</dd>
          </div>
          <div>
            <dt>Efficiency</dt>
            <dd>{Math.round(sleep.average_efficiency * 100)}%</dd>
          </div>
          <div>
            <dt>Main / naps</dt>
            <dd>{sleep.main_sleep_count} / {sleep.nap_count}</dd>
          </div>
        </dl>{/if}
    </header>
    {#if sleep}<MetricPanel
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
            values: sleepStages.map(([, seconds]) => hours(seconds)),
            color: "var(--accent-2)",
            formatter: (value) => `${number(value)}h`,
          }}
          categorical
          height={240}
        />
      </MetricPanel>{:else}<p class="empty">No canonical sleep records.</p>{/if}
  </article>

  <article class="domain health">
    <header>
      <div>
        <p>Daily health</p>
        <h3>Observation coverage</h3>
      </div>
      {#if health}<strong>{health.observed_days} observed days</strong>{/if}
    </header>
    {#if health?.metric_averages.length}<MetricPanel
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
            formatter: (value) => `${value}d`,
          }}
          orientation="horizontal"
          categorical
          height={240}
        />
      </MetricPanel>
      <ul>
        {#each health.metric_averages as item}<li>
            <span>{item.metric}</span><b
              >{formatMetricValue(item.value, item.unit)} {item.unit}</b
            >
          </li>{/each}
      </ul>{:else}<p class="empty">
        No canonical daily-health observations.
      </p>{/if}
  </article>

  <article class="domain media">
    <header>
      <div>
        <p>Media</p>
        <h3>Events and completions</h3>
      </div>
      {#if media}<dl>
          <div>
            <dt>Events</dt>
            <dd>{media.event_count}</dd>
          </div>
          <div>
            <dt>Completed</dt>
            <dd>{media.completed_count}</dd>
          </div>
          <div>
            <dt>Rating</dt>
            <dd>{media.average_rating ?? "—"}</dd>
          </div>
        </dl>{/if}
    </header>
    {#if media?.by_kind.length}<MetricPanel
        metricId="media.event_count"
        label="Events and completions"
        unit="count"
        method={report.sections.media.schema}
        rowHeader="Kind"
        rows={mediaRows}
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
      </MetricPanel>{:else}<p class="empty">No canonical media events.</p>{/if}
  </article>

  <article class="domain expenses">
    <header>
      <div>
        <p>Expenses</p>
        <h3>Spend by category</h3>
      </div>
      {#if expenses}<strong>{expenses.expense_count} canonical records</strong
        >{/if}
    </header>
    {#if categoryTotals.length}<MetricPanel
        metricId="expenses.amount_minor"
        label="Spend by category"
        unit={`${primaryCurrency} minor`}
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
      </MetricPanel>{:else}<p class="empty">
        No canonical expenses in {primaryCurrency}.
      </p>{/if}
  </article>
</section>

<style>
  .domain-charts {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
  }
  .domain {
    min-width: 0;
    border: 1px solid var(--border);
    padding: 1rem;
    background: var(--surface);
  }
  .coverage {
    grid-column: 1 / -1;
    display: flex;
    flex-wrap: wrap;
    gap: 0.45rem;
    align-items: center;
  }
  .coverage span,
  .coverage b {
    font-size: 0.66rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .coverage span {
    margin-right: 0.25rem;
    color: var(--text-muted);
  }
  .coverage b {
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 0.35rem 0.55rem;
    color: var(--text-muted);
  }
  .coverage b.available {
    border-color: color-mix(in srgb, var(--accent) 55%, var(--border));
    color: var(--accent);
  }
  .expenses {
    grid-column: 1 / -1;
  }
  header,
  dl,
  li {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  header {
    align-items: start;
    margin-bottom: 0.6rem;
  }
  h3,
  p,
  dl,
  dd,
  ul {
    margin: 0;
  }
  header p {
    color: var(--accent);
    font-size: 0.66rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  h3 {
    margin-top: 0.2rem;
    font-size: 1.05rem;
    letter-spacing: -0.03em;
  }
  dl {
    flex-wrap: wrap;
  }
  dl div {
    display: grid;
    text-align: right;
  }
  dt,
  .empty,
  li {
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  dd,
  header > strong {
    font-size: 0.78rem;
    font-weight: 700;
  }
  ul {
    display: grid;
    gap: 0.35rem;
    padding: 0;
    list-style: none;
  }
  li {
    border-top: 1px solid var(--border);
    padding-top: 0.35rem;
  }
  li b {
    color: var(--text);
  }
  @media (max-width: 760px) {
    .domain-charts {
      grid-template-columns: 1fr;
    }
    .expenses {
      grid-column: auto;
    }
    header {
      display: grid;
    }
    dl {
      justify-content: start;
    }
    dl div {
      text-align: left;
    }
  }
</style>
