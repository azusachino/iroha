<script lang="ts">
  import { page } from "$app/state";
  import {
    getSleep,
    getSleepSegments,
    type SleepSegment,
    type SleepSession,
  } from "$lib/api";
  import SleepTimelineChart from "@iroha/shared/theme-ui/components/SleepTimelineChart.svelte";
  import RouteIntro from "$lib/components/RouteIntro.svelte";
  import LoadingBoundary from "$lib/components/LoadingBoundary.svelte";
  import { formatDate, formatDateOnly, formatDuration } from "$lib/format";
  import { sleepStageLabel, sleepStageColor } from "@iroha/shared/sleep-stages";
  import SourceBadge from "@iroha/shared/SourceBadge.svelte";

  let session = $state<SleepSession | null>(null);
  let segments = $state<SleepSegment[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);
  const id = $derived(page.params.id ?? "");

  function segmentDuration(segment: SleepSegment): number {
    const seconds =
      (new Date(segment.ended_at).getTime() -
        new Date(segment.started_at).getTime()) /
      1000;
    return Number.isFinite(seconds) ? Math.max(0, seconds) : 0;
  }

  const stageRows = $derived(
    segments.map((segment, index) => ({
      number: index + 1,
      stage: sleepStageLabel(segment.stage),
      rawStage: segment.stage,
      startedAt: segment.started_at,
      endedAt: segment.ended_at,
      duration: segmentDuration(segment),
    })),
  );
  const observedSeconds = $derived(
    stageRows.reduce((total, row) => total + row.duration, 0),
  );
  const stageTotals = $derived.by(() => {
    const totals = new Map<string, number>();
    for (const row of stageRows) {
      totals.set(row.stage, (totals.get(row.stage) ?? 0) + row.duration);
    }
    return [...totals.entries()]
      .sort((left, right) => right[1] - left[1])
      .map(([stage, duration]) => ({
        stage,
        duration,
        share: observedSeconds > 0 ? (duration / observedSeconds) * 100 : 0,
      }));
  });

  $effect(() => {
    if (!id) return;
    loading = true;
    error = null;
    Promise.all([getSleep(id), getSleepSegments(id)])
      .then(([loadedSession, loadedSegments]) => {
        session = loadedSession;
        segments = loadedSegments;
      })
      .catch((value) => {
        error = value instanceof Error ? value.message : String(value);
      })
      .finally(() => {
        loading = false;
      });
  });
</script>

<svelte:head>
  <title
    >{session
      ? `${formatDateOnly(session.wake_date)} · Night · iroha`
      : "Night detail · iroha"}</title
  >
</svelte:head>

<section class="sleep-detail-shell">
  <RouteIntro
    eyebrow="Night / detail"
    title={session ? formatDateOnly(session.wake_date) : "Night detail"}
    description="A close look at this recorded sleep session, its stage architecture, and every observed interval."
    actionHref="/night"
    actionLabel="Back to Night"
  />

  {#if session || loading}
    <LoadingBoundary
      {loading}
      ready={session != null}
      label="Loading Night detail…"
    >
      {#snippet children()}
        {#if session}
          <section class="detail-grid">
            <article class="tile hero-card">
              <p
                class="eyebrow session-kind"
                class:nap={!session.is_main_sleep}
              >
                {session.is_main_sleep ? "Main sleep" : "Nap"}
              </p>
              <h2>{formatDuration(session.asleep_s)} <span>asleep</span></h2>
              <p class="muted">
                {formatDuration(session.time_in_bed_s)} in bed · {Math.round(
                  session.efficiency * 100,
                )}% efficiency
              </p>
              <div class="metric-grid">
                <div>
                  <span>Started</span><strong
                    >{formatDate(session.started_at)}</strong
                  >
                </div>
                <div>
                  <span>Ended</span><strong
                    >{formatDate(session.ended_at)}</strong
                  >
                </div>
                <div>
                  <span>Deep + REM</span><strong
                    >{formatDuration(session.deep_s + session.rem_s)}</strong
                  >
                </div>
                <div>
                  <span>Source</span><strong
                    ><SourceBadge source={session.source} /></strong
                  >
                </div>
              </div>
            </article>

            <article class="tile architecture-card">
              <div class="section-heading">
                <div>
                  <p class="eyebrow">Stage timeline</p>
                  <h2>Sleep architecture</h2>
                </div>
                <span class="section-note">{segments.length} segments</span>
              </div>
              {#if segments.length > 0}
                <p class="chart-note">
                  One bar, stacked proportionally by how long each stage lasted
                  across the whole session — hover a segment for its exact
                  duration, or read the totals below.
                </p>
                <SleepTimelineChart
                  {segments}
                  sessionKind={session.is_main_sleep ? "main" : "nap"}
                />
              {:else}
                <p class="muted">This session has no stage samples.</p>
              {/if}
            </article>
          </section>

          <section
            class="tile evidence-card"
            aria-labelledby="stage-evidence-title"
          >
            <div class="section-heading">
              <div>
                <p class="eyebrow">Stage evidence</p>
                <h2 id="stage-evidence-title">Every recorded interval</h2>
              </div>
              <span class="section-note"
                >{stageRows.length} intervals · {formatDuration(
                  observedSeconds,
                )} observed</span
              >
            </div>

            {#if stageTotals.length}
              <div class="stage-summary" aria-label="Stage totals">
                {#each stageTotals as item (item.stage)}
                  <div>
                    <span>{item.stage}</span>
                    <strong>{formatDuration(item.duration)}</strong>
                    <small>{Math.round(item.share)}% of observed stages</small>
                  </div>
                {/each}
              </div>
            {/if}

            {#if stageRows.length}
              <div class="segment-table">
                <table>
                  <caption
                    >Canonical sleep stage intervals in source order</caption
                  >
                  <thead>
                    <tr>
                      <th>#</th>
                      <th>Stage</th>
                      <th>Started</th>
                      <th>Ended</th>
                      <th>Duration</th>
                      <th>Share</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each stageRows as row (row.number)}
                      <tr>
                        <td>{row.number}</td>
                        <td>
                          <span
                            class="stage-dot"
                            style={`background: ${sleepStageColor(row.rawStage)}`}
                          ></span>
                          {row.stage}
                        </td>
                        <td>{formatDate(row.startedAt)}</td>
                        <td>{formatDate(row.endedAt)}</td>
                        <td>{formatDuration(row.duration)}</td>
                        <td
                          >{observedSeconds > 0
                            ? `${Math.round((row.duration / observedSeconds) * 100)}%`
                            : "—"}</td
                        >
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
              <p class="evidence-note">
                Intervals are the canonical stage records returned by Iroha. The
                stacked chart summarizes them; this table preserves their order
                and exact canonical timestamps.
              </p>
            {:else}
              <p class="muted">This session has no stage samples.</p>
            {/if}
          </section>
        {/if}
      {/snippet}
    </LoadingBoundary>
  {:else if error}
    <section class="status tile">
      <p class="error">Night could not be loaded: {error}</p>
    </section>
  {/if}
</section>

<style>
  .sleep-detail-shell {
    display: grid;
    gap: 1.25rem;
  }
  .detail-grid {
    display: grid;
    grid-template-columns: minmax(16rem, 0.8fr) minmax(0, 1.2fr);
    gap: 1rem;
  }
  .hero-card,
  .architecture-card,
  .evidence-card {
    padding: clamp(1.15rem, 3vw, 2rem);
  }
  h2 {
    margin: 0.25rem 0 0.5rem;
  }
  h2 span {
    color: var(--text-muted);
    font-size: 0.5em;
    font-weight: 500;
  }
  .metric-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.75rem;
    margin-top: 1.5rem;
  }
  .metric-grid div {
    display: grid;
    gap: 0.2rem;
    padding-top: 0.65rem;
    border-top: 1px solid var(--border);
  }
  .metric-grid span {
    color: var(--text-muted);
    font-size: 0.72rem;
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }
  .metric-grid strong {
    font-size: 0.95rem;
  }
  .metric-grid :global(.source-badge) {
    font-size: 0.75rem;
  }
  .section-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: start;
    margin-bottom: 1rem;
  }
  .section-heading h2 {
    font-size: 1.2rem;
  }
  .chart-note {
    margin: 0 0 0.9rem;
    color: var(--text-muted);
    font-size: 0.82rem;
    line-height: 1.5;
  }
  .stage-summary {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr));
    gap: 0.6rem;
    margin-bottom: 1.15rem;
  }
  .stage-summary div {
    display: grid;
    gap: 0.25rem;
    padding: 0.75rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 3px);
    background: color-mix(in srgb, var(--surface-2) 70%, transparent);
  }
  .stage-summary span,
  .stage-summary small {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .stage-summary strong {
    font-size: 1.05rem;
    font-variant-numeric: tabular-nums;
  }
  .segment-table {
    overflow-x: auto;
  }
  table {
    width: 100%;
    min-width: 48rem;
    border-collapse: collapse;
    font-size: 0.78rem;
  }
  caption {
    margin-bottom: 0.6rem;
    color: var(--text-muted);
    font-size: 0.72rem;
    text-align: left;
  }
  th,
  td {
    border-top: 1px solid var(--border);
    padding: 0.65rem 0.45rem;
    text-align: left;
    white-space: nowrap;
  }
  th {
    color: var(--text-muted);
    font-size: 0.66rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  td {
    font-variant-numeric: tabular-nums;
  }
  td:nth-child(2) {
    font-weight: 700;
  }
  .stage-dot {
    display: inline-block;
    width: 0.55rem;
    height: 0.55rem;
    margin-right: 0.35rem;
    border-radius: 50%;
    background: var(--text-muted);
  }
  .session-kind {
    width: fit-content;
    padding: 0.25rem 0.5rem;
    border: 1px solid color-mix(in srgb, var(--accent) 38%, var(--border));
    border-radius: 999px;
  }
  .session-kind.nap {
    color: #8b6fd1;
    border-color: color-mix(in srgb, #8b6fd1 48%, var(--border));
  }
  .evidence-note {
    margin: 0.85rem 0 0;
    color: var(--text-muted);
    font-size: 0.78rem;
    line-height: 1.5;
  }
  @media (max-width: 720px) {
    .detail-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
