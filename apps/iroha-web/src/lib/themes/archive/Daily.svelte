<script lang="ts">
  type Period = {
    label: string;
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
  }: {
    chrono: Period[];
    gran: "day" | "month" | "year";
    onGran: (value: "day" | "month" | "year") => void;
  } = $props();

  const max = $derived(
    Math.max(1, ...chrono.map((period) => period.steps ?? 0)),
  );
  const latest = $derived(chrono.at(-1));

  function number(value: number | null | undefined, digits = 0): string {
    if (typeof value !== "number" || !Number.isFinite(value)) return "—";
    return value.toLocaleString(undefined, {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
    });
  }

  function tone(pct: number | null | undefined): string {
    if (pct == null || !Number.isFinite(pct))
      return "color-mix(in srgb, var(--border) 55%, var(--surface))";
    const clamped = Math.max(0, Math.min(100, pct));
    return `color-mix(in srgb, var(--accent-2) ${clamped}%, var(--accent) ${100 - clamped}%)`;
  }

  // The primary device: a core log. Each period settles into the record as
  // one stratum -- thickness is real step volume, tone is real move-goal
  // closure -- read newest-first from the top of the column down, the way a
  // registrar reads the most recent accession before working back through
  // the shelf.
  const rows = $derived(
    chrono
      .slice(-30)
      .reverse()
      .map((period, index) => ({
        key: `${period.label}-${index}`,
        period,
        magnitude: Math.max((period.steps ?? 0) / max, 0.04),
        tone: tone(period.moveClosedPct),
      })),
  );
</script>

<section class="folio-daily" aria-labelledby="folio-daily-title">
  <header class="folio-head">
    <div>
      <p class="folio-kicker">Finding aid / {gran} interval</p>
      <h1 id="folio-daily-title">A record of accumulation.</h1>
      <p>
        Each period settles into the record like a layer in a core sample --
        compare without flattening the story into a single score.
      </p>
    </div>
    <div class="folio-readout">
      <strong>{chrono.length}</strong><span>periods catalogued</span>
    </div>
  </header>

  <nav class="folio-tabs" aria-label="Aggregation interval">
    {#each ["day", "month", "year"] as option}
      <button
        class:active={gran === option}
        type="button"
        onclick={() => onGran(option as "day" | "month" | "year")}
        >{option}</button
      >
    {/each}
  </nav>

  <section class="folio-core catalog-card" aria-labelledby="folio-core-title">
    <header>
      <div>
        <p class="folio-kicker">Primary series</p>
        <h2 id="folio-core-title">Steps, layered by period</h2>
      </div>
      {#if latest}<span>{latest.label} · {number(latest.steps)} steps</span
        >{/if}
    </header>
    {#if rows.length}
      <div
        class="core-log"
        role="img"
        aria-label="Steps across periods, most recent at top"
      >
        <div class="core-strip">
          {#each rows as row (row.key)}
            <div
              class="core-band"
              style={`flex-grow: ${row.magnitude}; background: ${row.tone};`}
              title={`${row.period.label}: ${number(row.period.steps)} steps`}
            ></div>
          {/each}
        </div>
        <div class="core-legend">
          {#each rows as row (row.key)}
            <div class="core-row" style={`flex-grow: ${row.magnitude};`}>
              <strong>{row.period.label}</strong>
              <span
                >{number(row.period.steps)} steps · {row.period.moveClosedPct ==
                null
                  ? "— move"
                  : `${row.period.moveClosedPct}% move`}</span
              >
            </div>
          {/each}
        </div>
      </div>
      <p class="core-legend-note">
        Band tone reads violet-to-ochre with move-goal closure; band thickness
        is real step volume for the period.
      </p>
    {:else}
      <p class="folio-empty">No periods available for this interval.</p>
    {/if}
  </section>

  <div class="folio-notes">
    <article class="catalog-card">
      <p class="folio-kicker">Latest period</p>
      <strong>{latest?.label ?? "—"}</strong><span
        >{number(latest?.distance, 1)} km · {number(latest?.resting_hr)} bpm</span
      >
    </article>
    <article class="catalog-card">
      <p class="folio-kicker">Movement closure</p>
      <strong
        >{latest?.moveClosedPct == null
          ? "—"
          : `${latest.moveClosedPct}%`}</strong
      ><span>move goal recorded</span>
    </article>
    <article class="catalog-card">
      <p class="folio-kicker">Recovery trace</p>
      <strong>{number(latest?.hrv_sdnn)} ms</strong><span>latest HRV value</span
      >
    </article>
  </div>

  <section class="folio-ledger">
    <header>
      <div>
        <p class="folio-kicker">Period register</p>
        <h2>Keep the detail.</h2>
      </div>
      <span>— means no source value</span>
    </header>
    <div class="ledger-scroll">
      <table>
        <thead
          ><tr
            ><th>Fol.</th><th>Period</th><th>Steps</th><th>Distance</th><th
              >Move</th
            ><th>Resting HR</th><th>HRV</th></tr
          ></thead
        ><tbody>
          {#each [...chrono].reverse() as period, index (period.label + index)}
            <tr
              ><td class="folio-index"
                >{String(chrono.length - index).padStart(3, "0")}</td
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
  <footer class="folio-source">
    Source: daily records and aggregates · presentation only
  </footer>
</section>

<style>
  .folio-daily {
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
    padding-bottom: 1.75rem;
    border-bottom: 1px solid var(--border);
  }
  .folio-head p:last-child {
    max-width: 38rem;
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
    font-size: 3.4rem;
    font-weight: 700;
    line-height: 0.85;
  }
  .folio-readout span {
    margin-top: 0.6rem;
    font-family: "Courier New", "Consolas", ui-monospace, monospace;
    font-size: 0.64rem;
    text-transform: uppercase;
  }
  .folio-tabs {
    display: flex;
    gap: 0.4rem;
  }
  .folio-tabs button {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 0.45rem 0.9rem;
    background: transparent;
    color: var(--text-muted);
    font: inherit;
    font-family: "Courier New", "Consolas", ui-monospace, monospace;
    font-size: 0.72rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    cursor: pointer;
  }
  .folio-tabs button.active,
  .folio-tabs button:hover {
    border-color: var(--accent);
    color: var(--accent);
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
    font-size: 0.7rem;
  }
  .core-log {
    display: flex;
    gap: 1rem;
    height: 19rem;
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
    border-top: 1px solid var(--bg);
  }
  .core-band:first-child {
    border-top: 0;
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
    min-height: 1.6rem;
    overflow: hidden;
    border-top: 1px solid var(--border);
    padding: 0 0.9rem 0 0.25rem;
  }
  .core-row:first-child {
    border-top: 0;
  }
  .core-row strong {
    overflow: hidden;
    color: var(--text);
    font-family: "Rockwell", "Roboto Slab", Georgia, serif;
    font-size: 0.9rem;
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
  .core-legend-note {
    margin: 0.9rem 0 0;
    color: var(--text-muted);
    font-size: 0.72rem;
    line-height: 1.5;
  }
  .folio-empty {
    margin-top: 1.4rem;
    color: var(--text-muted);
  }
  .folio-notes {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1rem;
  }
  .folio-notes article {
    min-height: 8rem;
  }
  .folio-notes strong,
  .folio-notes span {
    display: block;
  }
  .folio-notes strong {
    font-family: "Rockwell", "Roboto Slab", Georgia, serif;
    font-size: 1.7rem;
    font-weight: 700;
  }
  .folio-notes span {
    margin-top: 0.5rem;
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
    padding: 0.75rem 0.5rem;
    text-align: left;
    white-space: nowrap;
  }
  .folio-index {
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
      display: flex;
      align-items: baseline;
      justify-items: initial;
      gap: 0.5rem;
      margin-top: 1.5rem;
    }
    .folio-readout strong {
      font-size: 2.5rem;
    }
    .folio-core header > span,
    .folio-ledger header > span {
      display: block;
      margin-top: 0.8rem;
    }
    .folio-notes {
      grid-template-columns: 1fr;
    }
  }
</style>
