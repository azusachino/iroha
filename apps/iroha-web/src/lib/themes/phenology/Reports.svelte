<script lang="ts">
  import ReportDomainCharts from "$lib/components/ReportDomainCharts.svelte";
  import ReportDetails from "$lib/components/ReportDetails.svelte";
  import type { ReportThemeProps } from "$lib/report-view";
  let {
    month,
    report,
    primaryCurrency,
    primaryExponent,
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
  <div class="cycle-chart">
    <ReportDomainCharts
      {report}
      {primaryCurrency}
      {primaryExponent}
      {formatMoney}
      {formatDuration}
    />
  </div>
  <ReportDetails {report} {formatMoney} {formatDuration} />
</section>

<style>
  .phenology-reports {
    display: grid;
    gap: 1.25rem;
  }
  .phenology-reports h2,
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
  .cycle-chart {
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
</style>
