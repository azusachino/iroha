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
  const HERO_TITLE_CLAMP = { minRem: 2.4, vw: 6.5, maxRem: 5.2 };

  function tone(pct: number): string {
    const clamped = Math.max(0, Math.min(100, pct));
    return `color-mix(in srgb, var(--accent-2) ${clamped}%, var(--accent) ${100 - clamped}%)`;
  }

  // Derived from the item's own type and id -- a stable catalog number, not
  // a decoration -- following the same accessioning convention used for
  // activities and days elsewhere in this language.
  const accession = $derived(
    `${detail.item.media_type.slice(0, 3).toUpperCase()}-${detail.item.id.slice(0, 6).toUpperCase()}`,
  );
</script>

<article class="folio-detail-page" data-theme={theme}>
  <header class="media-hero">
    <div class="cover-frame">
      {#if detail.item.cover_image_url}
        <img src={detail.item.cover_image_url} alt="" />
      {:else}
        <span aria-hidden="true">{detail.item.title.slice(0, 1)}</span>
      {/if}
      <span class="cover-tag">{accession}</span>
    </div>
    <div class="hero-copy">
      <p class="folio-kicker">
        {detail.item.media_type.replaceAll("_", " ")} / collection record
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
          <p class="folio-kicker">Continuity</p>
          <h2>{detail.progress?.unit || "Current position"}</h2>
        </div>
        <strong class="progress-readout">{progressLabel}</strong>
      </div>
      <div class="progress-row">
        {#if hasKnownTotal}
          <div class="core-gauge-well">
            <i
              style={`height: ${boundedProgress}%; background: ${tone(boundedProgress)};`}
            ></i>
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
      </div>
    </section>
    <aside class="provenance-panel catalog-card">
      <p class="folio-kicker">Provenance</p>
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
          <p class="folio-kicker">Accession log</p>
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
            <p class="folio-kicker">Provider record</p>
            <h2>Reading updates</h2>
          </div>
          <span>{detail.updates.length} entries</span>
        </div>
        <MediaUpdateList updates={detail.updates} />
      </section>
    {/if}
    {#if detail.relations.length}
      <section class="relations-panel">
        <p class="folio-kicker">Connections</p>
        <h2>Related works</h2>
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
  .folio-detail-page {
    display: grid;
    gap: 1.3rem;
  }
  h1,
  h2 {
    margin: 0;
    font-family: var(--font-serif);
    font-weight: 700;
    letter-spacing: -0.01em;
  }
  h1 {
    max-width: min(34rem, 100%);
    line-height: 0.95;
  }
  h2 {
    font-size: 1.4rem;
  }
  .folio-kicker {
    margin: 0 0 0.45rem;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.64rem;
    letter-spacing: 0.15em;
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
    position: relative;
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
  .cover-frame span:not(.cover-tag) {
    display: grid;
    width: 100%;
    height: 100%;
    place-items: center;
    color: var(--accent);
    font-family: var(--font-serif);
    font-size: 3.6rem;
    font-weight: 700;
  }
  .cover-tag {
    position: absolute;
    right: 0.4rem;
    bottom: 0.4rem;
    border: 1px solid color-mix(in srgb, var(--accent) 65%, transparent);
    border-radius: 2px;
    padding: 0.15rem 0.4rem;
    background: color-mix(in srgb, var(--bg) 60%, transparent);
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.62rem;
    letter-spacing: 0.03em;
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
    font-family: var(--font-mono);
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.03em;
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
  .record-grid > *,
  .lower-grid > * {
    min-width: 0;
  }
  .progress-panel,
  .catalog-card,
  .events-panel,
  .relations-panel {
    position: relative;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
    padding: 1.25rem;
  }
  .catalog-card {
    padding-left: 1.45rem;
  }
  .catalog-card::before {
    content: "";
    position: absolute;
    left: -1px;
    top: 1.1rem;
    width: 4px;
    height: 2.2rem;
    background: var(--accent-2);
    border-radius: 0 2px 2px 0;
  }
  .panel-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: end;
  }
  .panel-heading > span {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.75rem;
  }
  .progress-readout {
    color: var(--accent);
    font-family: var(--font-serif);
    font-size: 2.2rem;
    font-weight: 700;
  }
  .progress-row {
    display: flex;
    align-items: center;
    gap: 1.25rem;
    margin-top: 1.5rem;
  }
  .core-gauge-well {
    position: relative;
    width: 1.5rem;
    height: 4rem;
    flex-shrink: 0;
    overflow: hidden;
    border: 1px solid var(--border);
    border-radius: 2px;
    background: color-mix(in srgb, var(--surface) 94%, transparent);
  }
  .core-gauge-well i {
    position: absolute;
    right: 0;
    bottom: 0;
    left: 0;
    display: block;
  }
  .progress-meta {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 0.5rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
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
    font-family: var(--font-mono);
    font-size: 0.74rem;
  }
  dd {
    margin: 0;
    font-family: var(--font-mono);
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
    font-family: var(--font-mono);
    font-size: 0.78rem;
  }
  li b {
    color: var(--accent);
    font-weight: 400;
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
    font-family: var(--font-mono);
  }
  @media (max-width: 768px) {
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
