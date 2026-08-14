<script lang="ts">
  import type { DailyThemeProps } from "../../daily-view";
  import { formatDateOnly } from "../../format";
  import RingGauge from "../components/RingGauge.svelte";
  import BarChart from "../components/BarChart.svelte";

  let {
    chrono,
    gran,
    onGran,
    onDrillIndex,
    onDrillPeriod,
    ringData,
    latestRingDay,
    theme,
    children,
  }: DailyThemeProps = $props();

  const drillable = $derived(gran !== "day");
  const latest = $derived(chrono.at(-1));

  function number(value: number | null | undefined, digits = 0): string {
    if (typeof value !== "number" || !Number.isFinite(value)) return "—";
    return value.toLocaleString(undefined, {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
    });
  }
</script>

<section class="mix-daily" data-theme={theme} aria-labelledby="mix-daily-title">
  <header class="mix-head">
    <div>
      <p class="mix-kicker">Patterns / {gran} interval</p>
      <h1 id="mix-daily-title">A longer signal.</h1>
      <p>
        Compare the periods without flattening the story into a single score.
      </p>
    </div>
    <div class="mix-readout">
      <strong>{chrono.length}</strong><span>periods</span>
    </div>
  </header>

  {@render children?.()}

  {#if ringData.length}
    <section class="mix-chart mix-rings" aria-labelledby="mix-rings-title">
      <header>
        <div>
          <p class="mix-kicker">Latest period</p>
          <h2 id="mix-rings-title">Move, exercise, stand.</h2>
        </div>
        {#if latestRingDay}<span>{formatDateOnly(latestRingDay.day)}</span>{/if}
      </header>
      <RingGauge rings={ringData} />
    </section>
  {/if}

  <nav class="mix-tabs" aria-label="Aggregation interval">
    {#each ["day", "month", "year"] as option (option)}
      <button
        class:active={gran === option}
        type="button"
        onclick={() => onGran(option as "day" | "month" | "year")}
        >{option}</button
      >
    {/each}
  </nav>

  <section class="mix-chart" aria-labelledby="mix-chart-title">
    <header>
      <div>
        <p class="mix-kicker">Primary series</p>
        <h2 id="mix-chart-title">Steps by period</h2>
      </div>
      {#if latest}<span>{latest.label} · {number(latest.steps)} steps</span
        >{/if}
    </header>
    {#if chrono.length}
      <BarChart
        categories={chrono.map((period) => period.label)}
        primary={{
          name: "Steps",
          values: chrono.map((period) => period.steps),
          formatter: (value) => value.toLocaleString(),
        }}
        secondary={{
          name: "Move closure",
          values: chrono.map((period) => period.moveClosedPct),
          formatter: (value) => `${value}%`,
        }}
        onBarClick={drillable ? onDrillIndex : undefined}
      />
      {#if drillable}
        <p class="drill-hint">Click a bar to zoom in.</p>
      {/if}
    {:else}
      <p class="mix-empty">No periods available for this interval.</p>
    {/if}
  </section>

  <div class="mix-notes">
    <article>
      <p class="mix-kicker">Latest period</p>
      <strong>{latest?.label ?? "—"}</strong><span
        >{number(latest?.distance, 1)} km · {number(latest?.resting_hr)} bpm</span
      >
    </article>
    <article>
      <p class="mix-kicker">Movement closure</p>
      <strong
        >{latest?.moveClosedPct == null
          ? "—"
          : `${latest.moveClosedPct}%`}</strong
      ><span>move goal recorded</span>
    </article>
    <article>
      <p class="mix-kicker">Recovery trace</p>
      <strong>{number(latest?.hrv_sdnn)} ms</strong><span>latest HRV value</span
      >
    </article>
  </div>

  <section class="mix-ledger">
    <header>
      <div>
        <p class="mix-kicker">Period ledger</p>
        <h2>Keep the detail.</h2>
      </div>
      <span>— means no source value</span>
    </header>
    <div class="ledger-scroll">
      <table>
        <thead
          ><tr
            ><th>Ch.</th><th>Period</th><th>Steps</th><th>Distance</th><th
              >Move</th
            ><th>Resting HR</th><th>HRV</th></tr
          ></thead
        ><tbody>
          {#each [...chrono].reverse() as period, index (period.label + index)}
            <tr
              class:drillable
              onclick={drillable
                ? () => onDrillPeriod(period.period)
                : undefined}
              ><td class="track-index"
                >{String(chrono.length - index).padStart(2, "0")}</td
              ><td>{period.label}</td><td>{number(period.steps)}</td><td
                >{number(period.distance, 1)} km</td
              ><td
                >{period.moveClosedPct == null
                  ? "—"
                  : `${period.moveClosedPct}%`}</td
              ><td>{number(period.resting_hr)} bpm</td><td
                >{number(period.hrv_sdnn)} ms</td
              ></tr
            >
          {/each}
        </tbody>
      </table>
    </div>
  </section>
  <footer class="mix-source">
    <span>Source: daily records and aggregates</span>
    <span>Presentation only</span>
  </footer>
</section>

<style>
  .mix-daily {
    display: grid;
    gap: 1.35rem;
    min-width: 0;
  }
  .mix-daily > * {
    min-width: 0;
  }
  .mix-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    font-weight: 700;
    letter-spacing: -0.03em;
  }
  h1 {
    font-size: clamp(2.3rem, 6vw, 4.2rem);
    line-height: 0.98;
    text-transform: uppercase;
  }
  h2 {
    font-size: 1.65rem;
  }
  .mix-head {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    padding-bottom: 1.75rem;
    border-bottom: 1px solid var(--border);
  }
  .mix-head p:last-child {
    max-width: 36rem;
    color: var(--text-muted);
    line-height: 1.6;
  }
  .mix-readout {
    display: grid;
    justify-items: end;
    padding: 0.6rem 0.9rem;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    color: var(--text-muted);
  }
  .mix-readout strong {
    color: var(--accent);
    font-size: 2.6rem;
    font-weight: 700;
    line-height: 1;
    font-variant-numeric: tabular-nums;
  }
  .mix-readout span {
    margin-top: 0.4rem;
    font-size: 0.62rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .mix-tabs {
    display: flex;
    gap: 0.4rem;
  }
  .mix-tabs button {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 0.45rem 0.9rem;
    background: transparent;
    color: var(--text-muted);
    font: inherit;
    font-size: 0.72rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    cursor: pointer;
  }
  .mix-tabs button.active,
  .mix-tabs button:hover {
    border-color: var(--accent);
    color: var(--accent);
  }
  .mix-chart,
  .mix-ledger,
  .mix-notes article {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
  }
  .mix-chart,
  .mix-ledger {
    padding: 1.4rem;
  }
  .mix-rings {
    margin-top: 1.5rem;
  }
  .mix-rings :global(.ring-gauge) {
    margin-top: 1.25rem;
  }
  .mix-chart header,
  .mix-ledger header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .mix-chart header > span,
  .mix-ledger header > span {
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  .mix-empty {
    margin-top: 1.4rem;
    color: var(--text-muted);
  }
  .mix-notes {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1rem;
  }
  .mix-notes article {
    min-height: 8rem;
    padding: 1.2rem;
  }
  .mix-notes strong,
  .mix-notes span {
    display: block;
  }
  .mix-notes strong {
    font-size: 1.6rem;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
  }
  .mix-notes span {
    margin-top: 0.5rem;
    color: var(--text-muted);
    font-size: 0.74rem;
  }
  .ledger-scroll {
    overflow-x: auto;
    margin-top: 1rem;
  }
  table {
    width: 100%;
    min-width: 40rem;
    border-collapse: collapse;
    font-size: 0.78rem;
  }
  th {
    color: var(--text-muted);
    font-size: 0.64rem;
    font-weight: 400;
    letter-spacing: 0.08em;
    text-align: left;
    text-transform: uppercase;
  }
  th,
  td {
    border-top: 1px solid var(--border);
    padding: 0.75rem 0.5rem;
    text-align: left;
    white-space: nowrap;
    font-variant-numeric: tabular-nums;
  }
  .track-index {
    color: var(--accent);
  }
  tr.drillable {
    cursor: pointer;
  }
  tr.drillable:hover td {
    background: color-mix(in srgb, var(--accent) 8%, transparent);
  }
  .drill-hint {
    margin: -0.5rem 0 0;
    color: var(--text-muted);
    font-size: 0.72rem;
    font-style: italic;
  }
  .mix-source {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding-top: 0.8rem;
    color: var(--text-muted);
    font-size: 0.66rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  @media (max-width: 768px) {
    .mix-head,
    .mix-chart header,
    .mix-ledger header,
    .mix-source {
      display: block;
    }
    .mix-readout {
      display: flex;
      align-items: baseline;
      justify-items: initial;
      gap: 0.5rem;
      margin-top: 1.5rem;
    }
    .mix-readout strong {
      font-size: 2.2rem;
    }
    .mix-chart header > span,
    .mix-ledger header > span {
      display: block;
      margin-top: 0.8rem;
    }
    .mix-notes {
      grid-template-columns: 1fr;
    }
  }
</style>
