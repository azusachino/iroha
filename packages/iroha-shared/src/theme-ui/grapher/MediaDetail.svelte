<script lang="ts">
  import type { MediaDetailThemeProps } from "../../media";
  import BarChart from "../components/BarChart.svelte";
  import { mediaEventLabel } from "../../media";

  let { detail, progress, theme }: MediaDetailThemeProps = $props();
  const progressEvents = $derived(
    detail.events.filter((event) => event.progress_percent != null),
  );
</script>

<article
  class="grapher-media-detail"
  data-theme={theme}
  aria-labelledby="grapher-media-detail-title"
>
  <header class="detail-header">
    <div>
      <p class="kicker">Library / item record</p>
      <h1 id="grapher-media-detail-title">
        {detail.item.native_title || detail.item.title}
      </h1>
      <p>
        {detail.item.media_type.replaceAll("_", " ")} · {detail.item.status ||
          "untracked"}
      </p>
    </div>
    {#if detail.item.rating != null}<strong class="rating"
        >{detail.item.rating.toFixed(1)}<small>/10</small></strong
      >{/if}
  </header>
  {#if detail.item.cover_image_url}<img
      class="cover"
      src={detail.item.cover_image_url}
      alt=""
    />{/if}
  <section class="chart-panel">
    <header>
      <p class="kicker">Continuity</p>
      <h2>Progress through the record</h2>
      <strong>{Math.round(progress)}%</strong>
    </header>
    <div class="progress">
      <span style={`width:${Math.min(Math.max(progress, 0), 100)}%`}></span>
    </div>
    <p class="muted">
      {detail.progress?.position ?? 0}{detail.progress?.total != null
        ? ` / ${detail.progress.total}`
        : ""}
      {detail.progress?.unit || ""}
    </p>
  </section>
  {#if progressEvents.length}<section class="chart-panel">
      <header>
        <p class="kicker">Comparison over events</p>
        <h2>Progress history</h2>
      </header>
      <BarChart
        categories={progressEvents.map(
          (event, index) =>
            event.event_at?.slice(0, 10) || `Event ${index + 1}`,
        )}
        primary={{
          name: "Complete",
          values: progressEvents.map((event) => event.progress_percent ?? 0),
          color: "var(--accent)",
          formatter: (value) => `${Math.round(value)}%`,
        }}
        primaryType="line"
        height={260}
      />
    </section>{/if}
  <div class="record-grid">
    <section>
      <header>
        <p class="kicker">Event log</p>
        <h2>{detail.events.length} recorded events</h2>
      </header>
      {#if detail.events.length}<ol>
          {#each detail.events.slice(0, 20) as event (event.id)}<li>
              <time>{event.event_at?.slice(0, 10) || "Undated"}</time><span
                >{mediaEventLabel(event.event_type)}</span
              >{#if event.progress_percent != null}<strong
                  >{Math.round(event.progress_percent)}%</strong
                >{/if}
            </li>{/each}
        </ol>{:else}<p class="muted">No event history recorded.</p>{/if}
    </section>
    <aside>
      <p class="kicker">Provenance</p>
      <h2>Held in context</h2>
      <dl>
        <div>
          <dt>People</dt>
          <dd>{detail.creators.length}</dd>
        </div>
        <div>
          <dt>Relations</dt>
          <dd>{detail.relations.length}</dd>
        </div>
        <div>
          <dt>Events</dt>
          <dd>{detail.events.length}</dd>
        </div>
        <div>
          <dt>Work kind</dt>
          <dd>{detail.work.work_kind}</dd>
        </div>
      </dl>
    </aside>
  </div>
</article>

<style>
  .grapher-media-detail {
    display: grid;
    gap: 1rem;
    font-family: var(--font-mono);
  }
  h1,
  h2,
  p {
    margin: 0;
  }
  h1 {
    max-width: 16ch;
    font-family: var(--font-sans);
    font-size: clamp(2.8rem, 8vw, 7rem);
    letter-spacing: -0.12em;
    line-height: 0.82;
  }
  h2 {
    font-family: var(--font-sans);
    font-size: 1.15rem;
  }
  .kicker {
    margin-bottom: 0.45rem;
    color: var(--accent);
    font-size: 0.64rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .detail-header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: end;
    border-bottom: 3px solid var(--text);
    padding-bottom: 1.5rem;
  }
  .detail-header p:last-child,
  .muted,
  dt {
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  .rating {
    color: var(--accent);
    font-family: var(--font-sans);
    font-size: 3.5rem;
    letter-spacing: -0.1em;
    white-space: nowrap;
  }
  .rating small {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.75rem;
    letter-spacing: 0;
  }
  .cover {
    width: 9rem;
    max-height: 13rem;
    object-fit: cover;
    border: 1px solid var(--border);
  }
  .chart-panel,
  .record-grid > section,
  .record-grid > aside {
    border: 1px solid var(--border);
    background: var(--surface);
    padding: 1rem;
  }
  .chart-panel {
    border-top: 4px solid var(--accent);
  }
  .chart-panel header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: end;
    margin-bottom: 0.7rem;
  }
  .chart-panel header > strong {
    color: var(--accent);
    font-family: var(--font-sans);
    font-size: 2rem;
  }
  .progress {
    height: 0.75rem;
    margin: 1.5rem 0 0.65rem;
    background: var(--surface-3);
  }
  .progress span {
    display: block;
    height: 100%;
    background: var(--accent);
  }
  .record-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.4fr) minmax(16rem, 0.8fr);
    gap: 1rem;
  }
  ol {
    margin: 0.8rem 0 0;
    padding: 0;
    list-style: none;
  }
  li {
    display: grid;
    grid-template-columns: 7rem minmax(0, 1fr) auto;
    gap: 0.8rem;
    border-top: 1px solid var(--border);
    padding: 0.7rem 0;
    font-size: 0.72rem;
  }
  li time {
    color: var(--text-muted);
  }
  li strong {
    color: var(--accent);
  }
  dl {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 0.8rem;
    margin: 1rem 0 0;
  }
  dd {
    margin: 0.25rem 0 0;
    font-size: 0.85rem;
  }
  @media (max-width: 720px) {
    .detail-header,
    .record-grid {
      display: grid;
    }
    li {
      grid-template-columns: 5.5rem minmax(0, 1fr) auto;
    }
  }
</style>
