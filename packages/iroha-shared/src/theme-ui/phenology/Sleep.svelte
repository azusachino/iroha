<script lang="ts">
  import type { SleepThemeProps } from "../../sleep-view";
  import type { SleepSession } from "../../sleep";
  import {
    formatDateOnly,
    formatDateShort,
    formatDuration,
  } from "../../format";
  import BarChart from "../components/BarChart.svelte";
  import SleepAggregateChart from "../components/SleepAggregateChart.svelte";

  let {
    sessions,
    selected,
    averageAsleep,
    averageEfficiency,
    onOpenDetail,
    sleepSummary = null,
    rollupBuckets = [],
    rollupGranularity = "year",
    rollupScope = "",
    theme,
    children,
  }: SleepThemeProps = $props();

  const maxAsleep = $derived(
    Math.max(1, ...sessions.map((session) => session.asleep_s)),
  );

  // Each night is still rendered as a phase disc in the field below (size
  // encodes time asleep, fill encodes efficiency) -- the chart above is the
  // same data as an interactive, hoverable bar+line series instead of a
  // second reading of the same two numbers.
  function discStyle(session: SleepSession): string {
    const size = 1.5 + (session.asleep_s / maxAsleep) * 1.9;
    const sweep = Math.max(0, Math.min(1, session.efficiency)) * 360;
    return `--d: ${size.toFixed(2)}rem; --sweep: ${sweep.toFixed(1)}deg;`;
  }

  const chartSessions = $derived([...sessions].reverse());
  const activeChartIndex = $derived(
    selected
      ? chartSessions.findIndex((session) => session.id === selected.id)
      : null,
  );
</script>

<section class="bloom-night" aria-labelledby="bloom-night-title">
  <header class="night-opening">
    <div>
      <p class="bloom-kicker">● Night record · recovery</p>
      <h1 id="bloom-night-title">What the night returned.</h1>
      <p>Rest read as a sequence of phases, not a single verdict.</p>
    </div>
    <div class="night-count">
      <strong>{sleepSummary?.session_count ?? sessions.length}</strong>
      <span>sessions</span>
    </div>
  </header>

  {@render children?.()}

  <dl class="night-summary">
    <div>
      <dt>Average asleep</dt>
      <dd>{formatDuration(averageAsleep)}</dd>
    </div>
    <div>
      <dt>Average efficiency</dt>
      <dd>{Math.round(averageEfficiency * 100)}%</dd>
    </div>
    <div>
      <dt>Selected</dt>
      <dd>{selected ? formatDateOnly(selected.wake_date) : "—"}</dd>
    </div>
  </dl>

  {#if rollupBuckets.length}
    <SleepAggregateChart
      buckets={rollupBuckets}
      granularity={rollupGranularity}
      scope={rollupScope}
      {theme}
    />
  {:else}<section class="phase-panel">
      <div class="panel-heading">
        <div>
          <p class="bloom-kicker">Nightly readings</p>
          <h2>Asleep time</h2>
        </div>
        <span>select a column to inspect</span>
      </div>
      <BarChart
        categories={chartSessions.map((session) =>
          formatDateShort(session.wake_date),
        )}
        primary={{
          name: "Asleep",
          values: chartSessions.map((session) => session.asleep_s),
          colors: chartSessions.map((session) =>
            session.is_main_sleep ? "var(--accent)" : "var(--accent-2)",
          ),
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
        onBarClick={(index) => onOpenDetail(chartSessions[index])}
      />
    </section>

    <section class="phase-panel">
      <div class="panel-heading">
        <div>
          <p class="bloom-kicker">Observed nights</p>
          <h2>Phase by phase</h2>
        </div>
        <span>size = time asleep · fill = efficiency</span>
      </div>
      <div
        class="phase-field"
        role="img"
        aria-label="Sleep sessions rendered as phase discs, sized by duration and filled by efficiency"
      >
        {#each chartSessions as session (session.id)}
          <button
            class="phase-disc"
            class:active={selected?.id === session.id}
            style={discStyle(session)}
            title={`${formatDateOnly(session.wake_date)} · ${formatDuration(session.asleep_s)} asleep · ${Math.round(session.efficiency * 100)}%`}
            onclick={() => onOpenDetail(session)}
          >
            <i></i>
            <small>{formatDateOnly(session.wake_date)}</small>
          </button>
        {/each}
      </div>
    </section>{/if}

  {#if selected}
    <aside class="margin-note">
      <p class="bloom-kicker">Latest margin</p>
      <h2>{formatDuration(selected.asleep_s)} asleep</h2>
      <p>
        {selected.is_main_sleep ? "Primary overnight sleep" : "Short session"} ·
        {formatDateOnly(selected.wake_date)} · {Math.round(
          selected.efficiency * 100,
        )}% efficient
      </p>
    </aside>
  {/if}

  <section class="bloom-ledger">
    <header>
      <div>
        <p class="bloom-kicker">Session ledger</p>
        <h2>Night by night</h2>
      </div>
      <span
        >{rollupBuckets.length
          ? "recent loaded records"
          : "imported values"}</span
      >
    </header>
    <div class="ledger-scroll">
      <table>
        <thead>
          <tr>
            <th>Date</th>
            <th>Asleep</th>
            <th>In bed</th>
            <th>Efficiency</th>
            <th>Type</th>
          </tr>
        </thead>
        <tbody>
          {#each sessions as session (session.id)}
            <tr
              class:selected={selected?.id === session.id}
              role="link"
              tabindex="0"
              onclick={() => onOpenDetail(session)}
              onkeydown={(event) => {
                if (event.key === "Enter" || event.key === " ") {
                  event.preventDefault();
                  onOpenDetail(session);
                }
              }}
            >
              <td>{formatDateOnly(session.wake_date)}</td>
              <td>{formatDuration(session.asleep_s)}</td>
              <td>{formatDuration(session.time_in_bed_s)}</td>
              <td>{Math.round(session.efficiency * 100)}%</td>
              <td>{session.is_main_sleep ? "Main sleep" : "Nap"}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </section>

  <footer class="bloom-source">
    Source: imported sleep sessions · no readiness score inferred
  </footer>
</section>

<style>
  .bloom-night {
    display: grid;
    gap: 1.5rem;
    font-family: var(--font-serif);
    min-width: 0;
  }
  .bloom-night > * {
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
    font-size: clamp(2.4rem, 6vw, 4.6rem);
    line-height: 0.95;
  }
  h2 {
    font-size: 1.5rem;
  }
  .night-opening {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.75rem;
  }
  .night-opening p:last-child {
    max-width: 34rem;
    color: var(--text-muted);
    line-height: 1.65;
  }
  .night-count {
    display: grid;
    justify-items: end;
    color: var(--text-muted);
  }
  .night-count strong {
    color: var(--accent);
    font-style: italic;
    font-size: 3.2rem;
    font-weight: 400;
  }
  .night-count span {
    margin-top: 0.4rem;
    font-size: 0.68rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .night-summary {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    margin: 0;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
  }
  .night-summary div {
    display: grid;
    gap: 0.4rem;
    border-right: 1px solid var(--border);
    padding: 1.2rem;
  }
  .night-summary div:last-child {
    border-right: 0;
  }
  dt {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  dd {
    margin: 0;
    font-style: italic;
    font-size: 1.3rem;
  }
  .phase-panel,
  .bloom-ledger {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 1.5rem;
    background: color-mix(in srgb, var(--surface) 90%, transparent);
  }
  .panel-heading,
  .bloom-ledger header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .panel-heading > span,
  .bloom-ledger header > span {
    color: var(--text-muted);
    font-size: 0.7rem;
    text-align: right;
  }
  .phase-field {
    display: flex;
    flex-wrap: wrap;
    align-items: end;
    gap: 1.1rem;
    min-height: 8rem;
    margin-top: 1.75rem;
    padding-bottom: 0.5rem;
  }
  .phase-disc {
    display: grid;
    justify-items: center;
    gap: 0.45rem;
    border: 0;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
  }
  .phase-disc i {
    display: block;
    width: var(--d);
    height: var(--d);
    border-radius: 50%;
    background: conic-gradient(
      var(--accent) var(--sweep),
      color-mix(in srgb, var(--border) 70%, transparent) 0
    );
    box-shadow: inset 0 0 0 1px var(--border);
    opacity: 0.75;
    transition:
      opacity 0.2s ease,
      box-shadow 0.2s ease;
  }
  .phase-disc.active i,
  .phase-disc:hover i {
    opacity: 1;
    box-shadow: inset 0 0 0 1px var(--accent);
  }
  .phase-disc small {
    font-size: 0.56rem;
  }
  .margin-note {
    border-radius: var(--radius);
    border-left: 0.35rem solid var(--accent);
    padding: 1.2rem 1.5rem;
    background: color-mix(in srgb, var(--accent) 8%, var(--surface));
  }
  .margin-note p {
    margin: 0;
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
  td:first-child {
    color: var(--accent);
    font-style: italic;
  }
  tbody tr {
    cursor: pointer;
  }
  tbody tr.selected td {
    color: var(--accent);
  }
  .bloom-source {
    border-top: 1px solid var(--border);
    padding-top: 0.85rem;
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  @media (max-width: 768px) {
    .night-opening,
    .panel-heading,
    .bloom-ledger header {
      display: block;
    }
    .night-count {
      display: flex;
      justify-items: initial;
      align-items: baseline;
      gap: 0.6rem;
      margin-top: 1.5rem;
    }
    .night-summary {
      grid-template-columns: 1fr;
    }
    .night-summary div {
      border-right: 0;
      border-bottom: 1px solid var(--border);
    }
    .night-summary div:last-child {
      border-bottom: 0;
    }
    .panel-heading > span {
      display: block;
      margin-top: 0.6rem;
    }
  }
</style>
