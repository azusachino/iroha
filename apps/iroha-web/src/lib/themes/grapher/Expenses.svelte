<script lang="ts">
  import BarChart from "$lib/components/BarChart.svelte";
  import ExpenseLedger from "$lib/components/ExpenseLedger.svelte";
  import MetricPanel from "@iroha/shared/MetricPanel.svelte";
  import type { ExpenseThemeProps } from "$lib/expense-view";
  import { formatExpenseDay } from "$lib/expense-view";
  let {
    month,
    primaryCurrency,
    primaryExponent,
    categoryTotals,
    dailyTotals,
    dailyPanel,
    categoryPanel,
    expenses,
    selected,
    selectedId,
    detailLoading,
    onSelect,
    onRemove,
    formatMoney,
  }: ExpenseThemeProps = $props();
</script>

<section class="grapher-expenses" aria-labelledby="grapher-expenses-title">
  <header>
    <p class="kicker">Expense series · {month}</p>
    <h2 id="grapher-expenses-title">The trend, then the evidence.</h2>
    <p>Compare the daily curve first. The canonical ledger follows below.</p>
  </header>
  <article class="chart-panel">
    <MetricPanel {...dailyPanel} label="Daily spend" period={month}>
      <BarChart
        categories={dailyTotals.map(([day]) => formatExpenseDay(day))}
        primary={{
          name: primaryCurrency,
          values: dailyTotals.map(([, amount]) => amount),
          color: "var(--accent)",
          formatter: (value) =>
            formatMoney(value, primaryCurrency, primaryExponent),
        }}
        primaryType="line"
        height={300}
      />
    </MetricPanel>
  </article>
  <article class="chart-panel">
    <div class="panel-title">
      <span>Composition</span><strong>{primaryCurrency}</strong>
    </div>
    <MetricPanel {...categoryPanel} label="Spend by category" period={month}>
      <BarChart
        categories={categoryTotals.map((item) => item.category)}
        primary={{
          name: primaryCurrency,
          values: categoryTotals.map((item) => item.amount),
          color: "var(--accent-2)",
          formatter: (value) =>
            formatMoney(value, primaryCurrency, primaryExponent),
        }}
        orientation="horizontal"
        categorical
        height={250}
      />
    </MetricPanel>
  </article>
  <ExpenseLedger
    {expenses}
    {selected}
    {selectedId}
    {detailLoading}
    {onSelect}
    {onRemove}
    {formatMoney}
  />
</section>

<style>
  .grapher-expenses {
    display: grid;
    gap: 1rem;
    font-family: "IBM Plex Mono", "SFMono-Regular", monospace;
  }
  h2,
  p {
    margin: 0;
  }
  h2 {
    max-width: 38rem;
    font-size: clamp(2rem, 6vw, 5rem);
    letter-spacing: -0.1em;
    line-height: 0.88;
  }
  header p:last-child {
    margin-top: 0.8rem;
    color: var(--text-muted);
    font-family: var(--font-sans);
  }
  .kicker {
    color: var(--accent);
    font-size: 0.68rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .chart-panel {
    min-width: 0;
    border-top: 3px solid var(--text);
    padding: 1rem 0;
  }
  .panel-title {
    display: flex;
    justify-content: space-between;
    margin-bottom: 0.5rem;
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .panel-title strong {
    color: var(--text);
  }
</style>
