<script lang="ts">
  import BarChart from "../components/BarChart.svelte";
  import ExpenseLedger from "../components/ExpenseLedger.svelte";
  import MetricPanel from "../../MetricPanel.svelte";
  import type { ExpenseThemeProps } from "../../expense-view";
  import { formatExpenseDay } from "../../expense-view";
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

<section class="archive-expenses" aria-labelledby="archive-expenses-title">
  <header class="archive-heading">
    <div>
      <p class="kicker">Archive / canonical expenses</p>
      <h2 id="archive-expenses-title">The ledger is the source.</h2>
    </div>
    <span>{month}</span>
  </header>
  <ExpenseLedger
    {expenses}
    {selected}
    {selectedId}
    {detailLoading}
    {onSelect}
    {onRemove}
    {formatMoney}
  />
  <section class="archive-analysis">
    <div class="analysis-title">
      <h3>Derived views</h3>
      <p>Computed from canonical records; values retain their currency.</p>
    </div>
    <div class="archive-charts">
      <MetricPanel {...categoryPanel} label="Spend by category" period={month}>
        <BarChart
          categories={categoryTotals.map((item) => item.category)}
          primary={{
            name: primaryCurrency,
            values: categoryTotals.map((item) => item.amount),
            color: "var(--accent)",
            formatter: (value) =>
              formatMoney(value, primaryCurrency, primaryExponent),
          }}
          orientation="horizontal"
          categorical
          height={250}
        />
      </MetricPanel><MetricPanel
        {...dailyPanel}
        label="Daily spend"
        period={month}
      >
        <BarChart
          categories={dailyTotals.map(([day]) => formatExpenseDay(day))}
          primary={{
            name: primaryCurrency,
            values: dailyTotals.map(([, amount]) => amount),
            color: "var(--accent-2)",
            formatter: (value) =>
              formatMoney(value, primaryCurrency, primaryExponent),
          }}
          height={250}
        />
      </MetricPanel>
    </div>
  </section>
</section>

<style>
  .archive-expenses {
    display: grid;
    gap: 1.25rem;
    font-family: var(--font-mono);
    min-width: 0;
  }
  .archive-expenses > * {
    min-width: 0;
  }
  h2,
  h3,
  p {
    margin: 0;
  }
  h2 {
    font-family: var(--font-sans);
    font-size: clamp(2rem, 5vw, 4.2rem);
    letter-spacing: -0.1em;
    line-height: 0.9;
  }
  .archive-heading {
    display: flex;
    justify-content: space-between;
    align-items: end;
    border-bottom: 1px solid var(--text);
    padding-bottom: 0.8rem;
  }
  .archive-heading > span {
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .kicker {
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  .archive-analysis {
    border-top: 3px double var(--text);
    padding-top: 1rem;
  }
  .analysis-title {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: baseline;
  }
  h3 {
    font-size: 0.85rem;
    text-transform: uppercase;
  }
  .analysis-title p {
    color: var(--text-muted);
    font-family: var(--font-sans);
    font-size: 0.78rem;
  }
  .archive-charts {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
    margin-top: 0.8rem;
  }
  @media (max-width: 768px) {
    .archive-charts {
      grid-template-columns: 1fr;
    }
  }
</style>
