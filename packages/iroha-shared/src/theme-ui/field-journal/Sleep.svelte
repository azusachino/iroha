<script lang="ts">
  import type { SleepThemeProps } from "../../sleep-view";
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

  const chartSessions = $derived([...sessions].reverse());
  const activeChartIndex = $derived(
    selected
      ? chartSessions.findIndex((session) => session.id === selected.id)
      : null,
  );
</script>

<section class="journal-night" aria-labelledby="journal-night-title">
  <header class="night-opening">
    <div>
      <p class="journal-kicker">Night notes · recovery record</p>
      <h1 id="journal-night-title">What the night recorded.</h1>
      <p>Rest kept as a sequence of nights, not a single verdict.</p>
    </div>
    <div class="night-stamp" aria-label="Recorded nights">
      <strong>{sleepSummary?.session_count ?? sessions.length}</strong>
      <span>sessions</span>
    </div>
  </header>

  {@render children?.()}

  <div class="journal-rule"><span>small signals</span></div>

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
  {:else}<section class="night-card">
      <div class="night-heading">
        <div>
          <p class="journal-kicker">Observed nights</p>
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
    </section>{/if}

  {#if selected}
    <aside class="margin-note">
      <p class="journal-kicker">Latest margin</p>
      <h2>{formatDuration(selected.asleep_s)} asleep</h2>
      <p>
        {selected.is_main_sleep ? "Primary overnight sleep" : "Short session"} ·
        {formatDateOnly(selected.wake_date)}
      </p>
    </aside>
  {/if}

  <section class="night-ledger">
    <header>
      <div>
        <p class="journal-kicker">Session ledger</p>
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

  <footer class="journal-source">
    <span>Source: imported sleep sessions</span>
    <span>Presentation only · no readiness score inferred</span>
  </footer>
</section>

<style>
  .journal-night {
    display: grid;
    gap: 1.5rem;
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
    margin: 0;
    font-family: var(--font-serif);
    font-weight: 400;
    letter-spacing: -0.04em;
  }
  h1 {
    font-size: clamp(2.6rem, 6vw, 4.8rem);
    line-height: 0.92;
  }
  h2 {
    margin: 0.25rem 0 0.5rem;
    font-size: 1.6rem;
  }
  .night-opening {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
  }
  .night-opening p:last-child {
    max-width: 36rem;
    color: var(--text-muted);
    line-height: 1.7;
  }
  .night-stamp {
    display: grid;
    justify-items: end;
    color: var(--text-muted);
  }
  .night-stamp strong {
    color: var(--accent);
    font-family: var(--font-serif);
    font-size: 3.2rem;
    font-weight: 400;
    line-height: 0.85;
  }
  .night-stamp span {
    margin-top: 0.5rem;
    font-size: 0.68rem;
    letter-spacing: 0.1em;
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
  .night-summary {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    margin: 0;
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface-1) 84%, transparent);
  }
  .night-summary div {
    display: grid;
    gap: 0.4rem;
    border-right: 1px solid var(--border);
    padding: 1.25rem;
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
    font-family: var(--font-serif);
    font-size: 1.35rem;
  }
  .night-card,
  .night-ledger {
    border: 1px solid var(--border);
    padding: 1.5rem;
    background: color-mix(in srgb, var(--surface-1) 84%, transparent);
  }
  .night-heading,
  .night-ledger header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .night-heading > span,
  .night-ledger header > span {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .margin-note {
    border-left: 0.35rem solid var(--accent);
    padding: 1.2rem 1.5rem;
    background: color-mix(in srgb, var(--accent) 8%, var(--surface-1));
  }
  .margin-note p {
    margin: 0;
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .night-ledger {
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
    font-family: var(--font-serif);
  }
  tbody tr {
    cursor: pointer;
  }
  tbody tr.selected td {
    color: var(--accent);
  }
  .journal-source {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding-top: 1rem;
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  @media (max-width: 680px) {
    .night-opening,
    .night-heading,
    .night-ledger header,
    .journal-source {
      display: block;
    }
    .night-stamp {
      align-items: start;
      justify-items: start;
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
    .night-heading > span {
      display: block;
      margin-top: 0.6rem;
    }
  }
</style>
