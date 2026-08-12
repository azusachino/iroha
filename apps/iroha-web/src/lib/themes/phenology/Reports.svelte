<script lang="ts">
  import BarChart from "$lib/components/BarChart.svelte";
  import ReportDetails from "$lib/components/ReportDetails.svelte";
  import type { ReportThemeProps } from "$lib/report-view";
  let {
    month,
    report,
    trend,
    primaryCurrency,
    primaryExponent,
    categoryTotals,
    formatMoney,
    formatDuration,
  }: ReportThemeProps = $props();
</script>

<section class="phenology-reports">
  <header>
    <p class="kicker">Monthly cycle · {month}</p>
    <h2>Observe the season of the data.</h2>
    <p>Recovery, movement, media, and spending share a month-shaped frame.</p>
  </header>
  <article class="cycle-chart">
    <BarChart
      categories={trend.map((item) => item.label)}
      primary={{
        name: primaryCurrency,
        values: trend.map((item) => item.amount),
        color: "var(--accent)",
        formatter: (value) =>
          formatMoney(value, primaryCurrency, primaryExponent),
      }}
      primaryType="line"
      height={280}
    />
  </article>
  <section class="category-cycle">
    <h3>Recurring categories</h3>
    <div>
      {#each categoryTotals as item}<span
          >{item.category}<b
            >{formatMoney(
              item.amount_minor,
              item.currency,
              item.currency_exponent,
            )}</b
          ></span
        >{/each}
    </div>
  </section>
  <ReportDetails {report} {formatMoney} {formatDuration} />
</section>

<style>
  .phenology-reports {
    display: grid;
    gap: 1.25rem;
  }
  .phenology-reports h2,
  .phenology-reports h3,
  .phenology-reports p {
    margin: 0;
  }
  .phenology-reports h2 {
    max-width: 43rem;
    font-size: clamp(2.4rem, 7vw, 5.5rem);
    letter-spacing: -0.11em;
    line-height: 0.88;
  }
  .phenology-reports header > p:last-child {
    max-width: 36rem;
    margin-top: 0.8rem;
    color: var(--text-muted);
    line-height: 1.5;
  }
  .kicker {
    color: var(--accent);
    font-size: 0.66rem;
    font-weight: 750;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  .cycle-chart,
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
  .category-cycle h3 {
    color: var(--accent);
    font-size: 0.85rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .category-cycle > div {
    display: flex;
    flex-wrap: wrap;
    gap: 0.55rem;
    margin-top: 1rem;
  }
  .category-cycle span {
    display: grid;
    gap: 0.3rem;
    border: 1px solid var(--border);
    padding: 0.7rem;
    font-size: 0.78rem;
  }
  .category-cycle b {
    color: var(--text-muted);
    font-size: 0.7rem;
    font-weight: 500;
  }
</style>
