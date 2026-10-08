<script lang="ts">
  let recordsScroll = $state<HTMLDivElement>();
  import type { SleepThemeProps } from "../../view-contracts/sleep-view";
  import BarChart from "../components/BarChart.svelte";
  import RetryNotice from "../components/RetryNotice.svelte";
  import SleepAggregateChart from "../components/SleepAggregateChart.svelte";
  import { formatDateOnly, formatDuration } from "../../format/format";

  let {
    sessions,
    selected,
    sessionsLoading = false,
    sessionsReady = false,
    sessionsError = null,
    recordsScope = "",
    onRetrySessions,
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
  let recordsPanel = $state<HTMLElement>();
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

  <div class="summary-row" role="region" aria-label="Sleep summary">
    <div>
      <span>Recorded sessions</span><strong
        >{sleepSummary?.session_count ?? "—"}</strong
      >
    </div>
    <div>
      <span>Average main sleep</span><strong
        >{averageAsleep == null ? "—" : formatDuration(averageAsleep)}</strong
      >
    </div>
    <div>
      <span>Average efficiency</span><strong
        >{averageEfficiency == null
          ? "—"
          : `${Math.round(averageEfficiency * 100)}%`}</strong
      >
    </div>
  </div>

  {#if rollupBuckets.length}
    <SleepAggregateChart
      buckets={rollupBuckets}
      granularity={rollupGranularity}
      scope={rollupScope}
      {theme}
    />
  {:else if sessions.length}<section
      class="sleep-series"
      aria-labelledby="sleep-series-title"
    >
      <div class="panel-heading">
        <div>
          <p class="kicker">Observed sessions</p>
          <h2 id="sleep-series-title">Asleep time by night</h2>
        </div>
        <span>Observed sessions: {recordsScope}</span>
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
          if (session) onOpenDetail(session);
        }}
      />
    </section>{/if}

  <section
    class="session-table"
    aria-label="Sleep session read state"
    tabindex="-1"
    bind:this={recordsPanel}
  >
    {#if sessionsError && onRetrySessions}<RetryNotice
        message={`Could not load sleep sessions: ${sessionsError}`}
        retryLabel="Retry sleep sessions"
        onRetry={onRetrySessions}
        focusTarget={recordsPanel}
      />{/if}
    {#if sessionsLoading}<p role="status">
        {sessionsReady ? "Updating sleep sessions…" : "Loading sleep sessions…"}
      </p>{/if}
    {#if recordsScope}<p>Observed sessions: {recordsScope}</p>{/if}
    {#if sessionsReady && !sessionsLoading && !sessionsError && !sessions.length}<p
      >
        No sleep sessions in this observed scope.
      </p>{/if}
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
    <button
      class="scroll-hint"
      type="button"
      aria-controls="night-records-scroll"
      onclick={() => recordsScroll?.focus({ preventScroll: true })}
      >Focus sleep records to scroll all columns</button
    >
    <div
      class="table-scroll"
      id="night-records-scroll"
      role="region"
      aria-label="Night exact sleep records"
      tabindex="-1"
      bind:this={recordsScroll}
    >
      <table aria-label="Sleep session records">
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
    --accent: var(--mark-violet);
    display: grid;
    gap: 1rem;
    min-width: 0;
  }
  .grapher-sleep > * {
    min-width: 0;
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
    font-size: var(--grapher-utility-title-size);
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
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 0.75rem;
  }
  .summary-row div {
    display: grid;
    gap: 0.4rem;
    min-width: 0;
    padding: 1rem;
    border: 1px solid var(--border);
    border-radius: var(--radius, 14px);
    background: var(--tile-surface, var(--surface));
    box-shadow: var(--tile-shadow);
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
    border-radius: var(--radius, 14px);
    background: var(--tile-surface, var(--surface));
    box-shadow: var(--tile-shadow);
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
  .scroll-hint {
    min-height: 44px;
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
    background: color-mix(in srgb, var(--accent) 10%, var(--surface));
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
