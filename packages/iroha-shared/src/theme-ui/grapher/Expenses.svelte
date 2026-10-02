<script lang="ts">
  import BarChart from "../components/BarChart.svelte";
  import ExpenseLedger from "../components/ExpenseLedger.svelte";
  import MetricPanel from "../../components/MetricPanel.svelte";
  import PanelFrame from "../../components/PanelFrame.svelte";
  import StatTile from "../../components/StatTile.svelte";
  import type { ExpenseThemeProps } from "../../view-contracts/expense-view";
  import { formatExpenseDay } from "../../view-contracts/expense-view";
  let {
    month, primaryCurrency, primaryExponent, currencyTotals, categoryTotals, dailyTotals,
    dailyPanel, categoryPanel, expenses, selected, selectedId, detailLoading,
    onSelect, onRemove, formatMoney,
  }: ExpenseThemeProps = $props();
</script>

<section class="grapher-expenses" aria-label="Expense evidence">
  <div class="summary-grid">
    {#each currencyTotals as total (total.currency)}
      <StatTile label={`Spending · ${total.currency}`} value={formatMoney(total.amountMinor, total.currency, total.exponent)} context={`Observed period: ${month}`} sub={`${total.count} ${total.count === 1 ? "record" : "records"}`} />
    {/each}
  </div>
  <MetricPanel {...dailyPanel} label="Daily spend" period={month} framed>
    {#snippet heading()}<h2>Daily spend · {primaryCurrency}</h2>{/snippet}
    <BarChart
      categories={dailyTotals.map(([day]) => formatExpenseDay(day))}
      primary={{
        name: primaryCurrency,
        values: dailyTotals.map(([, amount]) => amount),
        color: "var(--accent)",
        axis: { unit: primaryCurrency, scale: 10 ** -primaryExponent, minimumInterval: 1 },
        formatter: (value) => formatMoney(value, primaryCurrency, primaryExponent),
      }}
      primaryType="line"
      height={300}
    />
  </MetricPanel>
  <MetricPanel {...categoryPanel} label="Spend by category" period={month} framed>
    {#snippet heading()}<h2>Spend by category · {primaryCurrency}</h2>{/snippet}
    <BarChart
      categories={categoryTotals.map((item) => item.category)}
      primary={{
        name: primaryCurrency,
        values: categoryTotals.map((item) => item.amount),
        color: "var(--accent-2)",
        axis: { unit: primaryCurrency, scale: 10 ** -primaryExponent, minimumInterval: 1 },
        formatter: (value) => formatMoney(value, primaryCurrency, primaryExponent),
      }}
      orientation="horizontal"
      categorical
      height={250}
    />
  </MetricPanel>
  <PanelFrame label="Canonical ledger">
    <ExpenseLedger
      {expenses} {selected} {selectedId} {detailLoading} {onSelect} {onRemove} {formatMoney}
      embedded period={month}
    />
  </PanelFrame>
</section>

<style>
  .grapher-expenses { display: grid; gap: var(--space-4); min-width: 0; font-family: var(--font-mono); }
  h2 { margin: 0; font-size: var(--type-title-small); }
  .summary-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 12rem), 1fr)); gap: var(--space-3); }
</style>
