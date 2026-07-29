<script lang="ts">
  import type { SleepSession } from "$lib/api";
  import { formatDateOnly, formatDuration } from "$lib/format";

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

  const max = $derived(
    Math.max(1, ...sessions.map((session) => session.asleep_s)),
  );

  function tone(pct: number | null | undefined): string {
    if (pct == null || !Number.isFinite(pct))
      return "color-mix(in srgb, var(--border) 55%, var(--surface))";
    const clamped = Math.max(0, Math.min(100, pct));
    return `color-mix(in srgb, var(--accent-2) ${clamped}%, var(--accent) ${100 - clamped}%)`;
  }

  // Each recorded night becomes a stratum: thickness is real time asleep,
  // tone is real efficiency. Most recent night on top, older nights settle
  // toward the bottom of the column -- the same reading order as the
  // period core on the Daily page.
  const rows = $derived(
    sessions.slice(0, 30).map((session) => ({
      session,
      magnitude: Math.max(session.asleep_s / max, 0.05),
      tone: tone(session.efficiency * 100),
    })),
  );
</script>

<section class="folio-sleep" aria-labelledby="folio-sleep-title">
  <header class="folio-head">
    <div>
      <p class="folio-kicker">Night register / recovery</p>
      <h1 id="folio-sleep-title">How the night settles.</h1>
      <p>Rest is a sequence of recorded strata, not a single verdict.</p>
    </div>
    <div class="folio-readout">
      <strong>{sessions.length}</strong><span>nights held</span>
    </div>
  </header>

  <div class="folio-summary catalog-card">
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

  <section class="folio-core catalog-card">
    <header>
      <div>
        <p class="folio-kicker">Observed nights</p>
        <h2>Sleep core</h2>
      </div>
      <span>thickness = asleep · tone = efficiency</span>
    </header>
    {#if rows.length}
      <div
        class="core-log"
        role="img"
        aria-label="Asleep duration by recorded night, with efficiency as tone"
      >
        <div class="core-strip">
          {#each rows as row (row.session.id)}
            <button
              type="button"
              class="core-band"
              class:active={selected?.id === row.session.id}
              style={`flex-grow: ${row.magnitude}; background: ${row.tone};`}
              title={`${formatDateOnly(row.session.wake_date)} · ${formatDuration(row.session.asleep_s)} asleep · ${Math.round(row.session.efficiency * 100)}% efficient`}
              onclick={() => onSelect(row.session)}
            ></button>
          {/each}
        </div>
        <div class="core-legend">
          {#each rows as row (row.session.id)}
            <button
              type="button"
              class="core-row"
              class:active={selected?.id === row.session.id}
              style={`flex-grow: ${row.magnitude};`}
              onclick={() => onSelect(row.session)}
            >
              <strong>{formatDateOnly(row.session.wake_date)}</strong>
              <span
                >{formatDuration(row.session.asleep_s)} · {Math.round(
                  row.session.efficiency * 100,
                )}%</span
              >
            </button>
          {/each}
        </div>
      </div>
    {:else}
      <p class="folio-empty">No sleep sessions were recorded.</p>
    {/if}
  </section>

  {#if selected}
    <aside class="folio-note catalog-card">
      <p class="folio-kicker">Selected night</p>
      <strong>{formatDuration(selected.asleep_s)} asleep</strong><span
        >{selected.is_main_sleep ? "Primary overnight sleep" : "Short session"} ·
        {formatDateOnly(selected.wake_date)} · {Math.round(
          selected.efficiency * 100,
        )}% efficient</span
      >
    </aside>
  {/if}

  <section class="folio-ledger">
    <header>
      <div>
        <p class="folio-kicker">Session ledger</p>
        <h2>Night by night</h2>
      </div>
      <span>imported values</span>
    </header>
    <div class="ledger-scroll">
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
  <footer class="folio-source">
    Source: imported sleep sessions · no readiness score inferred
  </footer>
</section>

<style>
  .folio-sleep {
    display: grid;
    gap: 1.3rem;
  }
  .folio-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-family: "Courier New", "Consolas", ui-monospace, monospace;
    font-size: 0.64rem;
    letter-spacing: 0.15em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    font-family: "Rockwell", "Roboto Slab", Georgia, serif;
    font-weight: 700;
    letter-spacing: -0.01em;
  }
  h1 {
    font-size: clamp(2.5rem, 6.5vw, 5rem);
    line-height: 0.95;
  }
  h2 {
    font-size: 1.55rem;
  }
  .folio-head {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.75rem;
  }
  .folio-head p:last-child {
    color: var(--text-muted);
    line-height: 1.6;
  }
  .folio-readout {
    display: grid;
    justify-items: end;
    color: var(--text-muted);
  }
  .folio-readout strong {
    color: var(--accent);
    font-family: "Rockwell", "Roboto Slab", Georgia, serif;
    font-size: 3.3rem;
    font-weight: 700;
    white-space: nowrap;
  }
  .folio-readout span {
    margin-top: 0.5rem;
    font-family: "Courier New", "Consolas", ui-monospace, monospace;
    font-size: 0.64rem;
    text-transform: uppercase;
  }
  .catalog-card {
    position: relative;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
    padding: 1.5rem 1.5rem 1.5rem 1.7rem;
  }
  .catalog-card::before {
    content: "";
    position: absolute;
    left: -1px;
    top: 1.15rem;
    width: 4px;
    height: 2.3rem;
    background: var(--accent-2);
    border-radius: 0 2px 2px 0;
  }
  .folio-summary {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    padding: 0;
  }
  .folio-summary::before {
    display: none;
  }
  .folio-summary div {
    display: grid;
    gap: 0.4rem;
    border-right: 1px solid var(--border);
    padding: 1.25rem;
  }
  .folio-summary div:last-child {
    border-right: 0;
  }
  .folio-summary span {
    color: var(--text-muted);
    font-family: "Courier New", "Consolas", ui-monospace, monospace;
    font-size: 0.68rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .folio-summary strong {
    font-family: "Rockwell", "Roboto Slab", Georgia, serif;
    font-size: 1.4rem;
    font-weight: 700;
  }
  .folio-core header,
  .folio-ledger header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .folio-core header > span,
  .folio-ledger header > span {
    color: var(--text-muted);
    font-family: "Courier New", "Consolas", ui-monospace, monospace;
    font-size: 0.68rem;
    text-align: right;
  }
  .core-log {
    display: flex;
    gap: 1rem;
    height: 18rem;
    margin-top: 1.4rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    overflow: hidden;
  }
  .core-strip {
    display: flex;
    flex-direction: column;
    width: 1.9rem;
    flex-shrink: 0;
  }
  .core-band {
    flex-shrink: 0;
    border: 0;
    border-top: 1px solid var(--bg);
    padding: 0;
    cursor: pointer;
    opacity: 0.75;
  }
  .core-band:first-child {
    border-top: 0;
  }
  .core-band:hover,
  .core-band.active {
    opacity: 1;
    box-shadow: inset 0 0 0 2px var(--accent);
  }
  .core-legend {
    display: flex;
    flex: 1;
    min-width: 0;
    flex-direction: column;
    overflow-y: auto;
  }
  .core-row {
    display: flex;
    flex-shrink: 0;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    min-height: 1.55rem;
    overflow: hidden;
    border: 0;
    border-top: 1px solid var(--border);
    padding: 0 0.9rem 0 0.25rem;
    background: transparent;
    color: inherit;
    font: inherit;
    text-align: left;
    cursor: pointer;
  }
  .core-row:first-child {
    border-top: 0;
  }
  .core-row:hover,
  .core-row.active {
    background: color-mix(in srgb, var(--accent) 10%, transparent);
  }
  .core-row strong {
    overflow: hidden;
    font-family: "Courier New", "Consolas", ui-monospace, monospace;
    font-size: 0.76rem;
    font-weight: 400;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .core-row span {
    overflow: hidden;
    color: var(--text-muted);
    font-family: "Courier New", "Consolas", ui-monospace, monospace;
    font-size: 0.68rem;
    text-align: right;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .folio-empty {
    margin-top: 1.4rem;
    color: var(--text-muted);
  }
  .folio-note strong {
    display: block;
    font-family: "Rockwell", "Roboto Slab", Georgia, serif;
    font-size: 1.5rem;
    font-weight: 700;
  }
  .folio-note span {
    display: block;
    margin-top: 0.35rem;
    color: var(--text-muted);
    font-family: "Courier New", "Consolas", ui-monospace, monospace;
    font-size: 0.72rem;
  }
  .folio-ledger {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
    padding: 1.5rem;
  }
  .ledger-scroll {
    overflow-x: auto;
    margin-top: 1rem;
  }
  table {
    width: 100%;
    min-width: 38rem;
    border-collapse: collapse;
    font-family: "Courier New", "Consolas", ui-monospace, monospace;
    font-size: 0.76rem;
  }
  th {
    color: var(--text-muted);
    font-size: 0.62rem;
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
  .folio-source {
    border-top: 1px solid var(--border);
    padding-top: 0.8rem;
    color: var(--text-muted);
    font-family: "Courier New", "Consolas", ui-monospace, monospace;
    font-size: 0.66rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  @media (max-width: 680px) {
    .folio-head,
    .folio-core header,
    .folio-ledger header {
      display: block;
    }
    .folio-readout {
      display: block;
      margin-top: 1.5rem;
    }
    .folio-core header > span,
    .folio-ledger header > span {
      display: block;
      margin-top: 0.8rem;
    }
    .folio-summary {
      grid-template-columns: 1fr;
    }
    .folio-summary div {
      border-right: 0;
      border-bottom: 1px solid var(--border);
    }
    .folio-summary div:last-child {
      border-bottom: 0;
    }
  }
</style>
