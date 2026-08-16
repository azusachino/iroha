<script lang="ts">
  import type { DailyThemeProps } from "../../view-contracts/daily-view";
  import { formatDateOnly } from "../../format/format";
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
  const maxSteps = $derived(
    Math.max(1, ...chrono.map((period) => period.steps ?? 0)),
  );
  const averageSteps = $derived(
    chrono.length
      ? chrono.reduce((total, period) => total + (period.steps ?? 0), 0) /
          chrono.length
      : 0,
  );

  function number(value: number | null | undefined, digits = 0): string {
    if (typeof value !== "number" || !Number.isFinite(value)) return "—";
    return value.toLocaleString(undefined, {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
    });
  }
</script>

<section
  class="bloom-daily"
  data-theme={theme}
  aria-labelledby="bloom-daily-title"
>
  <header class="daily-opening">
    <div>
      <p class="bloom-kicker">◑ Pattern · {gran} cycle</p>
      <h1 id="bloom-daily-title">The seasons of a routine.</h1>
      <p>
        A pattern seen across periods, not flattened into a single verdict —
        where movement gathers, and where it rests.
      </p>
    </div>
    <div class="period-count">
      <strong>{chrono.length}</strong>
      <span>observed periods</span>
    </div>
  </header>

  {@render children?.()}

  {#if ringData.length}
    <section class="growth-panel rings-panel" aria-labelledby="rings-title">
      <div class="panel-heading">
        <div>
          <p class="bloom-kicker">Latest reading</p>
          <h2 id="rings-title">Move, exercise, stand.</h2>
        </div>
        {#if latestRingDay}<span>{formatDateOnly(latestRingDay.day)}</span>{/if}
      </div>
      <RingGauge rings={ringData} />
    </section>
  {/if}

  <nav class="bloom-tabs" aria-label="Aggregation interval">
    {#each ["day", "month", "year"] as option}
      <button
        class:active={gran === option}
        type="button"
        onclick={() => onGran(option as "day" | "month" | "year")}
        >{option}</button
      >
    {/each}
  </nav>

  <section class="growth-panel" aria-labelledby="growth-title">
    <div class="panel-heading">
      <div>
        <p class="bloom-kicker">The main thread</p>
        <h2 id="growth-title">Steps, growing and thinning</h2>
      </div>
      {#if latest}
        <p class="latest-note">
          <strong>{number(latest.steps)}</strong> steps<br />latest: {latest.label}
        </p>
      {/if}
    </div>
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

  <div class="bloom-notes">
    <article>
      <p class="bloom-kicker">A reading</p>
      <strong>{number(averageSteps)} steps</strong>
      <span>average across the visible {gran} entries</span>
      <div class="cycle-line">
        <i style={`width: ${Math.min(100, (averageSteps / maxSteps) * 100)}%`}
        ></i>
      </div>
    </article>
    <article>
      <p class="bloom-kicker">Movement closure</p>
      <strong
        >{latest?.moveClosedPct == null
          ? "—"
          : `${latest.moveClosedPct}%`}</strong
      >
      <span>move goal recorded, latest period</span>
    </article>
    <article>
      <p class="bloom-kicker">Recovery trace</p>
      <strong>{number(latest?.hrv_sdnn)} ms</strong>
      <span>latest HRV value</span>
    </article>
  </div>

  <section class="bloom-ledger">
    <header>
      <div>
        <p class="bloom-kicker">Period ledger</p>
        <h2>Each cycle, kept intact.</h2>
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
        >
        <tbody>
          {#each [...chrono].reverse() as period}
            <tr
              class:drillable
              onclick={drillable
                ? () => onDrillPeriod(period.period)
                : undefined}
            >
              <td>{period.label}</td>
              <td>{number(period.steps)}</td>
              <td>{number(period.distance, 1)} km</td>
              <td
                >{period.moveClosedPct == null
                  ? "—"
                  : `${period.moveClosedPct}%`}</td
              >
              <td>{number(period.resting_hr)} bpm</td>
              <td>{number(period.hrv_sdnn)} ms</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </section>

  <footer class="bloom-source">
    Source: daily records and aggregate periods · presentation only
  </footer>
</section>

<style>
  .bloom-daily {
    display: grid;
    gap: 1.5rem;
    font-family: var(--font-serif);
    min-width: 0;
  }
  .bloom-daily > * {
    min-width: 0;
  }
  .bloom-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.15em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    font-weight: 400;
    letter-spacing: -0.02em;
  }
  h1 {
    max-width: 13ch;
    font-size: clamp(2.5rem, 6vw, 4.9rem);
    line-height: 0.95;
  }
  h2 {
    font-size: 1.55rem;
  }
  .daily-opening {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    padding-bottom: 1.75rem;
    border-bottom: 1px solid var(--border);
  }
  .daily-opening p:last-child {
    max-width: 36rem;
    color: var(--text-muted);
    line-height: 1.65;
  }
  .period-count {
    display: grid;
    justify-items: end;
    color: var(--text-muted);
  }
  .period-count strong {
    color: var(--accent);
    font-family: inherit;
    font-style: italic;
    font-size: 3.2rem;
    font-weight: 400;
    line-height: 0.85;
  }
  .period-count span {
    margin-top: 0.5rem;
    font-size: 0.66rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .bloom-tabs {
    display: flex;
    gap: 0.4rem;
  }
  .bloom-tabs button {
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 0.45rem 0.9rem;
    background: transparent;
    color: var(--text-muted);
    font: inherit;
    font-size: 0.75rem;
    cursor: pointer;
  }
  .bloom-tabs button.active,
  .bloom-tabs button:hover {
    border-color: var(--accent);
    color: var(--accent);
  }
  .growth-panel,
  .bloom-notes article,
  .bloom-ledger {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
  }
  .growth-panel,
  .bloom-ledger {
    padding: clamp(1.25rem, 3vw, 2rem);
  }
  .rings-panel {
    margin-top: 1.5rem;
  }
  .rings-panel :global(.ring-gauge) {
    margin-top: 1.25rem;
  }
  .panel-heading,
  .bloom-ledger header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .latest-note,
  .bloom-ledger header > span {
    color: var(--text-muted);
    font-size: 0.75rem;
    text-align: right;
  }
  .latest-note strong {
    color: var(--accent);
    font-style: italic;
    font-size: 1.4rem;
    font-weight: 400;
  }
  .bloom-notes {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1rem;
  }
  .bloom-notes article {
    min-height: 8.5rem;
    padding: 1.25rem;
  }
  .bloom-notes strong,
  .bloom-notes span {
    display: block;
  }
  .bloom-notes strong {
    font-style: italic;
    font-size: 1.6rem;
    font-weight: 400;
  }
  .bloom-notes span {
    margin-top: 0.5rem;
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .cycle-line {
    height: 0.35rem;
    margin-top: 1.25rem;
    border-radius: 999px;
    background: var(--border);
    overflow: hidden;
  }
  .cycle-line i {
    display: block;
    height: 100%;
    border-radius: 999px;
    background: linear-gradient(90deg, var(--accent-2), var(--accent));
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
  }
  td:first-child {
    color: var(--accent);
    font-style: italic;
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
  .bloom-source {
    border-top: 1px solid var(--border);
    padding-top: 0.85rem;
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  @media (max-width: 768px) {
    .daily-opening,
    .panel-heading,
    .bloom-ledger header {
      display: block;
    }
    .period-count {
      display: flex;
      justify-items: initial;
      align-items: baseline;
      gap: 0.6rem;
      margin-top: 1.5rem;
    }
    .period-count strong {
      font-size: 2.4rem;
    }
    .latest-note {
      margin-top: 1rem;
      text-align: left;
    }
    .bloom-notes {
      grid-template-columns: 1fr;
    }
  }
</style>
