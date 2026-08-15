<script lang="ts">
  import type { MediaDetailThemeProps } from "../../media";
  import {
    cleanDescription,
    formatProgressCount,
    mediaEventLabel,
    mediaWorkTotal,
  } from "../../media";
  import { heroTitleFontSize } from "../../hero-title";
  import MediaUpdateList from "../components/MediaUpdateList.svelte";

  let { detail, progress, hasKnownTotal, theme }: MediaDetailThemeProps = $props();
  const boundedProgress = $derived(Math.min(Math.max(progress, 0), 100));
  // A percentage implies a known total, which most media never has (an
  // ongoing manga, an unfinished anime season). Show what's actually known.
  const progressLabel = $derived(
    formatProgressCount(
      detail.progress?.position ?? detail.item.position,
      detail.progress?.total ?? detail.item.total,
      detail.progress?.unit ?? detail.item.unit,
      detail.progress?.status ?? detail.item.status,
      mediaWorkTotal(
        detail.item.media_type,
        detail.item.episode_count,
        detail.item.chapter_count,
      ),
    ),
  );
  const HERO_TITLE_CLAMP = { minRem: 2.2, vw: 6, maxRem: 4.6 };
</script>

<article class="mix-detail-page" data-theme={theme}>
  <header class="media-hero">
    <div class="cover-frame">
      {#if detail.item.cover_image_url}
        <img src={detail.item.cover_image_url} alt="" />
      {:else}
        <span aria-hidden="true">{detail.item.title.slice(0, 1)}</span>
      {/if}
    </div>
    <div class="hero-copy">
      <p class="mix-kicker">
        {detail.item.media_type.replaceAll("_", " ")} · collection record
      </p>
      <h1
        style:font-size={heroTitleFontSize(
          detail.item.native_title || detail.item.title,
          HERO_TITLE_CLAMP,
        )}
      >
        {detail.item.native_title || detail.item.title}
      </h1>
      {#if detail.item.native_title && detail.item.native_title !== detail.item.title}
        <p class="original-title">{detail.item.title}</p>
      {/if}
      <p class="description">
        {cleanDescription(detail.work.description) ||
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
          <p class="mix-kicker">Continuity</p>
          <h2>{detail.progress?.unit || "Current position"}</h2>
        </div>
        <strong class="progress-readout">{progressLabel}</strong>
      </div>
      {#if hasKnownTotal}
        <div class="mix-scrub-track">
          <i style={`width: ${boundedProgress}%`}></i>
          <b style={`left: ${boundedProgress}%`}></b>
        </div>
      {/if}
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
      <p class="mix-kicker">Provenance</p>
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
          <dt>Provider updates</dt>
          <dd>{detail.updates.length}</dd>
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
          <p class="mix-kicker">Playback log</p>
          <h2>Exact event history</h2>
        </div>
        <span>{detail.events.length} entries</span>
      </div>
      {#if detail.events.length}
        <ol>
          {#each detail.events.slice(0, 10) as event (event.id)}<li>
              <b>{event.event_at?.slice(0, 10) ?? "undated"}</b><span
                >{mediaEventLabel(event.event_type)}</span
              >{#if event.progress_percent != null}<strong
                  >{Math.round(event.progress_percent)}%</strong
                >{/if}
            </li>{/each}
        </ol>
      {:else}
        <p class="empty">No exact events recorded.</p>
      {/if}
    </section>
    {#if detail.updates.length}
      <section class="events-panel">
        <div class="panel-heading">
          <div>
            <p class="mix-kicker">Provider record</p>
            <h2>Reading updates</h2>
          </div>
          <span>{detail.updates.length} entries</span>
        </div>
        <MediaUpdateList updates={detail.updates} />
      </section>
    {/if}
    {#if detail.relations.length}
      <section class="relations-panel">
        <p class="mix-kicker">Connections</p>
        <h2>Related tracks</h2>
        <div class="relations">
          {#each detail.relations.slice(0, 6) as relation (relation.id)}<a
              href={`/library/${relation.related_item_id}`}
              ><span>{relation.related_title}</span><small
                >{relation.relation_type.replaceAll("_", " ")}</small
              ></a
            >{/each}
        </div>
      </section>
    {/if}
  </div>
</article>

<style>
  .mix-detail-page {
    display: grid;
    gap: 1.35rem;
  }
  h1,
  h2 {
    margin: 0;
    font-weight: 700;
    letter-spacing: -0.03em;
  }
  h1 {
    max-width: min(34rem, 100%);
    line-height: 0.98;
    text-transform: uppercase;
  }
  h2 {
    font-size: 1.4rem;
  }
  .mix-kicker {
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
    border-radius: var(--radius);
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
    font-size: 3.6rem;
    font-weight: 700;
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
    white-space: pre-line;
  }
  .meta-row {
    display: flex;
    flex-wrap: wrap;
    gap: 0.8rem;
    color: var(--text-muted);
    font-size: 0.78rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .meta-row strong {
    color: var(--accent);
    font-variant-numeric: tabular-nums;
  }
  .record-grid,
  .lower-grid {
    display: grid;
    grid-template-columns: 1.4fr 0.8fr;
    gap: 1.25rem;
  }
  .record-grid > *,
  .lower-grid > * {
    min-width: 0;
  }
  .progress-panel,
  .provenance-panel,
  .events-panel,
  .relations-panel {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
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
  .progress-readout {
    color: var(--accent);
    font-size: 2.2rem;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
  }
  .mix-scrub-track {
    position: relative;
    height: 0.5rem;
    margin: 2rem 0 1rem;
    border-radius: 1px;
    background: var(--border);
  }
  .mix-scrub-track i {
    display: block;
    height: 100%;
    background: var(--accent);
  }
  .mix-scrub-track b {
    position: absolute;
    top: 50%;
    width: 0.85rem;
    height: 0.85rem;
    border-radius: 50%;
    background: var(--accent-2);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent-2) 25%, transparent);
    transform: translate(-50%, -50%);
  }
  .progress-meta {
    display: flex;
    justify-content: space-between;
    color: var(--text-muted);
    font-size: 0.75rem;
    font-variant-numeric: tabular-nums;
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
    font-variant-numeric: tabular-nums;
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
    font-variant-numeric: tabular-nums;
  }
  li b {
    color: var(--accent);
    font-weight: 700;
  }
  li strong {
    color: var(--accent-2);
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
  @media (max-width: 768px) {
    .media-hero,
    .record-grid,
    .lower-grid {
      grid-template-columns: 1fr;
    }
    .progress-panel .panel-heading {
      display: grid;
      align-items: start;
    }
    .cover-frame {
      width: 8rem;
    }
  }
</style>
