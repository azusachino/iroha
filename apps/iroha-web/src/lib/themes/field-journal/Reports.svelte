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
  <div class="observations">
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
  .journal-reports {
    display: grid;
    gap: 1.25rem;
    font-family: Georgia, "Times New Roman", serif;
  }
  .journal-reports h2,
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
  .observations {
    border: 1px solid var(--border);
    padding: 1rem;
    background: color-mix(in srgb, var(--surface) 88%, var(--accent) 12%);
  }
</style>
