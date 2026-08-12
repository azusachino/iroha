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

<section class="journal-reports">
  <header>
    <p class="date">{month}</p>
    <p class="kicker">Field report · monthly entry</p>
    <h2>What the month held.</h2>
    <p class="note">
      A dated record of observed domains, with no readiness score and no
      invented narrative.
    </p>
  </header>
  <article class="journal-chart">
    <div><span>Spending notes</span><strong>{primaryCurrency}</strong></div>
    <BarChart
      categories={trend.map((item) => item.label)}
      primary={{
        name: primaryCurrency,
        values: trend.map((item) => item.amount),
        color: "var(--accent-2)",
        formatter: (value) =>
          formatMoney(value, primaryCurrency, primaryExponent),
      }}
      primaryType="line"
      height={270}
    />
  </article>
  <section class="category-note">
    <h3>Repeated entries</h3>
    {#each categoryTotals as item}<p>
        <span>{item.category}</span><b
          >{formatMoney(
            item.amount_minor,
            item.currency,
            item.currency_exponent,
          )}</b
        >
      </p>{/each}
  </section>
  <ReportDetails {report} {formatMoney} {formatDuration} />
</section>

<style>
  .journal-reports {
    display: grid;
    gap: 1.25rem;
    font-family: Georgia, "Times New Roman", serif;
  }
  .journal-reports h2,
  .journal-reports h3,
  .journal-reports p {
    margin: 0;
  }
  .journal-reports h2 {
    font-size: clamp(2.5rem, 7vw, 5.5rem);
    font-weight: 500;
    letter-spacing: -0.1em;
    line-height: 0.88;
  }
  .date,
  .kicker {
    color: var(--accent);
    font-size: 0.68rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .note {
    max-width: 36rem;
    margin-top: 0.8rem !important;
    color: var(--text-muted);
    line-height: 1.5;
  }
  .journal-chart,
  .category-note {
    border: 1px solid var(--border);
    padding: 1rem;
    background: color-mix(in srgb, var(--surface) 88%, var(--accent) 12%);
  }
  .journal-chart > div {
    display: flex;
    justify-content: space-between;
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  .journal-chart strong {
    color: var(--text);
  }
  .category-note h3 {
    font-size: 1rem;
    font-weight: 500;
  }
  .category-note p {
    display: flex;
    justify-content: space-between;
    border-bottom: 1px dashed var(--border);
    padding: 0.65rem 0;
  }
  .category-note b {
    font-weight: 500;
  }
</style>
