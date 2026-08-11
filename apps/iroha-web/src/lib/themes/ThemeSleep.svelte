<script lang="ts">
  import type { SleepSession } from "$lib/api";
  import { formatDateOnly, formatDuration } from "$lib/format";

  export type SleepVariant =
    "atlas" | "field-journal" | "phenology" | "sound-map" | "archive";
  let {
    variant,
    sessions,
    selected,
    averageAsleep,
    averageEfficiency,
    onSelect,
  }: {
    variant: SleepVariant;
    sessions: SleepSession[];
    selected: SleepSession | null;
    averageAsleep: number;
    averageEfficiency: number;
    onSelect: (session: SleepSession) => void;
  } = $props();

  const max = $derived(
    Math.max(1, ...sessions.map((session) => session.asleep_s)),
  );
</script>

<section
  class={`theme-sleep theme-sleep-${variant}`}
  aria-labelledby="theme-sleep-title"
>
  <header class="sleep-head">
    <div>
      <p class="theme-kicker">Night / recovery record</p>
      <h1 id="theme-sleep-title">How the night unfolds.</h1>
      <p>Rest is a sequence of observed sessions, not a single verdict.</p>
    </div>
    <strong>{sessions.length}<small> nights</small></strong>
  </header>
  <div class="sleep-summary">
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
  <section class="sleep-chart">
    <header>
      <div>
        <p class="theme-kicker">Observed nights</p>
        <h2>Asleep time</h2>
      </div>
      <span>select a column to inspect</span>
    </header>
    <div
      class="sleep-bars"
      role="img"
      aria-label="Asleep duration by recorded night"
    >
      {#each [...sessions].reverse() as session}<button
          class:active={selected?.id === session.id}
          title={formatDateOnly(session.wake_date)}
          onclick={() => onSelect(session)}
          ><i style={`height: ${Math.max(3, (session.asleep_s / max) * 100)}%`}
          ></i><small>{formatDateOnly(session.wake_date)}</small></button
        >{/each}
    </div>
  </section>
  <section class="sleep-table">
    <header>
      <div>
        <p class="theme-kicker">Session ledger</p>
        <h2>Night by night</h2>
      </div>
      <span>imported values</span>
    </header>
    <div class="sleep-scroll">
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
  {#if selected}<aside class="sleep-note">
      <p class="theme-kicker">Selected note</p>
      <strong>{formatDuration(selected.asleep_s)} asleep</strong><span
        >{selected.is_main_sleep ? "Primary overnight sleep" : "Short session"} ·
        {formatDateOnly(selected.wake_date)}</span
      >
    </aside>{/if}
  <footer class="sleep-source">
    Source: imported sleep sessions · no readiness score inferred
  </footer>
</section>

<style>
  .theme-sleep {
    display: grid;
    gap: 1.25rem;
  }
  .theme-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    font-family: Georgia, serif;
    font-weight: 400;
    letter-spacing: -0.05em;
  }
  h1 {
    font-size: clamp(2.7rem, 7vw, 5.7rem);
    line-height: 0.9;
  }
  h2 {
    font-size: 1.7rem;
  }
  .sleep-head {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 2rem;
  }
  .sleep-head p:last-child {
    color: var(--text-muted);
    line-height: 1.6;
  }
  .sleep-head > strong {
    color: var(--accent);
    font-family: Georgia, serif;
    font-size: 3.5rem;
    font-weight: 400;
    white-space: nowrap;
  }
  .sleep-head > strong small {
    color: var(--text-muted);
    font-family: inherit;
    font-size: 0.8rem;
  }
  .sleep-summary {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface-1) 88%, transparent);
  }
  .sleep-summary div {
    display: grid;
    gap: 0.4rem;
    border-right: 1px solid var(--border);
    padding: 1.25rem;
  }
  .sleep-summary div:last-child {
    border-right: 0;
  }
  .sleep-summary span {
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  .sleep-summary strong {
    font-family: Georgia, serif;
    font-size: 1.4rem;
    font-weight: 400;
  }
  .sleep-chart,
  .sleep-table {
    border: 1px solid var(--border);
    padding: 1.5rem;
    background: color-mix(in srgb, var(--surface-1) 88%, transparent);
  }
  .sleep-chart header,
  .sleep-table header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .sleep-chart header > span,
  .sleep-table header > span {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .sleep-bars {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(0.8rem, 1fr));
    align-items: end;
    gap: 0.4rem;
    height: 17rem;
    margin-top: 1.5rem;
    border-bottom: 1px solid var(--border);
  }
  .sleep-bars button {
    display: grid;
    grid-template-rows: 1fr auto;
    align-items: end;
    height: 100%;
    border: 0;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
  }
  .sleep-bars i {
    display: block;
    width: 72%;
    min-height: 0.2rem;
    margin: 0 auto;
    background: var(--accent);
    opacity: 0.65;
  }
  .sleep-bars button.active i,
  .sleep-bars button:hover i {
    opacity: 1;
  }
  .sleep-bars small {
    overflow: hidden;
    margin-top: 0.5rem;
    font-size: 0.55rem;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .sleep-scroll {
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
  }
  tbody tr {
    cursor: pointer;
  }
  tbody tr.selected td {
    color: var(--accent);
  }
  .sleep-note {
    display: grid;
    gap: 0.35rem;
    border-left: 0.35rem solid var(--accent);
    padding: 1.2rem 1.5rem;
    background: color-mix(in srgb, var(--accent) 8%, var(--surface-1));
  }
  .sleep-note strong {
    font-family: Georgia, serif;
    font-size: 1.6rem;
    font-weight: 400;
  }
  .sleep-note span {
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .sleep-source {
    border-top: 1px solid var(--border);
    padding-top: 0.8rem;
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  .theme-sleep-atlas {
    font-family: "Avenir Next", sans-serif;
  }
  .theme-sleep-atlas .sleep-chart {
    border-top: 0.35rem solid var(--accent);
  }
  .theme-sleep-phenology h1,
  .theme-sleep-phenology h2 {
    font-style: italic;
  }
  .theme-sleep-phenology .sleep-note {
    border-radius: 1rem 0.2rem;
  }
  .theme-sleep-sound-map {
    font-family: "IBM Plex Mono", monospace;
  }
  .theme-sleep-sound-map h1,
  .theme-sleep-sound-map h2 {
    font-family: inherit;
  }
  .theme-sleep-archive .sleep-table {
    box-shadow: 0.45rem 0.45rem 0
      color-mix(in srgb, var(--accent) 18%, transparent);
  }
  @media (max-width: 680px) {
    .sleep-head,
    .sleep-chart header,
    .sleep-table header {
      display: block;
    }
    .sleep-head > strong {
      display: block;
      margin-top: 1.5rem;
      font-size: 2.6rem;
    }
    .sleep-chart header > span,
    .sleep-table header > span {
      display: block;
      margin-top: 0.8rem;
    }
    .sleep-summary {
      grid-template-columns: 1fr;
    }
    .sleep-summary div {
      border-right: 0;
      border-bottom: 1px solid var(--border);
    }
    .sleep-summary div:last-child {
      border-bottom: 0;
    }
  }
</style>
