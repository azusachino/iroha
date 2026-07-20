<script lang="ts">
  export type DailyVariant = "atlas" | "phenology" | "sound-map" | "archive";
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
    variant,
    chrono,
    gran,
    onGran,
  }: {
    variant: DailyVariant;
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
</script>

<section
  class={`theme-daily theme-daily-${variant}`}
  aria-labelledby="theme-daily-title"
>
  <header class="daily-head">
    <div>
      <p class="theme-kicker">Patterns / {gran} interval</p>
      <h1 id="theme-daily-title">A longer signal.</h1>
      <p>
        Compare the periods without flattening the story into a single score.
      </p>
    </div>
    <div class="daily-count">
      <strong>{chrono.length}</strong><span>periods</span>
    </div>
  </header>

  <nav class="daily-tabs" aria-label="Aggregation interval">
    {#each ["day", "month", "year"] as option}
      <button
        class:active={gran === option}
        type="button"
        onclick={() => onGran(option as "day" | "month" | "year")}
        >{option}</button
      >
    {/each}
  </nav>

  <section class="daily-chart" aria-labelledby="daily-chart-title">
    <header>
      <div>
        <p class="theme-kicker">Primary series</p>
        <h2 id="daily-chart-title">Steps by period</h2>
      </div>
      {#if latest}<span>{latest.label} · {number(latest.steps)} steps</span
        >{/if}
    </header>
    <div class="daily-bars" role="img" aria-label="Steps across periods">
      {#each chrono as period}
        <div
          class="daily-bar"
          title={`${period.label}: ${number(period.steps)} steps`}
        >
          <i
            style={`height: ${Math.max(3, ((period.steps ?? 0) / max) * 100)}%`}
          ></i><small>{period.label}</small>
        </div>
      {/each}
    </div>
  </section>

  <div class="daily-notes">
    <article>
      <p class="theme-kicker">Latest period</p>
      <strong>{latest?.label ?? "—"}</strong><span
        >{number(latest?.distance, 1)} km · {number(latest?.resting_hr)} bpm</span
      >
    </article>
    <article>
      <p class="theme-kicker">Movement closure</p>
      <strong
        >{latest?.moveClosedPct == null
          ? "—"
          : `${latest.moveClosedPct}%`}</strong
      ><span>move goal recorded</span>
    </article>
    <article>
      <p class="theme-kicker">Recovery trace</p>
      <strong>{number(latest?.hrv_sdnn)} ms</strong><span>latest HRV value</span
      >
    </article>
  </div>

  <section class="daily-ledger">
    <header>
      <div>
        <p class="theme-kicker">Period ledger</p>
        <h2>Keep the detail.</h2>
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
  <footer class="daily-source">
    Source: daily records and aggregates · presentation only
  </footer>
</section>

<style>
  .theme-daily {
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
    font-size: clamp(2.8rem, 7vw, 6rem);
    line-height: 0.9;
  }
  h2 {
    font-size: 1.7rem;
  }
  .daily-head {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    padding-bottom: 2rem;
    border-bottom: 1px solid var(--border);
  }
  .daily-head p:last-child {
    color: var(--text-muted);
    line-height: 1.6;
  }
  .daily-count {
    display: grid;
    justify-items: end;
    color: var(--text-muted);
  }
  .daily-count strong {
    color: var(--accent);
    font-family: Georgia, serif;
    font-size: 3.6rem;
    font-weight: 400;
    line-height: 0.8;
  }
  .daily-count span {
    margin-top: 0.6rem;
    font-size: 0.65rem;
    text-transform: uppercase;
  }
  .daily-tabs {
    display: flex;
    gap: 0.4rem;
  }
  .daily-tabs button {
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 0.45rem 0.85rem;
    background: transparent;
    color: var(--text-muted);
    font: inherit;
    font-size: 0.75rem;
    cursor: pointer;
  }
  .daily-tabs button.active,
  .daily-tabs button:hover {
    border-color: var(--accent);
    color: var(--accent);
  }
  .daily-chart,
  .daily-ledger,
  .daily-notes article {
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface-1) 88%, transparent);
  }
  .daily-chart,
  .daily-ledger {
    padding: 1.5rem;
  }
  .daily-chart header,
  .daily-ledger header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .daily-chart header > span,
  .daily-ledger header > span {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .daily-bars {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(0.8rem, 1fr));
    align-items: end;
    gap: 0.4rem;
    height: 18rem;
    margin-top: 1.5rem;
    border-bottom: 1px solid var(--border);
    background: repeating-linear-gradient(
      to top,
      transparent 0 3rem,
      color-mix(in srgb, var(--border) 65%, transparent) 3rem 3.05rem
    );
  }
  .daily-bar {
    display: grid;
    grid-template-rows: 1fr auto;
    align-items: end;
    height: 100%;
    min-width: 0;
  }
  .daily-bar i {
    display: block;
    width: 70%;
    min-height: 0.2rem;
    margin: 0 auto;
    background: var(--accent);
  }
  .daily-bar small {
    overflow: hidden;
    margin-top: 0.5rem;
    color: var(--text-muted);
    font-size: 0.55rem;
    text-align: center;
    text-overflow: ellipsis;
    white-space: nowrap;
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
    font-family: Georgia, serif;
    font-size: 1.7rem;
    font-weight: 400;
  }
  .daily-notes span {
    margin-top: 0.5rem;
    color: var(--text-muted);
    font-size: 0.75rem;
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
    font-family: Georgia, serif;
  }
  .daily-source {
    border-top: 1px solid var(--border);
    padding-top: 0.8rem;
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  .theme-daily-atlas {
    font-family: "Avenir Next", sans-serif;
  }
  .theme-daily-atlas .daily-chart {
    border-top: 0.35rem solid var(--accent);
  }
  .theme-daily-phenology .daily-notes article {
    border-radius: 1.1rem 0.3rem;
  }
  .theme-daily-phenology h1,
  .theme-daily-phenology h2 {
    font-style: italic;
  }
  .theme-daily-sound-map {
    font-family: "IBM Plex Mono", monospace;
  }
  .theme-daily-sound-map h1,
  .theme-daily-sound-map h2 {
    font-family: inherit;
  }
  .theme-daily-sound-map .daily-bars {
    background: repeating-linear-gradient(
      90deg,
      transparent 0 1rem,
      color-mix(in srgb, var(--accent) 7%, transparent) 1rem 1.05rem
    );
  }
  .theme-daily-archive .daily-ledger {
    box-shadow: 0.45rem 0.45rem 0
      color-mix(in srgb, var(--accent) 18%, transparent);
  }
  @media (max-width: 680px) {
    .daily-head,
    .daily-chart header,
    .daily-ledger header {
      display: block;
    }
    .daily-count {
      display: flex;
      align-items: baseline;
      justify-items: initial;
      gap: 0.5rem;
      margin-top: 1.5rem;
    }
    .daily-count strong {
      font-size: 2.5rem;
    }
    .daily-chart header > span,
    .daily-ledger header > span {
      display: block;
      margin-top: 0.8rem;
    }
    .daily-notes {
      grid-template-columns: 1fr;
    }
  }
</style>
