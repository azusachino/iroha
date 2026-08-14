<script lang="ts">
  import type { DailyThemeProps } from "../../daily-view";
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

  function display(value: number | null | undefined, digits = 0): string {
    if (value == null || !Number.isFinite(value)) return "—";
    return value.toLocaleString(undefined, {
      maximumFractionDigits: digits,
      minimumFractionDigits: digits,
    });
  }

  function axisLabel(label: string): string {
    if (gran !== "day") return label;
    const date = new Date(`${label}T00:00:00Z`);
    return `${date.getUTCMonth() + 1}/${date.getUTCDate()}`;
  }
</script>

<section
  class="grapher-daily"
  data-theme={theme}
  aria-labelledby="daily-data-title"
>
  <header class="page-intro">
    <p class="kicker">Patterns / time series</p>
    <h1 id="daily-data-title">How does the pattern move?</h1>
    <p>
      Choose a time scale, compare the signals, and inspect the underlying
      periods.
    </p>
  </header>

  {@render children?.()}

  <div class="controls">
    <span class="kicker">Aggregation</span>
    {#each ["day", "month", "year"] as option}
      <button
        class:active={gran === option}
        onclick={() => onGran(option as "day" | "month" | "year")}
        >{option}</button
      >
    {/each}
    <span class="period-count">{chrono.length} periods</span>
  </div>

  {#if ringData.length}
    <section class="rings-panel" aria-labelledby="grapher-rings-title">
      <div class="panel-heading">
        <div>
          <p class="kicker">Latest reading</p>
          <h2 id="grapher-rings-title">Move, exercise, stand.</h2>
        </div>
        {#if latestRingDay}<strong>{latestRingDay.day}</strong>{/if}
      </div>
      <RingGauge rings={ringData} />
    </section>
  {/if}

  <section class="series-panel" aria-labelledby="steps-series-title">
    <div class="panel-heading">
      <div>
        <p class="kicker">Primary series</p>
        <h2 id="steps-series-title">Steps over time</h2>
      </div>
      {#if latest}<strong>{latest.label}</strong>{/if}
    </div>
    <BarChart
      categories={chrono.map((item) => axisLabel(item.label))}
      primary={{
        name: "Steps",
        values: chrono.map((item) => item.steps),
        formatter: (value) => value.toLocaleString(),
      }}
      secondary={{
        name: "Move closure",
        values: chrono.map((item) => item.moveClosedPct),
        formatter: (value) => `${value}%`,
      }}
      onBarClick={drillable ? onDrillIndex : undefined}
    />
    {#if drillable}
      <p class="drill-hint">Click a bar to zoom in.</p>
    {/if}
  </section>

  <section class="series-table" aria-labelledby="period-table-title">
    <div class="panel-heading">
      <div>
        <p class="kicker">Underlying periods</p>
        <h2 id="period-table-title">Compare signals</h2>
      </div>
      <span>— means no source value</span>
    </div>
    <div class="table-scroll">
      <table>
        <thead
          ><tr
            ><th>Period</th><th>Steps</th><th>Distance</th><th>Resting HR</th
            ><th>HRV</th><th>Move</th></tr
          ></thead
        >
        <tbody>
          {#each [...chrono].reverse() as item}
            <tr
              class:drillable
              onclick={drillable ? () => onDrillPeriod(item.period) : undefined}
              ><td>{item.label}</td><td>{display(item.steps)}</td><td
                >{display(item.distance, 1)}</td
              ><td>{display(item.resting_hr, 1)}</td><td
                >{display(item.hrv_sdnn, 1)}</td
              ><td
                >{item.moveClosedPct == null
                  ? "—"
                  : `${Math.round(item.moveClosedPct)}%`}</td
              ></tr
            >
          {/each}
        </tbody>
      </table>
    </div>
  </section>
</section>

<style>
  .grapher-daily {
    display: grid;
    gap: 1rem;
  }
  .page-intro {
    max-width: 48rem;
    padding-bottom: 2rem;
    border-bottom: 3px solid var(--text);
  }
  .kicker {
    margin: 0 0 0.45rem;
    color: var(--accent);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    letter-spacing: -0.07em;
  }
  h1 {
    font-size: clamp(2.8rem, 7vw, 6.5rem);
    line-height: 0.88;
  }
  h2 {
    font-size: 1.25rem;
  }
  .page-intro p:last-child {
    margin: 1rem 0 0;
    color: var(--text-muted);
  }
  .controls {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 0.45rem;
    padding: 0.75rem 0;
    border-bottom: 1px solid var(--border);
  }
  .controls .kicker {
    margin: 0 0.6rem 0 0;
  }
  .controls button {
    padding: 0.4rem 0.65rem;
    border: 1px solid var(--border);
    border-radius: 0;
    background: var(--surface);
    color: var(--text-muted);
    font: inherit;
    font-size: 0.75rem;
    cursor: pointer;
  }
  .controls button.active,
  .controls button:hover {
    border-color: var(--accent);
    color: var(--text);
  }
  .period-count {
    margin-left: auto;
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .series-panel,
  .series-table {
    padding: 1.25rem;
    border: 1px solid var(--border);
    background: var(--surface);
  }
  .panel-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .panel-heading > strong,
  .panel-heading > span {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .rings-panel {
    display: grid;
    gap: 1rem;
    padding: 1.25rem;
    border: 1px solid var(--border);
    background: var(--surface);
  }
  tr.drillable {
    cursor: pointer;
  }
  tr.drillable:hover td {
    background: color-mix(in srgb, var(--accent) 8%, transparent);
  }
  .drill-hint {
    margin: 0.75rem 0 0;
    color: var(--text-muted);
    font-size: 0.72rem;
    font-style: italic;
  }
  .table-scroll {
    overflow-x: auto;
    margin-top: 1.5rem;
    border-top: 2px solid var(--text);
  }
  table {
    width: 100%;
    min-width: 38rem;
    border-collapse: collapse;
    font-size: 0.78rem;
  }
  th,
  td {
    padding: 0.7rem 0.4rem;
    border-bottom: 1px solid var(--border);
    text-align: right;
    white-space: nowrap;
  }
  th:first-child,
  td:first-child {
    text-align: left;
  }
  th {
    color: var(--text-muted);
    font-size: 0.64rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
</style>
