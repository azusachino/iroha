<script lang="ts">
  import type { MonthlyReport } from "$lib/api";
  import BarChart from "$lib/components/BarChart.svelte";
  import { formatDuration, formatMonth } from "$lib/format";
  import { useTheme } from "$lib/themes/context.svelte";

  let {
    current,
    previous = null,
    formatMoney,
  }: {
    current: MonthlyReport;
    previous?: MonthlyReport | null;
    formatMoney: (
      amountMinor: number,
      currency: string,
      exponent: number,
    ) => string;
  } = $props();

  const theme = useTheme();
  const periods = $derived([
    previous?.period.month ?? "Previous",
    current.period.month,
  ]);

  function sectionData<T>(
    report: MonthlyReport | null | undefined,
    section: keyof MonthlyReport["sections"],
  ): T | null {
    const value = report?.sections[section];
    return value?.state === "available" ? (value.data as T | null) : null;
  }

  const movement = $derived([
    sectionData<MonthlyReport["sections"]["movement"]["data"]>(
      previous,
      "movement",
    )?.distance_m ?? null,
    sectionData<MonthlyReport["sections"]["movement"]["data"]>(
      current,
      "movement",
    )?.distance_m ?? null,
  ]);
  const sleep = $derived([
    sectionData<MonthlyReport["sections"]["sleep"]["data"]>(previous, "sleep")
      ?.average_asleep_s ?? null,
    sectionData<MonthlyReport["sections"]["sleep"]["data"]>(current, "sleep")
      ?.average_asleep_s ?? null,
  ]);
  const health = $derived([
    sectionData<MonthlyReport["sections"]["daily_health"]["data"]>(
      previous,
      "daily_health",
    )?.observed_days ?? null,
    sectionData<MonthlyReport["sections"]["daily_health"]["data"]>(
      current,
      "daily_health",
    )?.observed_days ?? null,
  ]);
  const media = $derived([
    sectionData<MonthlyReport["sections"]["media"]["data"]>(previous, "media")
      ?.event_count ?? null,
    sectionData<MonthlyReport["sections"]["media"]["data"]>(current, "media")
      ?.event_count ?? null,
  ]);
  const mediaCompleted = $derived([
    sectionData<MonthlyReport["sections"]["media"]["data"]>(previous, "media")
      ?.completed_count ?? null,
    sectionData<MonthlyReport["sections"]["media"]["data"]>(current, "media")
      ?.completed_count ?? null,
  ]);

  const expenseCurrencies = $derived.by(() => {
    const currencies = new Set<string>();
    for (const report of [previous, current]) {
      for (const item of sectionData<
        MonthlyReport["sections"]["expenses"]["data"]
      >(report, "expenses")?.totals_by_currency ?? []) {
        currencies.add(item.currency);
      }
    }
    return [...currencies].sort();
  });

  function expenseValues(currency: string): (number | null)[] {
    return [previous, current].map(
      (report) =>
        sectionData<MonthlyReport["sections"]["expenses"]["data"]>(
          report,
          "expenses",
        )?.totals_by_currency.find((item) => item.currency === currency)
          ?.amount_minor ?? null,
    );
  }

  function expenseExponent(currency: string): number {
    return (
      [current, previous]
        .map(
          (report) =>
            sectionData<MonthlyReport["sections"]["expenses"]["data"]>(
              report,
              "expenses",
            )?.totals_by_currency.find((item) => item.currency === currency)
              ?.currency_exponent,
        )
        .find((value) => value != null) ?? (currency === "JPY" ? 0 : 2)
    );
  }

  function delta(values: (number | null)[]): string {
    const before = values[0];
    const after = values[1];
    if (before == null || after == null) return "No complete comparison";
    const change = after - before;
    if (change === 0) return "No change";
    return `${change > 0 ? "+" : "−"}${Math.abs(change).toLocaleString(undefined, { maximumFractionDigits: 1 })}`;
  }

  const monthLabels = $derived(periods.map((period) => formatMonth(period)));
</script>

<section
  class="report-comparison"
  data-theme={theme.definition().identity.id}
  aria-labelledby="report-comparison-title"
>
  <header class="comparison-header">
    <div>
      <p class="eyebrow">Month-on-month comparison</p>
      <h2 id="report-comparison-title">What changed?</h2>
      <p class="description">
        Like-for-like indicators are shown separately by unit. Missing prior
        observations remain gaps, not zeros.
      </p>
    </div>
    <span class="period-pair">{monthLabels.join(" → ")}</span>
  </header>

  {#if !previous}
    <p class="empty">The previous month is not available for comparison.</p>
  {:else}
    <div class="comparison-grid">
      <article>
        <header><span>Movement</span><strong>Distance</strong></header>
        <BarChart
          categories={monthLabels}
          primary={{
            name: "Distance",
            values: movement.map((value) =>
              value == null ? null : value / 1000,
            ),
            formatter: (value) => `${value.toFixed(1)} km`,
          }}
          primaryType="line"
          height={180}
        />
        <p class="delta">
          {delta(
            movement.map((value) => (value == null ? null : value / 1000)),
          )} km
        </p>
      </article>
      <article>
        <header><span>Sleep</span><strong>Average asleep</strong></header>
        <BarChart
          categories={monthLabels}
          primary={{
            name: "Average asleep",
            values: sleep,
            formatter: (value) => formatDuration(value),
          }}
          primaryType="line"
          height={180}
        />
        <p class="delta">
          {delta(sleep.map((value) => (value == null ? null : value / 3600)))} h
        </p>
      </article>
      <article>
        <header><span>Daily health</span><strong>Observed days</strong></header>
        <BarChart
          categories={monthLabels}
          primary={{ name: "Observed days", values: health }}
          primaryType="line"
          height={180}
        />
        <p class="delta">{delta(health)} days</p>
      </article>
      <article>
        <header>
          <span>Media</span><strong>Events and completions</strong>
        </header>
        <BarChart
          categories={monthLabels}
          primary={{ name: "Events", values: media }}
          secondary={{
            name: "Completed",
            values: mediaCompleted,
            color: "var(--accent-2)",
          }}
          primaryType="line"
          height={180}
        />
        <p class="delta">Events {delta(media)}</p>
      </article>
      {#each expenseCurrencies as currency (currency)}
        {@const values = expenseValues(currency)}
        <article>
          <header>
            <span>Expenses · {currency}</span><strong>Total spend</strong>
          </header>
          <BarChart
            categories={monthLabels}
            primary={{
              name: currency,
              values,
              formatter: (value) =>
                formatMoney(value, currency, expenseExponent(currency)),
            }}
            primaryType="line"
            height={180}
          />
          <p class="delta">
            {delta(
              values.map((value) =>
                value == null ? null : value / 10 ** expenseExponent(currency),
              ),
            )}
            {currency}
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
    font-size: clamp(1.35rem, 3vw, 2.2rem);
  }
  .description,
  .period-pair,
  .empty,
  .delta {
    color: var(--text-muted);
    font-size: 0.78rem;
    line-height: 1.5;
  }
  .period-pair {
    white-space: nowrap;
  }
  .comparison-grid {
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
    margin: 0;
    color: var(--accent);
    font-variant-numeric: tabular-nums;
  }
  @media (max-width: 720px) {
    .comparison-header {
      display: grid;
    }
    .comparison-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
