<script lang="ts">
  import type { MonthlyReportSeries } from "../../domain/report";
  import type { DesignLanguage } from "../../theme/themes";
  import BarChart from "./BarChart.svelte";
  import { formatCanonicalMonth } from "../../format/format";

  let {
    series,
    formatMoney,
    theme,
  }: {
    series: MonthlyReportSeries | null;
    formatMoney: (
      amountMinor: number,
      currency: string,
      exponent: number,
    ) => string;
    theme: DesignLanguage;
  } = $props();
  const points = $derived(series?.reports ?? []);
  const categories = $derived(
    points.map((point) => formatCanonicalMonth(point.month)),
  );
  const partialMonths = $derived(
    points
      .filter((point) => point.completeness === "partial")
      .map((point) => formatCanonicalMonth(point.month)),
  );

  const movementValues = $derived(
    points.map((point) =>
      point.movement == null ? null : point.movement.distance_m / 1000,
    ),
  );
  const sleepValues = $derived(
    points.map((point) =>
      point.sleep == null ? null : point.sleep.average_asleep_s / 3600,
    ),
  );
  const healthValues = $derived(
    points.map((point) => point.daily_health?.observed_days ?? null),
  );
  const mediaValues = $derived(
    points.map((point) => point.media?.event_count ?? null),
  );
  const mediaCompletedValues = $derived(
    points.map((point) => point.media?.completed_count ?? null),
  );

  const expenseCurrencies = $derived.by(() => {
    const currencies = new Set<string>();
    for (const point of points) {
      for (const item of point.expenses?.totals_by_currency ?? []) {
        currencies.add(item.currency);
      }
    }
    return [...currencies].sort();
  });

  function expenseValues(currency: string): (number | null)[] {
    return points.map(
      (point) =>
        point.expenses?.totals_by_currency.find(
          (item) => item.currency === currency,
        )?.amount_minor ?? null,
    );
  }

  function expenseExponent(currency: string): number {
    return (
      points
        .map(
          (point) =>
            point.expenses?.totals_by_currency.find(
              (item) => item.currency === currency,
            )?.currency_exponent,
        )
        .find((value) => value != null) ?? (currency === "JPY" ? 0 : 2)
    );
  }

  function number(value: number): string {
    return new Intl.NumberFormat(undefined, {
      maximumFractionDigits: 1,
    }).format(value);
  }

  function trendDelta(
    values: (number | null)[],
    formatter: (value: number) => string,
  ): string {
    const observed = values.filter(
      (value): value is number => value != null && Number.isFinite(value),
    );
    if (observed.length < 2) return "Need two observed months";
    const change = observed.at(-1)! - observed.at(-2)!;
    if (change === 0) return "Stable vs prior observed month";
    return (
      (change > 0 ? "+" : "−") +
      formatter(Math.abs(change)) +
      " vs prior observed month"
    );
  }
</script>

<section
  class="report-comparison"
  data-theme={theme}
  aria-labelledby="report-comparison-title"
>
  <header class="comparison-header">
    <div>
      <p class="eyebrow">Canonical comparison · twelve months</p>
      <h2 id="report-comparison-title">Twelve-month trends</h2>
      <p class="description">
        Monthly points are aggregated by the server. Months with no canonical
        records stay out of the plot; partial months remain marked as partial.
      </p>
    </div>
    {#if series}<span class="period-range"
        >{formatCanonicalMonth(series.from_month)} →
        {formatCanonicalMonth(series.to_month)}</span
      >{/if}
  </header>

  {#if partialMonths.length}
    <p class="coverage-note">
      Partial observation: {partialMonths.join(", ")}. No annualization is
      applied.
    </p>
  {/if}

  {#if !series || points.length === 0}
    <p class="empty">
      No canonical records are available in this trend window.
    </p>
  {:else}
    <div class="trend-grid">
      <article>
        <header><span>Movement</span><strong>Distance</strong></header>
        <BarChart
          {categories}
          primary={{
            name: "Distance",
            values: movementValues,
            formatter: (value) => number(value) + " km",
          }}
          primaryType="line"
          height={210}
        />
        <p class="delta">
          {trendDelta(movementValues, (value) => number(value) + " km")}
        </p>
      </article>
      <article>
        <header><span>Sleep</span><strong>Average asleep</strong></header>
        <BarChart
          {categories}
          primary={{
            name: "Average asleep",
            values: sleepValues,
            formatter: (value) => number(value) + " h",
          }}
          primaryType="line"
          height={210}
        />
        <p class="delta">
          {trendDelta(sleepValues, (value) => number(value) + " h")}
        </p>
      </article>
      <article>
        <header><span>Daily health</span><strong>Observed days</strong></header>
        <BarChart
          {categories}
          primary={{ name: "Observed days", values: healthValues }}
          primaryType="line"
          height={210}
        />
        <p class="delta">
          {trendDelta(healthValues, (value) => number(value) + " days")}
        </p>
      </article>
      <article>
        <header>
          <span>Media</span><strong>Events and completions</strong>
        </header>
        <BarChart
          {categories}
          primary={{ name: "Events", values: mediaValues }}
          secondary={{
            name: "Completed",
            values: mediaCompletedValues,
            color: "var(--accent-2)",
          }}
          primaryType="line"
          height={210}
        />
        <p class="delta">
          {trendDelta(mediaValues, (value) => number(value) + " events")}
        </p>
      </article>
      {#each expenseCurrencies as currency (currency)}
        {@const values = expenseValues(currency)}
        {@const exponent = expenseExponent(currency)}
        <article>
          <header>
            <span>Expenses · {currency}</span><strong>Spend</strong>
          </header>
          <BarChart
            {categories}
            primary={{
              name: currency,
              values,
              formatter: (value) => formatMoney(value, currency, exponent),
            }}
            primaryType="line"
            height={210}
          />
          <p class="delta">
            {trendDelta(values, (value) =>
              formatMoney(value, currency, exponent),
            )}
          </p>
        </article>
      {/each}
    </div>
  {/if}
</section>

<style>
  .report-comparison {
    display: grid;
    gap: 1rem;
    padding: clamp(1rem, 2.5vw, 1.5rem);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }

  .report-comparison[data-theme="atlas"] {
    border-width: 2px;
    border-radius: 2px;
  }
  .report-comparison[data-theme="field-journal"] {
    border-style: dashed;
    border-radius: 0;
  }
  .report-comparison[data-theme="phenology"] {
    border-radius: 1.2rem;
  }
  .report-comparison[data-theme="sound-map"] {
    border-inline-width: 3px;
  }
  .report-comparison[data-theme="archive"] {
    border-width: 3px;
    border-radius: 0;
  }
  .report-comparison[data-theme="grapher"] {
    border-radius: 2px;
    border-bottom-width: 3px;
  }

  .comparison-header,
  .comparison-header h2,
  .comparison-header p {
    margin: 0;
  }
  .comparison-header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .eyebrow {
    margin-bottom: 0.35rem !important;
    color: var(--accent);
    font-size: 0.66rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .comparison-header h2 {
    font-size: clamp(1.45rem, 3vw, 2.3rem);
    letter-spacing: -0.04em;
  }
  .description,
  .period-range,
  .coverage-note,
  .empty,
  .delta {
    color: var(--text-muted);
    font-size: 0.78rem;
    line-height: 1.5;
  }
  .description {
    max-width: 44rem;
    margin-top: 0.35rem !important;
  }
  .period-range {
    white-space: nowrap;
  }
  .coverage-note {
    margin: 0;
    padding: 0.55rem 0.7rem;
    border-inline-start: 3px solid var(--accent-2);
    background: color-mix(in srgb, var(--accent-2) 8%, transparent);
  }
  .trend-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
  }
  article {
    min-width: 0;
    padding: 0.8rem;
    border: 1px solid var(--border);
    background: var(--surface-2);
  }
  article header {
    display: flex;
    justify-content: space-between;
    gap: 0.8rem;
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  article header strong {
    color: var(--text);
  }
  .delta {
    margin: 0.35rem 0 0;
  }
  @media (max-width: 768px) {
    .comparison-header {
      display: grid;
    }
    .trend-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
