<script lang="ts">
  import BarChart from "../components/BarChart.svelte";
  import ExpenseLedger from "../components/ExpenseLedger.svelte";
  import MetricPanel from "../../components/MetricPanel.svelte";
  import type { ExpenseThemeProps } from "../../view-contracts/expense-view";
  import { formatExpenseDay } from "../../view-contracts/expense-view";
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
          axis: { unit: primaryCurrency, scale: 10 ** -primaryExponent, minimumInterval: 1 },
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
          axis: { unit: primaryCurrency, scale: 10 ** -primaryExponent, minimumInterval: 1 },
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
    gap: var(--space-4);
    font-family: "IBM Plex Mono", "SFMono-Regular", monospace;
    min-width: 0;
  }
  .grapher-expenses > * {
    min-width: 0;
  }
  h2,
  p {
    margin: 0;
  }
  h2 {
    max-width: 38rem;
    font-size: var(--grapher-utility-title-size);
    letter-spacing: -0.05em;
    line-height: 1;
  }
  header p:last-child {
    margin-top: var(--space-3);
    color: var(--text-muted);
    font-family: var(--font-sans);
  }
  .kicker {
    color: var(--accent);
    font-size: var(--type-caption);
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .chart-panel {
    min-width: 0;
    border-top: 3px solid var(--text);
    padding: var(--space-4) 0;
  }
  .panel-title {
    display: flex;
    justify-content: space-between;
    margin-bottom: var(--space-2);
    color: var(--text-muted);
    font-size: var(--type-caption);
  }
  .panel-title strong {
    color: var(--text);
  }
</style>
