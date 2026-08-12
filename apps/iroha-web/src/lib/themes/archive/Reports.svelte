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

<section class="archive-reports">
  <header>
    <p class="kicker">Archive / monthly report</p>
    <h2>The report, exactly as generated.</h2>
    <span>{report.period.from} → {report.period.to} · {month}</span>
  </header>
  <ReportDetails {report} {formatMoney} {formatDuration} />
  <section class="derived">
    <h3>Derived index views</h3>
    <div>
      <BarChart
        categories={categoryTotals.map((item) => item.category)}
        primary={{
          name: primaryCurrency,
          values: categoryTotals.map((item) => item.amount_minor),
          color: "var(--accent)",
          formatter: (value) =>
            formatMoney(value, primaryCurrency, primaryExponent),
        }}
        orientation="horizontal"
        categorical
        height={250}
      /><BarChart
        categories={trend.map((item) => item.label)}
        primary={{
          name: primaryCurrency,
          values: trend.map((item) => item.amount),
          color: "var(--accent-2)",
          formatter: (value) =>
            formatMoney(value, primaryCurrency, primaryExponent),
        }}
        primaryType="line"
        height={250}
      />
    </div>
  </section>
</section>

<style>
  .archive-reports {
    display: grid;
    gap: 1.25rem;
    font-family: var(--font-mono);
  }
  .archive-reports h2,
  .archive-reports h3,
  .archive-reports p {
    margin: 0;
  }
  .archive-reports h2 {
    font-family: var(--font-sans);
    font-size: clamp(2.4rem, 6vw, 5rem);
    letter-spacing: -0.11em;
    line-height: 0.88;
  }
  .archive-reports header {
    display: flex;
    justify-content: space-between;
    align-items: end;
    gap: 1rem;
    border-bottom: 3px double var(--text);
    padding-bottom: 0.8rem;
  }
  .archive-reports header span {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .kicker {
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  .derived {
    border-top: 3px double var(--text);
    padding-top: 1rem;
  }
  .derived h3 {
    font-size: 0.85rem;
    text-transform: uppercase;
  }
  .derived > div {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
    margin-top: 0.8rem;
  }
  @media (max-width: 760px) {
    .archive-reports header,
    .derived > div {
      display: grid;
      grid-template-columns: 1fr;
    }
  }
</style>
