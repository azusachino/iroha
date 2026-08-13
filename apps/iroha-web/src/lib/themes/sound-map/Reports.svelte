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

<section class="sound-reports">
  <header>
    <p class="kicker">Signal report / {month}</p>
    <h2>A month with a pulse.</h2>
    <p>
      Intensity, quiet, and recurring bands stay visible without filling missing
      observations.
    </p>
  </header>
  <div class="signal">
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
  .sound-reports {
    display: grid;
    gap: 1rem;
  }
  .sound-reports h2,
  .sound-reports p {
    margin: 0;
  }
  .sound-reports h2 {
    font-size: clamp(2.8rem, 9vw, 7rem);
    letter-spacing: -0.13em;
    line-height: 0.82;
    text-shadow: 0 0 1.5rem color-mix(in srgb, var(--accent) 25%, transparent);
  }
  .sound-reports header > p:last-child {
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
  .signal {
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
</style>
