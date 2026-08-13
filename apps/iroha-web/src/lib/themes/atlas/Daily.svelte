<script lang="ts">
  import type { Snippet } from "svelte";
  import type { DailyRow } from "$lib/api";
  import { formatDateOnly } from "$lib/format";
  import RingGauge, { type Ring } from "$lib/components/RingGauge.svelte";
  import BarChart from "$lib/components/BarChart.svelte";

  type Period = {
    label: string;
    period: string;
    days: number | null;
    move: number | null;
    exercise: number | null;
    stand: number | null;
    moveClosedPct: number | null;
    steps: number | null;
    distance: number | null;
    resting_hr: number | null;
    hrv_sdnn: number | null;
  };

  let {
    chrono,
    gran,
    onGran,
    onDrillIndex,
    onDrillPeriod,
    ringData,
    latestRingDay,
    children,
  }: {
    chrono: Period[];
    gran: "day" | "month" | "year";
    onGran: (value: "day" | "month" | "year") => void;
    onDrillIndex: (index: number) => void;
    onDrillPeriod: (period: string) => void;
    ringData: Ring[];
    latestRingDay: DailyRow | null;
    children?: Snippet;
  } = $props();

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

<section class="atlas-daily" aria-labelledby="atlas-daily-title">
  <header class="daily-header">
    <div>
      <p class="atlas-kicker">Survey series · {gran} scale</p>
      <h1 id="atlas-daily-title">The territory, mapped over time.</h1>
      <p class="daily-sub">
        Repeated surveys of the same ground, laid side by side without smoothing
        the terrain into a single reading.
      </p>
    </div>
    <div class="grid-ref">
      <span>{chrono.length}</span>
      <small>periods surveyed</small>
    </div>
  </header>

  {@render children?.()}

  {#if ringData.length}
    <section
      class="atlas-plate rings-plate"
      aria-labelledby="daily-rings-title"
    >
      <header class="chart-heading">
        <div>
          <p class="atlas-kicker">Latest fix</p>
          <h2 id="daily-rings-title">Move, exercise, stand.</h2>
        </div>
        {#if latestRingDay}<span>{formatDateOnly(latestRingDay.day)}</span>{/if}
      </header>
      <RingGauge rings={ringData} />
    </section>
  {/if}

  <nav class="scale-select" aria-label="Aggregation interval">
    {#each ["day", "month", "year"] as option}
      <button
        class:active={gran === option}
        type="button"
        onclick={() => onGran(option as "day" | "month" | "year")}
        >{option}</button
      >
    {/each}
  </nav>

  <section
    class="atlas-plate contour-chart"
    aria-labelledby="daily-chart-title"
  >
    <header class="chart-heading">
      <div>
        <p class="atlas-kicker">Primary transect</p>
        <h2 id="daily-chart-title">Steps by period</h2>
      </div>
      {#if latest}<span>{latest.label} · {number(latest.steps)} steps</span
        >{/if}
    </header>
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
  </section>

  <div class="daily-notes">
    <article class="atlas-plate">
      <p class="atlas-kicker">Latest fix</p>
      <strong>{latest?.label ?? "—"}</strong><span
        >{number(latest?.distance, 1)} km · {number(latest?.resting_hr)} bpm</span
      >
    </article>
    <article class="atlas-plate">
      <p class="atlas-kicker">Move closure</p>
      <strong
        >{latest?.moveClosedPct == null
          ? "—"
          : `${latest.moveClosedPct}%`}</strong
      ><span>of move goal recorded</span>
    </article>
    <article class="atlas-plate">
      <p class="atlas-kicker">Recovery bearing</p>
      <strong>{number(latest?.hrv_sdnn)} ms</strong><span>latest HRV fix</span>
    </article>
  </div>

  <section class="atlas-plate ledger-plate">
    <header class="ledger-heading">
      <div>
        <p class="atlas-kicker">The gazetteer</p>
        <h2>Every period, indexed.</h2>
      </div>
      <span>— means no source value</span>
    </header>
    <div class="ledger-scroll">
      <table>
        <thead
          ><tr
            ><th>Period</th><th>Steps</th><th>Distance</th><th>Move</th><th
              >Resting HR</th
            ><th>HRV</th></tr
          ></thead
        ><tbody>
          {#each [...chrono].reverse() as period}
            <tr
              class:drillable
              onclick={drillable
                ? () => onDrillPeriod(period.period)
                : undefined}
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
  <footer class="atlas-source">
    Source: daily records and aggregates · presentation only
  </footer>
</section>

<style>
  .atlas-daily {
    display: grid;
    gap: 1.5rem;
    font-family: var(--font-sans);
  }
  .atlas-kicker {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.66rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .atlas-kicker::before {
    content: "⌖";
  }
  h1,
  h2 {
    margin: 0;
    font-weight: 600;
    letter-spacing: -0.03em;
  }
  h1 {
    max-width: 12ch;
    font-size: clamp(2.5rem, 6vw, 4.6rem);
    line-height: 1;
  }
  h2 {
    font-size: 1.45rem;
  }
  .daily-header {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    padding-bottom: 1.75rem;
    border-bottom: 1px solid var(--border);
  }
  .daily-sub {
    max-width: 40rem;
    margin: 1rem 0 0;
    color: var(--text-muted);
    line-height: 1.65;
  }
  .grid-ref {
    display: grid;
    justify-items: end;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    padding: 0.6rem 0.9rem;
    color: var(--accent);
    font-family: var(--font-mono);
    text-align: right;
  }
  .grid-ref span {
    font-size: 1.6rem;
  }
  .grid-ref small {
    margin-top: 0.2rem;
    color: var(--text-muted);
    font-size: 0.6rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  .scale-select {
    display: flex;
    gap: 0.4rem;
  }
  .scale-select button {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 0.45rem 0.9rem;
    background: transparent;
    color: var(--text-muted);
    font: inherit;
    font-size: 0.78rem;
    cursor: pointer;
  }
  .scale-select button.active,
  .scale-select button:hover {
    border-color: var(--accent);
    color: var(--accent);
  }
  .atlas-plate {
    position: relative;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface-1) 88%, transparent);
  }
  .atlas-plate::before,
  .atlas-plate::after {
    content: "";
    position: absolute;
    width: 0.7rem;
    height: 0.7rem;
    opacity: 0.7;
  }
  .atlas-plate::before {
    top: -1px;
    left: -1px;
    border-top: 2px solid var(--accent);
    border-left: 2px solid var(--accent);
  }
  .atlas-plate::after {
    right: -1px;
    bottom: -1px;
    border-right: 2px solid var(--accent);
    border-bottom: 2px solid var(--accent);
  }
  .contour-chart,
  .ledger-plate {
    padding: 1.5rem;
  }
  .rings-plate {
    margin-top: 1.5rem;
  }
  .rings-plate :global(.ring-gauge) {
    margin-top: 1.25rem;
  }
  .chart-heading,
  .ledger-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .chart-heading > span,
  .ledger-heading > span {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.72rem;
  }
  .daily-notes {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1rem;
  }
  .daily-notes article {
    min-height: 8rem;
    padding: 1.25rem;
  }
  .daily-notes strong,
  .daily-notes span {
    display: block;
  }
  .daily-notes strong {
    font-family: var(--font-mono);
    font-size: 1.6rem;
    font-weight: 600;
  }
  .daily-notes span {
    margin-top: 0.5rem;
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .ledger-scroll {
    overflow-x: auto;
    margin-top: 1rem;
  }
  table {
    width: 100%;
    min-width: 38rem;
    border-collapse: collapse;
    font-size: 0.78rem;
  }
  th {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.62rem;
    font-weight: 400;
    letter-spacing: 0.06em;
    text-align: left;
    text-transform: uppercase;
  }
  th,
  td {
    border-top: 1px solid var(--border);
    padding: 0.75rem 0.5rem;
    text-align: left;
    white-space: nowrap;
  }
  td {
    font-family: var(--font-mono);
  }
  td:first-child {
    color: var(--accent);
    font-family: var(--font-sans);
    font-weight: 600;
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
  .atlas-source {
    border-top: 1px solid var(--border);
    padding-top: 0.8rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.66rem;
  }
  @media (max-width: 680px) {
    .daily-header,
    .chart-heading,
    .ledger-heading {
      display: block;
    }
    .grid-ref {
      margin-top: 1.5rem;
      justify-items: start;
      text-align: left;
    }
    .chart-heading > span,
    .ledger-heading > span {
      display: block;
      margin-top: 0.6rem;
    }
    .daily-notes {
      grid-template-columns: 1fr;
    }
  }
</style>
