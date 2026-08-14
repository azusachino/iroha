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

<section class="phenology-expenses" aria-labelledby="phenology-expenses-title">
  <header>
    <p class="kicker">Seasonal ledger · {month}</p>
    <h2 id="phenology-expenses-title">The rhythm of recurring needs.</h2>
    <p>
      Read the month as a cycle: intensity over days, repeated categories
      beneath it.
    </p>
  </header>
  <div class="rhythm">
    <div class="rhythm-axis">
      {#each dailyTotals as [day]}<span title={day}></span>{/each}
    </div>
    <MetricPanel {...dailyPanel} label="Daily rhythm" period={month}>
      <BarChart
        categories={dailyTotals.map(([day]) => formatExpenseDay(day))}
        primary={{
          name: primaryCurrency,
          values: dailyTotals.map(([, amount]) => amount),
          color: "var(--accent)",
          formatter: (value) =>
            formatMoney(value, primaryCurrency, primaryExponent),
        }}
        height={250}
      />
    </MetricPanel>
  </div>
  <section class="category-cycle">
    <h3>Category cycle</h3>
    <MetricPanel {...categoryPanel} label="Category cycle" period={month}>
      <div class="cycle-list">
        {#each categoryTotals as item}<span
            style={`--weight:${Math.max(0.25, Math.min(1, item.amount / (categoryTotals[0]?.amount || 1)))}`}
            >{item.category}<b
              >{formatMoney(item.amount, primaryCurrency, primaryExponent)}</b
            ></span
          >{/each}
      </div>
    </MetricPanel>
  </section>
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
  .phenology-expenses {
    display: grid;
    gap: 1.25rem;
  }
  h2,
  h3,
  p {
    margin: 0;
  }
  h2 {
    max-width: 42rem;
    color: var(--text);
    font-size: clamp(2.2rem, 6vw, 5rem);
    letter-spacing: -0.1em;
    line-height: 0.9;
  }
  header p:last-child {
    max-width: 38rem;
    margin-top: 0.8rem;
    color: var(--text-muted);
    line-height: 1.5;
  }
  .kicker {
    color: var(--accent);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  .rhythm,
  .category-cycle {
    border: 1px solid var(--border);
    padding: 1rem;
    background:
      radial-gradient(
        circle at 15% 20%,
        color-mix(in srgb, var(--accent) 16%, transparent),
        transparent 45%
      ),
      var(--surface);
  }
  .rhythm-axis {
    display: flex;
    justify-content: space-between;
    height: 0.35rem;
    margin: 0 2rem 0.3rem;
  }
  .rhythm-axis span {
    width: 0.35rem;
    height: 0.35rem;
    border-radius: 50%;
    background: var(--accent);
    opacity: 0.55;
  }
  h3 {
    color: var(--accent);
    font-size: 0.85rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .cycle-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0.55rem;
    margin-top: 1rem;
  }
  .cycle-list span {
    display: grid;
    gap: 0.25rem;
    min-width: 6rem;
    border: 1px solid
      color-mix(
        in srgb,
        var(--accent) calc(var(--weight) * 100%),
        var(--border)
      );
    padding: 0.65rem;
    color: var(--text);
    font-size: 0.78rem;
  }
  .cycle-list b {
    color: var(--text-muted);
    font-size: 0.7rem;
    font-weight: 500;
  }
</style>
