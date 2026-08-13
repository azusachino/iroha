<script lang="ts">
  import type { Snippet } from "svelte";
  import type { SleepAggregateBucket, SleepSession } from "$lib/api";
  import BarChart from "$lib/components/BarChart.svelte";
  import SleepAggregateChart from "$lib/components/SleepAggregateChart.svelte";
  import { formatDateOnly, formatDuration } from "$lib/format";

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
    children,
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
    children?: Snippet;
  } = $props();

  const chartSessions = $derived([...sessions].reverse());
</script>

<section class="grapher-sleep" aria-labelledby="sleep-data-title">
  <header class="page-intro">
    <p class="kicker">Night data / recovery series</p>
    <h1 id="sleep-data-title">How did the night unfold?</h1>
    <p>
      Compare recorded nights as a time series, then inspect one session without
      turning it into a score.
    </p>
  </header>

  {@render children?.()}

  <div class="summary-row" aria-label="Sleep summary">
    <div>
      <span>Recorded sessions</span><strong
        >{sleepSummary?.session_count ?? sessions.length}</strong
      >
    </div>
    <div>
      <span>Average asleep</span><strong>{formatDuration(averageAsleep)}</strong
      >
    </div>
    <div>
      <span>Average efficiency</span><strong
        >{Math.round(averageEfficiency * 100)}%</strong
      >
    </div>
  </div>

  {#if rollupBuckets.length}
    <SleepAggregateChart
      buckets={rollupBuckets}
      granularity={rollupGranularity}
      scope={rollupScope}
    />
  {:else}<section class="sleep-series" aria-labelledby="sleep-series-title">
      <div class="panel-heading">
        <div>
          <p class="kicker">Observed sessions</p>
          <h2 id="sleep-series-title">Asleep time by night</h2>
        </div>
        <span>Newest first</span>
      </div>
      <BarChart
        categories={chartSessions.map((session) =>
          formatDateOnly(session.wake_date),
        )}
        primary={{
          name: "Asleep",
          values: chartSessions.map((session) => session.asleep_s),
          colors: chartSessions.map((session) =>
            session.is_main_sleep ? "var(--accent)" : "var(--accent-2)",
          ),
          formatter: (value) => formatDuration(value),
        }}
        onBarClick={(index) => {
          const session = chartSessions[index];
          if (session) onSelect(session);
        }}
      />
    </section>{/if}

  <section class="session-table" aria-labelledby="sleep-table-title">
    <div class="panel-heading">
      <div>
        <p class="kicker">Session records</p>
        <h2 id="sleep-table-title">Night by night</h2>
      </div>
      <span
        >{rollupBuckets.length
          ? "Recent loaded records"
          : "Imported values"}</span
      >
    </div>
    <div class="table-scroll">
      <table>
        <thead
          ><tr
            ><th>Date</th><th>Asleep</th><th>In bed</th><th>Efficiency</th><th
              >Type</th
            ></tr
          ></thead
        ><tbody>
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
              ><td>{formatDateOnly(session.wake_date)}</td><td
                >{formatDuration(session.asleep_s)}</td
              ><td>{formatDuration(session.time_in_bed_s)}</td><td
                >{Math.round(session.efficiency * 100)}%</td
              ><td>{session.is_main_sleep ? "Main sleep" : "Nap"}</td></tr
            >
          {/each}
        </tbody>
      </table>
    </div>
  </section>

  {#if selected}
    <aside class="selected-note">
      <p class="kicker">Selected session</p>
      <strong
        >{formatDateOnly(selected.wake_date)} · {formatDuration(
          selected.asleep_s,
        )} asleep</strong
      ><span>Open the standard detail view for stage-level inspection.</span>
    </aside>
  {/if}
</section>

<style>
  .grapher-sleep {
    display: grid;
    gap: 1rem;
  }
  .page-intro {
    max-width: 50rem;
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
    line-height: 1.55;
  }
  .summary-row {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    border-block: 1px solid var(--border);
  }
  .summary-row div {
    display: grid;
    gap: 0.4rem;
    padding: 1rem;
    border-right: 1px solid var(--border);
  }
  .summary-row div:last-child {
    border: 0;
  }
  .summary-row span {
    color: var(--text-muted);
    font-size: 0.66rem;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .summary-row strong {
    font-size: clamp(1.35rem, 3vw, 2.5rem);
    letter-spacing: -0.08em;
  }
  .sleep-series,
  .session-table {
    padding: 1.25rem;
    border: 1px solid var(--border);
    background: var(--surface);
  }
  .panel-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .panel-heading > span {
    color: var(--text-muted);
    font-size: 0.72rem;
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
  tbody tr {
    cursor: pointer;
  }
  tbody tr:hover,
  tbody tr.selected {
    background: var(--surface-2);
  }
  .selected-note {
    display: grid;
    gap: 0.3rem;
    padding: 1rem;
    border-left: 3px solid var(--accent);
    background: var(--surface-2);
  }
  .selected-note strong {
    font-size: 1rem;
  }
  .selected-note span {
    color: var(--text-muted);
    font-size: 0.75rem;
  }
</style>
