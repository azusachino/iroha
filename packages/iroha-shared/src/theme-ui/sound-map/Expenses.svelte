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

<section class="sound-expenses" aria-labelledby="sound-expenses-title">
  <header>
    <p class="kicker">Signal map / {month}</p>
    <h2 id="sound-expenses-title">Spending has a pulse.</h2>
    <p>Bursts are visible; quiet days remain missing, not fabricated.</p>
  </header>
  <article class="signal-panel">
    <MetricPanel {...dailyPanel} label="Daily signal" period={month}>
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
  <MetricPanel {...categoryPanel} label="Category bands" period={month}>
    <div class="signal-bands">
      {#each categoryTotals as item, index}<div
          class="band"
          style={`--band:${index % 6}`}
        >
          <span>{item.category}</span><strong
            >{formatMoney(
              item.amount,
              primaryCurrency,
              primaryExponent,
            )}</strong
          ><i
            style={`width:${Math.max(8, Math.min(100, (item.amount / (categoryTotals[0]?.amount || 1)) * 100))}%`}
          ></i>
        </div>{/each}
    </div>
  </MetricPanel>
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
  .sound-expenses {
    display: grid;
    gap: 1.1rem;
  }
  h2,
  p {
    margin: 0;
  }
  h2 {
    font-size: clamp(2.4rem, 8vw, 6rem);
    letter-spacing: -0.12em;
    line-height: 0.84;
    text-shadow: 0 0 1.5rem color-mix(in srgb, var(--accent) 25%, transparent);
  }
  header p:last-child {
    margin-top: 0.8rem;
    color: var(--text-muted);
  }
  .kicker {
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.66rem;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  .signal-panel {
    border: 1px solid var(--border);
    border-radius: 1rem;
    padding: 0.8rem;
    background:
      radial-gradient(
        circle at 50% 0,
        color-mix(in srgb, var(--accent) 18%, transparent),
        transparent 58%
      ),
      var(--surface);
    box-shadow: 0 0 2rem color-mix(in srgb, var(--accent) 10%, transparent);
  }
  .signal-bands {
    display: grid;
    gap: 0.45rem;
  }
  .band {
    display: grid;
    grid-template-columns: 8rem 6rem 1fr;
    gap: 0.8rem;
    align-items: center;
    font-family: var(--font-mono);
    font-size: 0.75rem;
  }
  .band strong {
    text-align: right;
  }
  .band i {
    display: block;
    height: 0.55rem;
    border-radius: 1rem;
    background: var(--accent);
    opacity: calc(1 - var(--band) * 0.1);
  }
  @media (max-width: 640px) {
    .band {
      grid-template-columns: 6rem 5rem 1fr;
    }
  }
</style>
