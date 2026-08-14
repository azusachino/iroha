<script lang="ts">
  import BarChart from "@iroha/shared/theme-ui/components/BarChart.svelte";
  import ExpenseLedger from "$lib/components/ExpenseLedger.svelte";
  import StatTile from "$lib/components/StatTile.svelte";
  import MetricPanel from "@iroha/shared/MetricPanel.svelte";
  import type { ExpenseThemeProps } from "$lib/expense-view";
  import { formatExpenseDay } from "$lib/expense-view";

  let {
    month,
    primaryCurrency,
    primaryExponent,
    currencyTotals,
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

<section class="atlas-expenses" aria-labelledby="atlas-expenses-title">
  <header class="atlas-heading">
    <div>
      <p class="kicker">Ledger atlas · {month}</p>
      <h2 id="atlas-expenses-title">Where the money went.</h2>
    </div>
    <p class="orientation">
      A plotted survey of places, categories, and exact records.
    </p>
  </header>
  <div class="atlas-stats">
    {#each currencyTotals.slice(0, 3) as total (total.currency)}
      <StatTile
        label={`Spent · ${total.currency}`}
        value={formatMoney(total.amountMinor, total.currency, total.exponent)}
        sub={`${total.count} records`}
      />
    {/each}
  </div>
  <div class="atlas-charts">
    <article class="atlas-plate">
      <p class="kicker">Category coordinates · {primaryCurrency}</p>
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
          height={270}
        />
      </MetricPanel>
    </article>
    <article class="atlas-plate">
      <p class="kicker">Daily route</p>
      <MetricPanel {...dailyPanel} label="Daily spend" period={month}>
        <BarChart
          categories={dailyTotals.map(([day]) => formatExpenseDay(day))}
          primary={{
            name: primaryCurrency,
            values: dailyTotals.map(([, amount]) => amount),
            color: "var(--accent-2)",
            formatter: (value) =>
              formatMoney(value, primaryCurrency, primaryExponent),
          }}
          height={270}
        />
      </MetricPanel>
    </article>
  </div>
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
  .atlas-expenses {
    display: grid;
    gap: 1.25rem;
    font-family: var(--font-sans);
  }
  .atlas-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: end;
    border-bottom: 2px solid var(--text);
    padding-bottom: 1rem;
  }
  h2,
  p {
    margin: 0;
  }
  h2 {
    font-size: clamp(1.8rem, 5vw, 3.8rem);
    letter-spacing: -0.08em;
    line-height: 0.92;
  }
  .kicker {
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.66rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .orientation {
    max-width: 15rem;
    color: var(--text-muted);
    font-size: 0.78rem;
    line-height: 1.5;
    text-align: right;
  }
  .atlas-stats {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1px;
    background: var(--text);
  }
  .atlas-stats :global(.stat-tile) {
    border: 0;
    background: var(--surface);
  }
  .atlas-charts {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
  }
  .atlas-plate {
    min-width: 0;
    border: 1px solid var(--border);
    padding: 1rem;
    background: var(--surface);
  }
  @media (max-width: 760px) {
    .atlas-heading,
    .atlas-charts {
      grid-template-columns: 1fr;
      display: grid;
    }
    .orientation {
      text-align: left;
    }
    .atlas-stats {
      grid-template-columns: 1fr;
    }
  }
</style>
