<script lang="ts">
  import type { SleepSession } from "$lib/api";
  import { formatDateOnly, formatDateShort, formatDuration } from "$lib/format";
  import BarChart from "$lib/components/BarChart.svelte";

  let {
    sessions,
    selected,
    averageAsleep,
    averageEfficiency,
    onSelect,
  }: {
    sessions: SleepSession[];
    selected: SleepSession | null;
    averageAsleep: number;
    averageEfficiency: number;
    onSelect: (session: SleepSession) => void;
  } = $props();

  const chartSessions = $derived(sessions.slice(0, 28).reverse());
  const activeChartIndex = $derived(
    selected
      ? chartSessions.findIndex((session) => session.id === selected.id)
      : null,
  );
</script>

<section class="mix-sleep" aria-labelledby="mix-sleep-title">
  <header class="mix-head">
    <div>
      <p class="mix-kicker">Night record / recovery</p>
      <h1 id="mix-sleep-title">How the night unfolds.</h1>
      <p>Rest is a sequence of recorded levels, not a single verdict.</p>
    </div>
    <div class="mix-readout">
      <strong>{sessions.length}</strong><span>nights</span>
    </div>
  </header>

  <div class="mix-summary">
    <div>
      <span>Average asleep</span><strong>{formatDuration(averageAsleep)}</strong
      >
    </div>
    <div>
      <span>Average efficiency</span><strong
        >{Math.round(averageEfficiency * 100)}%</strong
      >
    </div>
    <div>
      <span>Selected</span><strong
        >{selected ? formatDateOnly(selected.wake_date) : "—"}</strong
      >
    </div>
  </div>

  <section class="mix-chart">
    <header>
      <div>
        <p class="mix-kicker">Observed nights</p>
        <h2>Sleep waveform</h2>
      </div>
      <span>bar = time asleep · cap = efficiency</span>
    </header>
    {#if sessions.length}
      <BarChart
        categories={chartSessions.map((session) =>
          formatDateShort(session.wake_date),
        )}
        primary={{
          name: "Asleep",
          values: chartSessions.map((session) => session.asleep_s),
          formatter: (value) => formatDuration(value),
        }}
        secondary={{
          name: "Efficiency",
          values: chartSessions.map((session) =>
            Math.round(session.efficiency * 100),
          ),
          formatter: (value) => `${value}%`,
        }}
        activeIndex={activeChartIndex}
        onBarClick={(index) => onSelect(chartSessions[index])}
      />
    {:else}
      <p class="mix-empty">No sleep sessions were recorded.</p>
    {/if}
  </section>

  {#if selected}
    <aside class="mix-note">
      <p class="mix-kicker">Selected night</p>
      <strong>{formatDuration(selected.asleep_s)} asleep</strong><span
        >{selected.is_main_sleep ? "Primary overnight sleep" : "Short session"} ·
        {formatDateOnly(selected.wake_date)} · {Math.round(
          selected.efficiency * 100,
        )}% efficient</span
      >
    </aside>
  {/if}

  <section class="mix-table">
    <header>
      <div>
        <p class="mix-kicker">Session ledger</p>
        <h2>Night by night</h2>
      </div>
      <span>imported values</span>
    </header>
    <div class="mix-scroll">
      <table>
        <thead
          ><tr
            ><th>Date</th><th>Asleep</th><th>In bed</th><th>Efficiency</th><th
              >Type</th
            ></tr
          ></thead
        ><tbody>
          {#each sessions as session (session.id)}<tr
              class:selected={selected?.id === session.id}
              onclick={() => onSelect(session)}
              ><td>{formatDateOnly(session.wake_date)}</td><td
                >{formatDuration(session.asleep_s)}</td
              ><td>{formatDuration(session.time_in_bed_s)}</td><td
                >{Math.round(session.efficiency * 100)}%</td
              ><td>{session.is_main_sleep ? "Main sleep" : "Nap"}</td></tr
            >{/each}
        </tbody>
      </table>
    </div>
  </section>
  <footer class="mix-source">
    <span>Source: imported sleep sessions</span>
    <span>No readiness score inferred</span>
  </footer>
</section>

<style>
  .mix-sleep {
    display: grid;
    gap: 1.35rem;
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
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.75rem;
  }
  .mix-head p:last-child {
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
  .mix-summary {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
  }
  .mix-summary div {
    display: grid;
    gap: 0.4rem;
    border-right: 1px solid var(--border);
    padding: 1.2rem;
  }
  .mix-summary div:last-child {
    border-right: 0;
  }
  .mix-summary span {
    color: var(--text-muted);
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  .mix-summary strong {
    font-size: 1.4rem;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
  }
  .mix-chart,
  .mix-table {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 1.4rem;
    background: color-mix(in srgb, var(--surface) 90%, transparent);
  }
  .mix-chart header,
  .mix-table header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .mix-chart header > span,
  .mix-table header > span {
    color: var(--text-muted);
    font-size: 0.7rem;
    text-align: right;
  }
  .mix-empty {
    margin-top: 1.4rem;
    color: var(--text-muted);
  }
  .mix-note {
    display: grid;
    gap: 0.35rem;
    border-left: 0.3rem solid var(--accent);
    border-radius: var(--radius);
    padding: 1.2rem 1.5rem;
    background: color-mix(in srgb, var(--accent) 8%, var(--surface));
  }
  .mix-note strong {
    font-size: 1.5rem;
    font-weight: 700;
  }
  .mix-note span {
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .mix-scroll {
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
    padding: 0.8rem 0.5rem;
    text-align: left;
    white-space: nowrap;
    font-variant-numeric: tabular-nums;
  }
  tbody tr {
    cursor: pointer;
  }
  tbody tr.selected td {
    color: var(--accent);
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
  @media (max-width: 680px) {
    .mix-head,
    .mix-chart header,
    .mix-table header,
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
    .mix-table header > span {
      display: block;
      margin-top: 0.8rem;
    }
    .mix-summary {
      grid-template-columns: 1fr;
    }
    .mix-summary div {
      border-right: 0;
      border-bottom: 1px solid var(--border);
    }
    .mix-summary div:last-child {
      border-bottom: 0;
    }
  }
</style>
