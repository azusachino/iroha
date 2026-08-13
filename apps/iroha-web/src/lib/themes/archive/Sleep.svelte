<script lang="ts">
  import type { Snippet } from "svelte";
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

  // Each recorded night becomes a row: most recent night on top, older
  // nights settle toward the bottom of the list -- the same reading order
  // as the period core on the Daily page. The chart above reads
  // chronologically (oldest to newest, left to right) instead.
  const rows = $derived(sessions.map((session) => ({ session })));
  const chartSessions = $derived([...sessions].reverse());
  const activeChartIndex = $derived(
    selected
      ? chartSessions.findIndex((session) => session.id === selected.id)
      : null,
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
      <strong>{sleepSummary?.session_count ?? sessions.length}</strong><span
        >sessions held</span
      >
    </div>
  </header>

  {@render children?.()}

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

  {#if rollupBuckets.length}
    <SleepAggregateChart
      buckets={rollupBuckets}
      granularity={rollupGranularity}
      scope={rollupScope}
    />
  {:else}<section class="folio-core catalog-card">
      <header>
        <div>
          <p class="folio-kicker">Observed nights</p>
          <h2>Sleep core</h2>
        </div>
        <span>thickness = asleep · tone = efficiency</span>
      </header>
      {#if rows.length}
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
          orientation="horizontal"
          activeIndex={activeChartIndex}
          onBarClick={(index) => onSelect(chartSessions[index])}
          height={Math.max(220, chartSessions.length * 22)}
        />
        <div class="core-legend">
          {#each rows as row (row.session.id)}
            <button
              type="button"
              class="core-row"
              class:active={selected?.id === row.session.id}
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
      {:else}
        <p class="folio-empty">No sleep sessions were recorded.</p>
      {/if}
    </section>{/if}

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
    font-family: var(--font-mono);
    font-size: 0.64rem;
    letter-spacing: 0.15em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    font-family: var(--font-serif);
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
    font-family: var(--font-serif);
    font-size: 3.3rem;
    font-weight: 700;
    white-space: nowrap;
  }
  .folio-readout span {
    margin-top: 0.5rem;
    font-family: var(--font-mono);
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
    font-family: var(--font-mono);
    font-size: 0.68rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .folio-summary strong {
    font-family: var(--font-serif);
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
    font-family: var(--font-mono);
    font-size: 0.68rem;
    text-align: right;
  }
  .core-legend {
    display: flex;
    flex-direction: column;
    margin-top: 1.4rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    overflow: hidden;
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
    font-family: var(--font-mono);
    font-size: 0.76rem;
    font-weight: 400;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .core-row span {
    overflow: hidden;
    color: var(--text-muted);
    font-family: var(--font-mono);
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
    font-family: var(--font-serif);
    font-size: 1.5rem;
    font-weight: 700;
  }
  .folio-note span {
    display: block;
    margin-top: 0.35rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
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
    font-family: var(--font-mono);
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
    font-family: var(--font-mono);
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
