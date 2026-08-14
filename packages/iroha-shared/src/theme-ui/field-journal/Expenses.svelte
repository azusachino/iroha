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

<section class="journal-expenses" aria-labelledby="journal-expenses-title">
  <header class="journal-head">
    <p class="date">{month}</p>
    <div>
      <p class="kicker">Field note · spending</p>
      <h2 id="journal-expenses-title">What the month asked for.</h2>
    </div>
    <p class="note">
      An observed sequence of dated entries. No score, no invented explanation.
    </p>
  </header>
  <article class="journal-chart">
    <div class="journal-caption">
      <span>Daily entries</span><strong>{primaryCurrency}</strong>
    </div>
    <MetricPanel {...dailyPanel} label="Daily entries" period={month}>
      <BarChart
        categories={dailyTotals.map(([day]) => formatExpenseDay(day))}
        primary={{
          name: primaryCurrency,
          values: dailyTotals.map(([, amount]) => amount),
          color: "var(--accent-2)",
          formatter: (value) =>
            formatMoney(value, primaryCurrency, primaryExponent),
        }}
        height={260}
      />
    </MetricPanel>
  </article>
  <div class="journal-category">
    <h3>Recurring categories</h3>
    <MetricPanel {...categoryPanel} label="Recurring categories" period={month}>
      <div class="category-list">
        {#each categoryTotals as item}<div>
            <span>{item.category}</span><b
              >{formatMoney(item.amount, primaryCurrency, primaryExponent)}</b
            >
          </div>{/each}
      </div>
    </MetricPanel>
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
  .journal-expenses {
    display: grid;
    gap: 1.25rem;
    font-family: Georgia, "Times New Roman", serif;
    min-width: 0;
  }
  .journal-expenses > * {
    min-width: 0;
  }
  h2,
  h3,
  p {
    margin: 0;
  }
  h2 {
    max-width: 34rem;
    font-size: clamp(2.1rem, 6vw, 4.5rem);
    font-weight: 500;
    letter-spacing: -0.07em;
    line-height: 0.92;
  }
  h3 {
    font-size: 1.1rem;
    font-weight: 500;
  }
  .journal-head {
    display: grid;
    grid-template-columns: 7rem 1fr 13rem;
    gap: 1rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.25rem;
  }
  .date,
  .kicker {
    color: var(--accent);
    font-size: 0.72rem;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .note {
    color: var(--text-muted);
    font-size: 0.85rem;
    line-height: 1.5;
  }
  .journal-chart,
  .journal-category {
    border: 1px solid var(--border);
    padding: 1rem;
    background: color-mix(in srgb, var(--surface) 88%, var(--accent) 12%);
  }
  .journal-caption {
    display: flex;
    justify-content: space-between;
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  .journal-caption strong {
    color: var(--text);
  }
  .category-list {
    display: grid;
    gap: 0.6rem;
    margin-top: 1rem;
  }
  .category-list div {
    display: flex;
    justify-content: space-between;
    border-bottom: 1px dashed var(--border);
    padding-bottom: 0.45rem;
  }
  @media (max-width: 768px) {
    .journal-head {
      grid-template-columns: 1fr;
    }
  }
</style>
