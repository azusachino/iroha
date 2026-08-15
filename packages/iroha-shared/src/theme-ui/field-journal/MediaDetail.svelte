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
  const HERO_TITLE_CLAMP = { minRem: 2.4, vw: 6, maxRem: 4.6 };
</script>

<article class="journal-archive-entry" data-theme={theme}>
  <header class="archive-hero">
    <div class="cover-frame">
      {#if detail.item.cover_image_url}
        <img src={detail.item.cover_image_url} alt="" />
      {:else}
        <span aria-hidden="true">{detail.item.title.slice(0, 1)}</span>
      {/if}
    </div>
    <div class="hero-copy">
      <p class="journal-kicker">
        {detail.item.media_type.replaceAll("_", " ")} · journal entry
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
          "A media record held in the personal journal."}
      </p>
      <div class="meta-row">
        <span>{detail.item.status?.replaceAll("_", " ") ?? "untracked"}</span>
        {#if detail.item.rating != null}
          <strong>{detail.item.rating.toFixed(1)} / 10</strong>
        {/if}
        {#if detail.work.first_release_date}
          <span>{detail.work.first_release_date.slice(0, 4)}</span>
        {/if}
      </div>
    </div>
  </header>

  <div class="journal-rule"><span>continuity</span></div>

  <div class="archive-grid">
    <section class="progress-card">
      <div class="panel-heading">
        <div>
          <p class="journal-kicker">Continuity</p>
          <h2>{detail.progress?.unit || "Current position"}</h2>
        </div>
        <strong>{progressLabel}</strong>
      </div>
      {#if hasKnownTotal}
        <div class="ink-line">
          <i style={`width: ${boundedProgress}%`}></i>
        </div>
      {/if}
      <div class="progress-meta">
        <span
          >{detail.progress?.position ?? 0}{detail.progress?.total != null
            ? ` / ${detail.progress.total}`
            : ""}</span
        >
        <span
          >{detail.progress?.play_count
            ? `${detail.progress.play_count} replays`
            : "No replays logged"}</span
        >
      </div>
    </section>
    <aside class="provenance-card">
      <p class="journal-kicker">Provenance</p>
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

  <div class="archive-grid-lower">
    <section class="timeline-card">
      <div class="panel-heading">
        <div>
          <p class="journal-kicker">Timeline</p>
          <h2>Exact event history</h2>
        </div>
        <span>{detail.events.length} entries</span>
      </div>
      {#if detail.events.length}
        <ol>
          {#each detail.events.slice(0, 10) as event, index (event.id)}
            <li>
              <span class="timeline-index"
                >{String(index + 1).padStart(2, "0")}</span
              >
              <span class="timeline-date"
                >{event.event_at?.slice(0, 10) ?? "undated"}</span
              >
              <span>{mediaEventLabel(event.event_type)}</span>
              {#if event.progress_percent != null}
                <strong>{Math.round(event.progress_percent)}%</strong>
              {/if}
            </li>
          {/each}
        </ol>
      {:else}
        <p class="journal-empty">No exact events recorded.</p>
      {/if}
    </section>
    {#if detail.updates.length}
      <section class="timeline-card">
        <div class="panel-heading">
          <div>
            <p class="journal-kicker">Provider record</p>
            <h2>Reading updates</h2>
          </div>
          <span>{detail.updates.length} entries</span>
        </div>
        <MediaUpdateList updates={detail.updates} />
      </section>
    {/if}
    {#if detail.relations.length}
      <section class="relations-card">
        <p class="journal-kicker">Connections</p>
        <h2>Related works</h2>
        <div class="relations">
          {#each detail.relations.slice(0, 6) as relation (relation.id)}
            <a href={`/library/${relation.related_item_id}`}>
              <span>{relation.related_title}</span>
              <small>{relation.relation_type.replaceAll("_", " ")}</small>
            </a>
          {/each}
        </div>
      </section>
    {/if}
  </div>

  <footer class="journal-source">
    <span>Source: imported media record</span>
    <span>Presentation only</span>
  </footer>
</article>

<style>
  .journal-archive-entry {
    display: grid;
    gap: 1.5rem;
  }
  .journal-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.15em;
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
    max-width: min(34rem, 100%);
    line-height: 0.92;
  }
  h2 {
    font-size: 1.4rem;
  }
  .archive-hero {
    display: grid;
    grid-template-columns: 11rem 1fr;
    gap: 2rem;
    align-items: center;
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
    font-family: var(--font-serif);
    font-size: 3.5rem;
  }
  .hero-copy {
    display: grid;
    gap: 0.7rem;
  }
  .original-title,
  .description {
    margin: 0;
    color: var(--text-muted);
  }
  .description {
    max-width: 52ch;
    line-height: 1.65;
    white-space: pre-line;
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
    font-family: var(--font-serif);
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
  .archive-grid,
  .archive-grid-lower {
    display: grid;
    grid-template-columns: 1.4fr 0.8fr;
    gap: 1.25rem;
  }
  .archive-grid > *,
  .archive-grid-lower > * {
    min-width: 0;
  }
  .progress-card,
  .provenance-card,
  .timeline-card,
  .relations-card {
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface-1) 84%, transparent);
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
    font-family: var(--font-serif);
    font-size: 2.2rem;
    font-weight: 400;
  }
  .ink-line {
    height: 0.35rem;
    margin: 1.75rem 0 0.8rem;
    background: var(--border);
  }
  .ink-line i {
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
    border-bottom: 1px dotted var(--border);
    padding: 0.7rem 0;
  }
  dt {
    color: var(--text-muted);
    font-size: 0.76rem;
  }
  dd {
    margin: 0;
    font-family: var(--font-serif);
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
    grid-template-columns: 1.6rem 5.5rem 1fr auto;
    gap: 1rem;
    align-items: baseline;
    border-top: 1px solid var(--border);
    padding: 0.75rem 0;
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  .timeline-index {
    color: var(--accent);
    font-family: var(--font-serif);
  }
  .timeline-date {
    color: var(--accent);
  }
  li strong {
    color: var(--accent);
    font-family: var(--font-serif);
    font-weight: 400;
  }
  .journal-empty {
    color: var(--text-muted);
  }
  .relations {
    display: grid;
    gap: 0;
    margin-top: 1.25rem;
  }
  .relations a {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-bottom: 1px dotted var(--border);
    padding: 0.7rem 0;
    color: var(--text);
    text-decoration: none;
  }
  .relations a:hover span {
    color: var(--accent);
  }
  .relations small {
    color: var(--text-muted);
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
  @media (max-width: 768px) {
    .archive-hero,
    .archive-grid,
    .archive-grid-lower {
      grid-template-columns: 1fr;
    }
    .cover-frame {
      width: 8rem;
    }
    .journal-source {
      display: block;
    }
  }
</style>
