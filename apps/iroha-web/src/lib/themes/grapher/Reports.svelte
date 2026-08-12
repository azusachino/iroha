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
    currentTotal,
    previousTotal,
    comparisonLabel,
    formatMoney,
    formatDuration,
  }: ReportThemeProps = $props();
</script>

<section class="grapher-reports">
  <header>
    <p class="kicker">Monthly comparison / {month}</p>
    <h2>What changed?</h2>
    <p>{comparisonLabel}</p>
  </header>
  <article class="hero-chart">
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
      height={320}
    />
  </article>
  <div class="comparison">
    <strong
      >{formatMoney(currentTotal, primaryCurrency, primaryExponent)}</strong
    ><span
      >versus {formatMoney(previousTotal, primaryCurrency, primaryExponent)} in the
      prior month</span
    >
  </div>
  <article class="category-chart">
    <p class="kicker">Category comparison · current month</p>
    <BarChart
      categories={categoryTotals.map((item) => item.category)}
      primary={{
        name: primaryCurrency,
        values: categoryTotals.map((item) => item.amount_minor),
        color: "var(--accent-2)",
        formatter: (value) =>
          formatMoney(value, primaryCurrency, primaryExponent),
      }}
      orientation="horizontal"
      categorical
      height={250}
    />
  </article>
  <ReportDetails {report} {formatMoney} {formatDuration} />
</section>

<style>
  .grapher-reports {
    display: grid;
    gap: 1rem;
    font-family: "IBM Plex Mono", "SFMono-Regular", monospace;
  }
  .grapher-reports h2,
  .grapher-reports p {
    margin: 0;
  }
  .grapher-reports h2 {
    font-size: clamp(2.8rem, 8vw, 6rem);
    letter-spacing: -0.12em;
    line-height: 0.84;
  }
  .grapher-reports header > p:last-child {
    margin-top: 0.8rem;
    color: var(--text-muted);
    font-family: var(--font-sans);
  }
  .kicker {
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .hero-chart,
  .category-chart {
    min-width: 0;
    border-top: 3px solid var(--text);
    padding: 1rem 0;
  }
  .comparison {
    display: flex;
    align-items: baseline;
    gap: 1rem;
    border-block: 1px solid var(--border);
    padding: 1rem 0;
  }
  .comparison strong {
    font-size: 2rem;
  }
  .comparison span {
    color: var(--text-muted);
    font-family: var(--font-sans);
    font-size: 0.8rem;
  }
</style>
