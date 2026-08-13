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

<section class="atlas-reports">
  <header>
    <p class="kicker">Monthly survey · {month}</p>
    <h2>Read the month as a territory.</h2>
    <p>
      Movement, rest, library, and ledger plotted together; source details
      remain below.
    </p>
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
  .atlas-reports {
    display: grid;
    gap: 1.25rem;
  }
  .atlas-reports h2,
  .atlas-reports p {
    margin: 0;
  }
  .atlas-reports h2 {
    font-size: clamp(2.2rem, 6vw, 5rem);
    letter-spacing: -0.1em;
    line-height: 0.88;
  }
  .atlas-reports header > p:last-child {
    max-width: 35rem;
    margin-top: 0.8rem;
    color: var(--text-muted);
    line-height: 1.5;
  }
  .kicker {
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.66rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
</style>
