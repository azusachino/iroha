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

<section class="sound-reports">
  <header>
    <p class="kicker">Signal report / {month}</p>
    <h2>A month with a pulse.</h2>
    <p>
      Intensity, quiet, and recurring bands stay visible without filling missing
      observations.
    </p>
  </header>
  <article class="signal">
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
      height={310}
    />
  </article>
  <div class="bands">
    {#each categoryTotals as item, index}<div>
        <span>{item.category}</span><b
          >{formatMoney(
            item.amount_minor,
            item.currency,
            item.currency_exponent,
          )}</b
        ><i style={`width:${Math.max(8, 100 - index * 13)}%`}></i>
      </div>{/each}
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
  .bands {
    display: grid;
    gap: 0.45rem;
  }
  .bands div {
    display: grid;
    grid-template-columns: 8rem 6rem 1fr;
    gap: 0.8rem;
    align-items: center;
    font-family: var(--font-mono);
    font-size: 0.75rem;
  }
  .bands b {
    text-align: right;
  }
  .bands i {
    height: 0.55rem;
    border-radius: 1rem;
    background: var(--accent);
  }
</style>
