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

<section class="grapher-reports">
  <header>
    <p class="kicker">Monthly comparison / {month}</p>
    <h2>What changed?</h2>
    <p>Comparable canonical domains, one month at a time.</p>
  </header>
  <ReportDomainCharts
    {report}
    {primaryCurrency}
    {primaryExponent}
    {formatMoney}
    {formatDuration}
  />
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
</style>
