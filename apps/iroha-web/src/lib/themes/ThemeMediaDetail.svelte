<script lang="ts">
  import type { MediaDetail } from "$lib/api";

  export type MediaDetailVariant =
    "atlas" | "field-journal" | "phenology" | "sound-map" | "archive";
  let {
    variant,
    detail,
    progress,
  }: { variant: MediaDetailVariant; detail: MediaDetail; progress: number } =
    $props();
  const boundedProgress = $derived(Math.min(Math.max(progress, 0), 100));
</script>

<article class={`theme-media-detail theme-media-detail-${variant}`}>
  <header class="media-hero">
    <div class="cover-frame">
      {#if detail.item.cover_image_url}<img
          src={detail.item.cover_image_url}
          alt=""
        />{:else}<span aria-hidden="true">{detail.item.title.slice(0, 1)}</span
        >{/if}
    </div>
    <div class="hero-copy">
      <p class="theme-kicker">
        {detail.item.media_type.replaceAll("_", " ")} / collection record
      </p>
      <h1>{detail.item.native_title || detail.item.title}</h1>
      {#if detail.item.native_title && detail.item.native_title !== detail.item.title}<p
          class="original-title"
        >
          {detail.item.title}
        </p>{/if}
      <p class="description">
        {detail.work.description ||
          "A media record held in the personal archive."}
      </p>
      <div class="meta-row">
        <span>{detail.item.status?.replaceAll("_", " ") ?? "untracked"}</span
        >{#if detail.item.rating != null}<strong
            >{detail.item.rating.toFixed(1)} / 10</strong
          >{/if}{#if detail.work.first_release_date}<span
            >{detail.work.first_release_date.slice(0, 4)}</span
          >{/if}
      </div>
    </div>
  </header>

  <div class="record-grid">
    <section class="progress-panel">
      <div class="panel-heading">
        <div>
          <p class="theme-kicker">Continuity</p>
          <h2>{detail.progress?.unit || "Current position"}</h2>
        </div>
        <strong>{Math.round(boundedProgress)}%</strong>
      </div>
      <div class="progress-track">
        <span style={`width: ${boundedProgress}%`}></span>
      </div>
      <div class="progress-meta">
        <span
          >{detail.progress?.position ?? 0}{detail.progress?.total != null
            ? ` / ${detail.progress.total}`
            : ""}</span
        ><span
          >{detail.progress?.play_count
            ? `${detail.progress.play_count} replays`
            : "No replays logged"}</span
        >
      </div>
    </section>
    <aside class="provenance-panel">
      <p class="theme-kicker">Provenance</p>
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
          <dt>Recorded events</dt>
          <dd>{detail.events.length}</dd>
        </div>
        <div>
          <dt>Work kind</dt>
          <dd>{detail.work.work_kind}</dd>
        </div>
      </dl>
    </aside>
  </div>

  <div class="lower-grid">
    <section class="events-panel">
      <div class="panel-heading">
        <div>
          <p class="theme-kicker">Timeline</p>
          <h2>Watch history</h2>
        </div>
        <span>{detail.events.length} entries</span>
      </div>
      {#if detail.events.length}<ol>
          {#each detail.events.slice(0, 10) as event (event.id)}<li>
              <b>{event.event_at?.slice(0, 10) ?? "undated"}</b><span
                >{event.event_type.replaceAll("_", " ")}</span
              >{#if event.progress_percent != null}<strong
                  >{Math.round(event.progress_percent)}%</strong
                >{/if}
            </li>{/each}
        </ol>{:else}<p class="empty">No event history recorded.</p>{/if}
    </section>
    {#if detail.relations.length}<section class="relations-panel">
        <p class="theme-kicker">Connections</p>
        <h2>Related works</h2>
        <div class="relations">
          {#each detail.relations.slice(0, 6) as relation (relation.id)}<a
              href={`/media/${relation.related_item_id}`}
              ><span>{relation.related_title}</span><small
                >{relation.relation_type.replaceAll("_", " ")}</small
              ></a
            >{/each}
        </div>
      </section>{/if}
  </div>
</article>

<style>
  .theme-media-detail {
    display: grid;
    gap: 1.25rem;
  }
  h1,
  h2 {
    margin: 0;
    font-family: Georgia, serif;
    font-weight: 400;
    letter-spacing: -0.05em;
  }
  h1 {
    max-width: 12ch;
    font-size: clamp(2.6rem, 7vw, 6rem);
    line-height: 0.9;
  }
  h2 {
    font-size: 1.55rem;
  }
  .theme-kicker {
    margin: 0 0 0.45rem;
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  .media-hero {
    display: grid;
    grid-template-columns: 12rem 1fr;
    gap: 2rem;
    align-items: center;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.75rem;
  }
  .cover-frame {
    aspect-ratio: 2/3;
    overflow: hidden;
    border: 1px solid var(--border);
    background: var(--surface-2);
  }
  .cover-frame img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
  .cover-frame span {
    display: grid;
    width: 100%;
    height: 100%;
    place-items: center;
    color: var(--accent);
    font:
      4rem Georgia,
      serif;
  }
  .hero-copy {
    display: grid;
    gap: 0.8rem;
  }
  .original-title,
  .description {
    margin: 0;
    color: var(--text-muted);
  }
  .description {
    max-width: 55ch;
    line-height: 1.55;
  }
  .meta-row {
    display: flex;
    flex-wrap: wrap;
    gap: 0.8rem;
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .meta-row strong {
    color: var(--accent);
  }
  .record-grid,
  .lower-grid {
    display: grid;
    grid-template-columns: 1.4fr 0.8fr;
    gap: 1.25rem;
  }
  .progress-panel,
  .provenance-panel,
  .events-panel,
  .relations-panel {
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface-1) 88%, transparent);
    padding: 1.25rem;
  }
  .panel-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: end;
  }
  .panel-heading > span {
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .panel-heading > strong {
    color: var(--accent);
    font:
      2.5rem Georgia,
      serif;
  }
  .progress-track {
    height: 0.7rem;
    margin: 2rem 0 0.8rem;
    overflow: hidden;
    background: var(--surface-3);
  }
  .progress-track span {
    display: block;
    height: 100%;
    background: var(--accent);
  }
  .progress-meta {
    display: flex;
    justify-content: space-between;
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  dl {
    margin: 1.25rem 0 0;
  }
  dl div {
    display: flex;
    justify-content: space-between;
    border-top: 1px solid var(--border);
    padding: 0.7rem 0;
  }
  dt {
    color: var(--text-muted);
    font-size: 0.76rem;
  }
  dd {
    margin: 0;
  }
  ol {
    display: grid;
    gap: 0;
    margin: 1.25rem 0 0;
    padding: 0;
    list-style: none;
  }
  li {
    display: grid;
    grid-template-columns: 7rem 1fr auto;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding: 0.75rem 0;
    font-size: 0.8rem;
  }
  li b {
    color: var(--accent);
    font-weight: 400;
  }
  li strong {
    color: var(--accent);
  }
  .empty {
    color: var(--text-muted);
  }
  .relations {
    display: grid;
    gap: 0.5rem;
    margin-top: 1.25rem;
  }
  .relations a {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding: 0.7rem 0;
    color: var(--text);
    text-decoration: none;
  }
  .relations small {
    color: var(--text-muted);
  }
  @media (max-width: 700px) {
    .media-hero,
    .record-grid,
    .lower-grid {
      grid-template-columns: 1fr;
    }
    .cover-frame {
      width: 8rem;
    }
  }
</style>
