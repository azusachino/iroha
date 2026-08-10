<script lang="ts">
  import { page } from "$app/state";
  import {
    getSleep,
    getSleepSegments,
    type SleepSegment,
    type SleepSession,
  } from "$lib/api";
  import SleepTimelineChart from "$lib/components/SleepTimelineChart.svelte";
  import RouteIntro from "$lib/components/RouteIntro.svelte";
  import { formatDateOnly, formatDuration } from "$lib/format";

  let session = $state<SleepSession | null>(null);
  let segments = $state<SleepSegment[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);
  const id = $derived(page.params.id ?? "");

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
      ? `${formatDateOnly(session.wake_date)} sleep · iroha`
      : "Sleep detail · iroha"}</title
  >
</svelte:head>

<section class="sleep-detail-shell">
  <RouteIntro
    eyebrow="Night / detail"
    title={session ? formatDateOnly(session.wake_date) : "Sleep detail"}
    description="A close look at this sleep session and its stage architecture."
    actionHref="/sleep"
    actionLabel="Back to sleep"
  />

  {#if loading}
    <section class="status tile"><p>Loading sleep detail…</p></section>
  {:else if error}
    <section class="status tile">
      <p class="error">Sleep could not be loaded: {error}</p>
    </section>
  {:else if session}
    <section class="detail-grid">
      <article class="tile hero-card">
        <p class="eyebrow">
          {session.is_main_sleep ? "Primary overnight sleep" : "Short session"}
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
              >{new Date(session.started_at).toLocaleTimeString([], {
                hour: "2-digit",
                minute: "2-digit",
              })}</strong
            >
          </div>
          <div>
            <span>Ended</span><strong
              >{new Date(session.ended_at).toLocaleTimeString([], {
                hour: "2-digit",
                minute: "2-digit",
              })}</strong
            >
          </div>
          <div>
            <span>Deep + REM</span><strong
              >{formatDuration(session.deep_s + session.rem_s)}</strong
            >
          </div>
          <div>
            <span>Source</span><strong
              >{session.source || "Apple Health"}</strong
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
            One bar, stacked proportionally by how long each stage lasted across
            the whole session — hover a segment for its exact duration, or read
            the totals below.
          </p>
          <SleepTimelineChart {segments} />
        {:else}
          <p class="muted">This session has no stage samples.</p>
        {/if}
      </article>
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
  .architecture-card {
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
  @media (max-width: 720px) {
    .detail-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
