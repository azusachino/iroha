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

<section class="archive-reports">
  <header>
    <p class="kicker">Archive / monthly report</p>
    <h2>The report, exactly as generated.</h2>
    <span>{report.period.from} → {report.period.to} · {month}</span>
  </header>
  <section class="derived">
    <h3>Canonical domain index</h3>
    <ReportDomainCharts
      {report}
      {primaryCurrency}
      {primaryExponent}
      {formatMoney}
      {formatDuration}
    />
  </section>
  <ReportDetails {report} {formatMoney} {formatDuration} />
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
  .derived :global(.domain-charts) {
    margin-top: 0.8rem;
  }
  @media (max-width: 760px) {
    .archive-reports header {
      display: grid;
      grid-template-columns: 1fr;
    }
  }
</style>
