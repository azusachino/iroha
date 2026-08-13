<script lang="ts">
  import type { SleepAggregateBucket, SleepSession } from "$lib/api";
  import { formatDateOnly, formatDateShort, formatDuration } from "$lib/format";
  import BarChart from "$lib/components/BarChart.svelte";
  import SleepAggregateChart from "$lib/components/SleepAggregateChart.svelte";

  let {
    sessions,
    selected,
    averageAsleep,
    averageEfficiency,
    onSelect,
    onOpenDetail,
    sleepSummary = null,
    rollupBuckets = [],
    rollupGranularity = "year",
    rollupScope = "",
  }: {
    sessions: SleepSession[];
    selected: SleepSession | null;
    averageAsleep: number;
    averageEfficiency: number;
    onSelect: (session: SleepSession) => void;
    onOpenDetail: (session: SleepSession) => void;
    sleepSummary?: SleepAggregateBucket | null;
    rollupBuckets?: SleepAggregateBucket[];
    rollupGranularity?: "month" | "year";
    rollupScope?: string;
  } = $props();

  const avgPct = $derived(Math.round(averageEfficiency * 100));
  const chartSessions = $derived([...sessions].reverse());
  const activeChartIndex = $derived(
    selected
      ? chartSessions.findIndex((session) => session.id === selected.id)
      : null,
  );
</script>

<section class="atlas-nights" aria-labelledby="atlas-nights-title">
  <header class="nights-header">
    <div>
      <p class="atlas-kicker">Recovery survey · overnight bearings</p>
      <h1 id="atlas-nights-title">A chart of the nights.</h1>
      <p class="nights-sub">
        Rest fixed night by night — a sequence of recorded positions, not a
        single reading.
      </p>
    </div>
    <div class="grid-ref">
      <span>{sleepSummary?.session_count ?? sessions.length}</span>
      <small>sessions in scope</small>
    </div>
  </header>

  <div class="nights-summary">
    <div class="atlas-plate">
      <p class="atlas-kicker">Average asleep</p>
      <strong>{formatDuration(averageAsleep)}</strong>
    </div>
    <div class="atlas-plate">
      <p class="atlas-kicker">Average efficiency</p>
      <strong>{avgPct}%</strong>
      <div class="scale-bar">
        <div class="scale-track">
          <i class="scale-fill" style={`width: ${avgPct}%`}></i>
        </div>
      </div>
    </div>
    <div class="atlas-plate">
      <p class="atlas-kicker">Selected fix</p>
      <strong>{selected ? formatDateOnly(selected.wake_date) : "—"}</strong>
    </div>
  </div>

  {#if rollupBuckets.length}
    <SleepAggregateChart
      buckets={rollupBuckets}
      granularity={rollupGranularity}
      scope={rollupScope}
    />
  {:else}<section class="atlas-plate nights-chart">
      <header class="chart-heading">
        <div>
          <p class="atlas-kicker">Nightly readings</p>
          <h2>Asleep time</h2>
        </div>
        <span>select a column to inspect</span>
      </header>
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
        onBarClick={(index) => onSelect(chartSessions[index])}
      />
    </section>{/if}

  {#if selected}<aside class="atlas-plate selected-note">
      <p class="atlas-kicker">Latest margin</p>
      <strong>{formatDuration(selected.asleep_s)} asleep</strong><span
        >{selected.is_main_sleep ? "Primary overnight sleep" : "Short session"} ·
        {formatDateOnly(selected.wake_date)}</span
      >
    </aside>{/if}

  <section class="atlas-plate nights-ledger">
    <header class="ledger-heading">
      <div>
        <p class="atlas-kicker">Night log</p>
        <h2>Every night, indexed.</h2>
      </div>
      <span
        >{rollupBuckets.length
          ? "recent loaded records"
          : "imported values"}</span
      >
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
              role="link"
              tabindex="0"
              onclick={() => onOpenDetail(session)}
              onkeydown={(event) => {
                if (event.key === "Enter" || event.key === " ") {
                  event.preventDefault();
                  onOpenDetail(session);
                }
              }}
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

  <footer class="atlas-source">
    Source: imported sleep sessions · no readiness score inferred
  </footer>
</section>

<style>
  .atlas-nights {
    display: grid;
    gap: 1.5rem;
    font-family: var(--font-sans);
  }
  .atlas-kicker {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.66rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .atlas-kicker::before {
    content: "⌖";
  }
  h1,
  h2 {
    margin: 0;
    font-weight: 600;
    letter-spacing: -0.03em;
  }
  h1 {
    max-width: 12ch;
    font-size: clamp(2.4rem, 6vw, 4.4rem);
    line-height: 1;
  }
  h2 {
    font-size: 1.45rem;
  }
  .nights-header {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.75rem;
  }
  .nights-sub {
    max-width: 40rem;
    margin: 1rem 0 0;
    color: var(--text-muted);
    line-height: 1.65;
  }
  .grid-ref {
    display: grid;
    justify-items: end;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    padding: 0.6rem 0.9rem;
    color: var(--accent);
    font-family: var(--font-mono);
    text-align: right;
  }
  .grid-ref span {
    font-size: 1.6rem;
  }
  .grid-ref small {
    margin-top: 0.2rem;
    color: var(--text-muted);
    font-size: 0.6rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  .atlas-plate {
    position: relative;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface-1) 88%, transparent);
  }
  .atlas-plate::before,
  .atlas-plate::after {
    content: "";
    position: absolute;
    width: 0.7rem;
    height: 0.7rem;
    opacity: 0.7;
  }
  .atlas-plate::before {
    top: -1px;
    left: -1px;
    border-top: 2px solid var(--accent);
    border-left: 2px solid var(--accent);
  }
  .atlas-plate::after {
    right: -1px;
    bottom: -1px;
    border-right: 2px solid var(--accent);
    border-bottom: 2px solid var(--accent);
  }
  .nights-summary {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1rem;
  }
  .nights-summary .atlas-plate {
    padding: 1.25rem;
  }
  .nights-summary strong {
    display: block;
    margin-top: 0.3rem;
    font-family: var(--font-mono);
    font-size: 1.5rem;
    font-weight: 600;
  }
  .scale-bar {
    margin-top: 0.75rem;
  }
  .scale-track {
    height: 0.4rem;
    border: 1px solid var(--border);
    background: repeating-linear-gradient(
      90deg,
      color-mix(in srgb, var(--border) 70%, transparent) 0 1px,
      transparent 1px 10%
    );
  }
  .scale-fill {
    display: block;
    height: 100%;
    background: var(--accent);
  }
  .nights-chart,
  .nights-ledger {
    padding: 1.5rem;
  }
  .chart-heading,
  .ledger-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .chart-heading > span,
  .ledger-heading > span {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.72rem;
  }
  .selected-note {
    display: grid;
    gap: 0.35rem;
    padding: 1.2rem 1.5rem;
    background: color-mix(in srgb, var(--accent) 8%, var(--surface-1));
  }
  .selected-note strong {
    font-family: var(--font-mono);
    font-size: 1.4rem;
    font-weight: 600;
  }
  .selected-note span {
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
    font-family: var(--font-mono);
    font-size: 0.62rem;
    font-weight: 400;
    letter-spacing: 0.06em;
    text-align: left;
    text-transform: uppercase;
  }
  th,
  td {
    border-top: 1px solid var(--border);
    padding: 0.8rem 0.5rem;
    text-align: left;
    white-space: nowrap;
    font-family: var(--font-mono);
  }
  tbody tr {
    cursor: pointer;
  }
  tbody tr.selected td {
    color: var(--accent);
  }
  .atlas-source {
    border-top: 1px solid var(--border);
    padding-top: 0.8rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.66rem;
  }
  @media (max-width: 680px) {
    .nights-header,
    .chart-heading,
    .ledger-heading {
      display: block;
    }
    .grid-ref {
      margin-top: 1.5rem;
      justify-items: start;
      text-align: left;
    }
    .chart-heading > span,
    .ledger-heading > span {
      display: block;
      margin-top: 0.6rem;
    }
    .nights-summary {
      grid-template-columns: 1fr;
    }
  }
</style>
