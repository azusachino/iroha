<script lang="ts">
  import BarChart from "$lib/components/BarChart.svelte";
  import ReportDetails from "$lib/components/ReportDetails.svelte";
  import StatTile from "$lib/components/StatTile.svelte";
  import type { ReportThemeProps } from "$lib/report-view";
  let {
    month,
    report,
    trend,
    primaryCurrency,
    primaryExponent,
    categoryTotals,
    currentTotal,
    expenseRecordCount,
    formatMoney,
    formatDuration,
  }: ReportThemeProps = $props();
</script>

<section class="atlas-reports">
  <header>
    <p class="kicker">Monthly survey · {month}</p>
    <h2>Read the month as a territory.</h2>
    <p>
      Movement, rest, library, and ledger plotted together; source details
      remain below.
    </p>
  </header>
  <div class="stats">
    <StatTile
      label={`Spend · ${primaryCurrency}`}
      value={formatMoney(currentTotal, primaryCurrency, primaryExponent)}
      sub="selected month"
    /><StatTile
      label="Records"
      value={String(expenseRecordCount)}
      sub="canonical expenses"
    />
  </div>
  <div class="charts">
    <article>
      <p class="kicker">Spend by category</p>
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
        height={250}
      />
    </article>
    <article>
      <p class="kicker">Recent contour</p>
      <BarChart
        categories={trend.map((item) => item.label)}
        primary={{
          name: primaryCurrency,
          values: trend.map((item) => item.amount),
          color: "var(--accent-2)",
          formatter: (value) =>
            formatMoney(value, primaryCurrency, primaryExponent),
        }}
        primaryType="line"
        height={250}
      />
    </article>
  </div>
  <ReportDetails {report} {formatMoney} {formatDuration} />
</section>

<style>
  .atlas-reports {
    display: grid;
    gap: 1.25rem;
  }
  .atlas-reports h2,
  .atlas-reports p {
    margin: 0;
  }
  .atlas-reports h2 {
    font-size: clamp(2.2rem, 6vw, 5rem);
    letter-spacing: -0.1em;
    line-height: 0.88;
  }
  .atlas-reports header > p:last-child {
    max-width: 35rem;
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
  .stats {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 1px;
    background: var(--text);
  }
  .stats :global(.stat-tile) {
    border: 0;
    background: var(--surface);
  }
  .charts {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
  }
  .charts article {
    min-width: 0;
    border: 1px solid var(--border);
    padding: 1rem;
    background: var(--surface);
  }
  @media (max-width: 760px) {
    .charts {
      grid-template-columns: 1fr;
    }
  }
</style>
