<script lang="ts">
  type Disp = {
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
    chrono: Disp[];
    gran: "day" | "month" | "year";
    onGran: (value: "day" | "month" | "year") => void;
  } = $props();

  const maxSteps = $derived(
    Math.max(1, ...chrono.map((item) => item.steps ?? 0)),
  );
  const latest = $derived(chrono.at(-1));
</script>

<section class="grapher-daily" aria-labelledby="daily-data-title">
  <header class="page-intro">
    <p class="kicker">Patterns / time series</p>
    <h1 id="daily-data-title">How does the pattern move?</h1>
    <p>
      Choose a time scale, compare the signals, and inspect the underlying
      periods.
    </p>
  </header>

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

  <section class="series-panel" aria-labelledby="steps-series-title">
    <div class="panel-heading">
      <div>
        <p class="kicker">Primary series</p>
        <h2 id="steps-series-title">Steps over time</h2>
      </div>
      {#if latest}<strong>{latest.label}</strong>{/if}
    </div>
    <div
      class="series-chart"
      role="img"
      aria-label="Steps by selected time period"
    >
      {#each chrono as item}
        <div
          class="series-column"
          title={`${item.label}: ${item.steps ?? "no value"} steps`}
        >
          <i
            style={`height: ${Math.max(2, ((item.steps ?? 0) / maxSteps) * 100)}%`}
          ></i>
          <small>{item.label}</small>
        </div>
      {/each}
    </div>
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
              ><td>{item.label}</td><td>{item.steps ?? "—"}</td><td
                >{item.distance ?? "—"}</td
              ><td>{item.resting_hr ?? "—"}</td><td>{item.hrv_sdnn ?? "—"}</td
              ><td
                >{item.moveClosedPct == null
                  ? "—"
                  : `${item.moveClosedPct}%`}</td
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
  .series-chart {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(0.7rem, 1fr));
    align-items: end;
    gap: 0.4rem;
    height: 18rem;
    margin-top: 2rem;
    padding-top: 1rem;
    border-bottom: 1px solid var(--border);
    background: repeating-linear-gradient(
      to top,
      transparent 0 3rem,
      color-mix(in srgb, var(--border) 65%, transparent) 3rem 3.05rem
    );
  }
  .series-column {
    display: grid;
    grid-template-rows: 1fr auto;
    align-items: end;
    min-width: 0;
    height: 100%;
  }
  .series-column i {
    display: block;
    width: 70%;
    min-height: 0.2rem;
    margin: 0 auto;
    background: var(--accent);
  }
  .series-column small {
    overflow: hidden;
    margin-top: 0.5rem;
    color: var(--text-muted);
    font-size: 0.56rem;
    text-align: center;
    text-overflow: ellipsis;
    white-space: nowrap;
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
