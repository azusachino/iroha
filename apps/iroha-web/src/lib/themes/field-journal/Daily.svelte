<script lang="ts">
  import type { DailyRow } from "$lib/api";
  import { formatDateOnly } from "$lib/format";
  import RingGauge, { type Ring } from "$lib/components/RingGauge.svelte";
  import BarChart from "$lib/components/BarChart.svelte";

  type JournalPeriod = {
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
  }: {
    chrono: JournalPeriod[];
    gran: "day" | "month" | "year";
    onGran: (value: "day" | "month" | "year") => void;
    onDrillIndex: (index: number) => void;
    onDrillPeriod: (period: string) => void;
    ringData: Ring[];
    latestRingDay: DailyRow | null;
  } = $props();

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

<section class="journal-daily" aria-labelledby="journal-daily-title">
  <header class="daily-opening">
    <div>
      <p class="journal-kicker">Pattern notebook · {gran} view</p>
      <h1 id="journal-daily-title">The shape of the days.</h1>
      <p>
        A quiet record of repetition: where movement gathers, where it thins,
        and which signals stay with you.
      </p>
    </div>
    <div class="period-count">
      <strong>{chrono.length}</strong>
      <span>observed periods</span>
    </div>
  </header>

  {#if ringData.length}
    <section class="pattern-card rings-card" aria-labelledby="rings-title">
      <div class="pattern-heading">
        <div>
          <p class="journal-kicker">Latest entry</p>
          <h2 id="rings-title">Move, exercise, stand.</h2>
        </div>
        {#if latestRingDay}
          <p class="latest-note">{formatDateOnly(latestRingDay.day)}</p>
        {/if}
      </div>
      <RingGauge rings={ringData} />
    </section>
  {/if}

  <div class="journal-rule"><span>turn the page</span></div>

  <nav class="granularity" aria-label="Pattern time scale">
    {#each ["day", "month", "year"] as option}
      <button
        class:active={gran === option}
        type="button"
        onclick={() => onGran(option as "day" | "month" | "year")}
        >{option}</button
      >
    {/each}
  </nav>

  <section class="pattern-card" aria-labelledby="steps-title">
    <div class="pattern-heading">
      <div>
        <p class="journal-kicker">The main thread</p>
        <h2 id="steps-title">Steps, over time</h2>
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

  <div class="notebook-grid">
    <section class="notebook-card">
      <p class="journal-kicker">A reading</p>
      <h2>{number(averageSteps)} steps</h2>
      <p>average across the visible {gran} entries.</p>
      <div class="ink-line">
        <i style={`width: ${Math.min(100, (averageSteps / maxSteps) * 100)}%`}
        ></i>
      </div>
    </section>
    <section class="notebook-card">
      <p class="journal-kicker">Latest margin</p>
      {#if latest}
        <dl>
          <div>
            <dt>Distance</dt>
            <dd>{number(latest.distance, 1)} km</dd>
          </div>
          <div>
            <dt>Resting heart</dt>
            <dd>{number(latest.resting_hr)} bpm</dd>
          </div>
          <div>
            <dt>HRV</dt>
            <dd>{number(latest.hrv_sdnn)} ms</dd>
          </div>
        </dl>
      {:else}
        <p>No visible values yet.</p>
      {/if}
    </section>
  </div>

  <section class="period-ledger" aria-labelledby="ledger-title">
    <header>
      <div>
        <p class="journal-kicker">The ledger</p>
        <h2 id="ledger-title">Each period, kept intact.</h2>
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

  <footer class="journal-source">
    Source: daily records and aggregate periods · presentation only
  </footer>
</section>

<style>
  .journal-daily {
    display: grid;
    gap: 1.5rem;
  }
  .daily-opening {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
  }
  .journal-kicker {
    margin: 0 0 0.55rem;
    color: var(--accent);
    font-size: 0.68rem;
    letter-spacing: 0.16em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    font-family: var(--font-serif);
    font-weight: 400;
    letter-spacing: -0.04em;
  }
  h1 {
    max-width: 11ch;
    margin: 0;
    font-size: clamp(2.8rem, 7vw, 5.7rem);
    line-height: 0.92;
  }
  h2 {
    margin: 0.25rem 0 0.7rem;
    font-size: clamp(1.45rem, 3vw, 2.1rem);
  }
  .daily-opening p:last-child,
  .notebook-card > p:last-child {
    max-width: 37rem;
    color: var(--text-muted);
    line-height: 1.7;
  }
  .period-count {
    display: grid;
    justify-items: end;
    color: var(--text-muted);
  }
  .period-count strong {
    color: var(--accent);
    font-family: var(--font-serif);
    font-size: 3.5rem;
    font-weight: 400;
    line-height: 0.8;
  }
  .period-count span {
    margin-top: 0.6rem;
    font-size: 0.68rem;
    text-transform: uppercase;
  }
  .journal-rule {
    display: flex;
    align-items: center;
    gap: 1rem;
    color: var(--text-muted);
    font-family: var(--font-serif);
    font-size: 0.8rem;
    font-style: italic;
  }
  .journal-rule::after {
    content: "";
    height: 1px;
    flex: 1;
    background: var(--border);
  }
  .granularity {
    display: flex;
    gap: 0.4rem;
    border-bottom: 1px solid var(--border);
    padding-bottom: 0.8rem;
  }
  .granularity button {
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 0.45rem 0.8rem;
    background: transparent;
    color: var(--text-muted);
    font: inherit;
    font-size: 0.75rem;
    cursor: pointer;
  }
  .granularity button.active,
  .granularity button:hover {
    border-color: var(--accent);
    color: var(--accent);
  }
  .pattern-card,
  .notebook-card,
  .period-ledger {
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface-1) 84%, transparent);
  }
  .pattern-card {
    padding: clamp(1.25rem, 4vw, 2.5rem);
  }
  .rings-card {
    margin-top: 1.5rem;
  }
  .rings-card :global(.ring-gauge) {
    margin-top: 1.25rem;
  }
  .pattern-heading,
  .period-ledger header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .latest-note,
  .period-ledger header > span {
    color: var(--text-muted);
    font-size: 0.75rem;
    text-align: right;
  }
  .latest-note strong {
    color: var(--accent);
    font-family: var(--font-serif);
    font-size: 1.5rem;
    font-weight: 400;
  }
  .notebook-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 1rem;
  }
  .notebook-card {
    min-height: 13rem;
    padding: 1.5rem;
  }
  .notebook-card h2 {
    font-size: 2.25rem;
  }
  .ink-line {
    height: 0.35rem;
    margin-top: 2rem;
    background: var(--border);
  }
  .ink-line i {
    display: block;
    height: 100%;
    background: var(--accent);
  }
  dl {
    display: grid;
    gap: 0.8rem;
    margin: 1.2rem 0 0;
  }
  dl div {
    display: flex;
    justify-content: space-between;
    border-bottom: 1px dotted var(--border);
    padding-bottom: 0.5rem;
  }
  dt {
    color: var(--text-muted);
  }
  dd {
    margin: 0;
    font-family: var(--font-serif);
  }
  .period-ledger {
    padding: 1.5rem;
  }
  .ledger-scroll {
    overflow-x: auto;
    margin-top: 1rem;
  }
  table {
    width: 100%;
    border-collapse: collapse;
    min-width: 38rem;
    font-size: 0.78rem;
  }
  th {
    color: var(--text-muted);
    font-size: 0.65rem;
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
    font-family: var(--font-serif);
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
  .journal-source {
    border-top: 1px solid var(--border);
    padding-top: 1rem;
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  @media (max-width: 680px) {
    .daily-opening,
    .pattern-heading,
    .period-ledger header {
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
      font-size: 2.5rem;
    }
    .latest-note {
      margin-top: 1rem;
      text-align: left;
    }
    .notebook-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
