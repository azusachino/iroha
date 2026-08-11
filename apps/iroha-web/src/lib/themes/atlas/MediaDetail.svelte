<script lang="ts">
  import type { MediaDetail } from "$lib/api";
  import {
    cleanDescription,
    formatProgressCount,
    mediaEventLabel,
    mediaWorkTotal,
  } from "$lib/format";
  import { heroTitleFontSize } from "$lib/hero-title";

  let { detail, progress }: { detail: MediaDetail; progress: number } =
    $props();
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
  const HERO_TITLE_CLAMP = { minRem: 2.4, vw: 6, maxRem: 4.4 };
</script>

<article class="atlas-entry">
  <header class="entry-hero">
    <div class="cover-frame">
      {#if detail.item.cover_image_url}<img
          src={detail.item.cover_image_url}
          alt=""
        />{:else}<span aria-hidden="true">{detail.item.title.slice(0, 1)}</span
        >{/if}
    </div>
    <div class="hero-copy">
      <p class="atlas-kicker">
        {detail.item.media_type.replaceAll("_", " ")} · catalog entry
      </p>
      <h1
        style:font-size={heroTitleFontSize(
          detail.item.native_title || detail.item.title,
          HERO_TITLE_CLAMP,
        )}
      >
        {detail.item.native_title || detail.item.title}
      </h1>
      {#if detail.item.native_title && detail.item.native_title !== detail.item.title}<p
          class="original-title"
        >
          {detail.item.title}
        </p>{/if}
      <p class="description">
        {cleanDescription(detail.work.description) ||
          "A media record held in the personal catalog."}
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

  <div class="entry-grid">
    <section class="atlas-plate progress-plate">
      <div class="panel-heading">
        <div>
          <p class="atlas-kicker">Position</p>
          <h2>{detail.progress?.unit || "Current position"}</h2>
        </div>
        <strong>{progressLabel}</strong>
      </div>
      <div class="scale-bar">
        <div class="scale-track">
          <i class="scale-fill" style={`width: ${boundedProgress}%`}></i>
        </div>
        <div class="scale-ticks">
          <span>0</span><span>25</span><span>50</span><span>75</span><span
            >100%</span
          >
        </div>
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
    <aside class="atlas-plate provenance-plate">
      <p class="atlas-kicker">Provenance</p>
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

  <div class="entry-grid-lower">
    <section class="atlas-plate timeline-plate">
      <div class="panel-heading">
        <div>
          <p class="atlas-kicker">Timeline</p>
          <h2>Watch history</h2>
        </div>
        <span>{detail.events.length} entries</span>
      </div>
      {#if detail.events.length}<ol class="waypoint-list">
          {#each detail.events.slice(0, 10) as event, index (event.id)}<li>
              <span class="waypoint-index"
                >{String(index + 1).padStart(2, "0")}</span
              ><b>{event.event_at?.slice(0, 10) ?? "undated"}</b><span
                >{mediaEventLabel(event.event_type)}</span
              >{#if event.progress_percent != null}<strong
                  >{Math.round(event.progress_percent)}%</strong
                >{/if}
            </li>{/each}
        </ol>{:else}<p class="atlas-empty">No event history recorded.</p>{/if}
    </section>
    {#if detail.relations.length}<section class="atlas-plate relations-plate">
        <p class="atlas-kicker">Connections</p>
        <h2>Related works</h2>
        <div class="relations">
          {#each detail.relations.slice(0, 6) as relation (relation.id)}<a
              href={`/library/${relation.related_item_id}`}
              ><span>{relation.related_title}</span><small
                >{relation.relation_type.replaceAll("_", " ")}</small
              ></a
            >{/each}
        </div>
      </section>{/if}
  </div>
  <footer class="atlas-source">
    Source: imported media record · presentation only
  </footer>
</article>

<style>
  .atlas-entry {
    display: grid;
    gap: 1.5rem;
    font-family: var(--font-sans);
  }
  .atlas-kicker {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    margin: 0 0 0.45rem;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.64rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .atlas-kicker::before {
    content: "⌖";
  }
  h1,
  h2 {
    margin: 0;
    font-weight: 600;
    letter-spacing: -0.03em;
  }
  h1 {
    max-width: min(34rem, 100%);
    line-height: 1;
  }
  h2 {
    font-size: 1.3rem;
  }
  .entry-hero {
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
    border-radius: calc(var(--radius) * 0.6);
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
    font-family: var(--font-mono);
    font-size: 3.5rem;
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
    font-size: 0.76rem;
  }
  .meta-row strong {
    color: var(--accent);
  }
  .atlas-plate {
    position: relative;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface-1) 88%, transparent);
  }
  .atlas-plate::before,
  .atlas-plate::after {
    content: "";
    position: absolute;
    width: 0.7rem;
    height: 0.7rem;
    opacity: 0.7;
  }
  .atlas-plate::before {
    top: -1px;
    left: -1px;
    border-top: 2px solid var(--accent);
    border-left: 2px solid var(--accent);
  }
  .atlas-plate::after {
    right: -1px;
    bottom: -1px;
    border-right: 2px solid var(--accent);
    border-bottom: 2px solid var(--accent);
  }
  .entry-grid,
  .entry-grid-lower {
    display: grid;
    grid-template-columns: 1.4fr 0.8fr;
    gap: 1.25rem;
  }
  .progress-plate,
  .provenance-plate,
  .timeline-plate,
  .relations-plate {
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
    font-family: var(--font-mono);
    font-size: 0.75rem;
  }
  .panel-heading > strong {
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 2.2rem;
    font-weight: 600;
  }
  .scale-bar {
    margin-top: 1.5rem;
  }
  .scale-track {
    height: 0.65rem;
    border: 1px solid var(--border);
    background: repeating-linear-gradient(
      90deg,
      color-mix(in srgb, var(--border) 70%, transparent) 0 1px,
      transparent 1px 10%
    );
  }
  .scale-fill {
    display: block;
    height: 100%;
    background: var(--accent);
  }
  .scale-ticks {
    display: flex;
    justify-content: space-between;
    margin-top: 0.4rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.6rem;
  }
  .progress-meta {
    display: flex;
    justify-content: space-between;
    margin-top: 0.8rem;
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
    border-top: 1px dashed var(--border);
    padding: 0.7rem 0;
  }
  dt {
    color: var(--text-muted);
    font-size: 0.76rem;
  }
  dd {
    margin: 0;
    font-family: var(--font-mono);
  }
  .waypoint-list {
    display: grid;
    gap: 0;
    margin: 1.25rem 0 0;
    padding: 0;
    list-style: none;
  }
  .waypoint-list li {
    display: grid;
    grid-template-columns: 1.8rem 6rem 1fr auto;
    gap: 1rem;
    align-items: baseline;
    border-top: 1px solid var(--border);
    padding: 0.75rem 0;
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  .waypoint-index {
    display: inline-block;
    border: 1px solid var(--accent);
    border-radius: 50%;
    padding: 0.1rem 0;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.6rem;
    text-align: center;
  }
  .waypoint-list b {
    color: var(--accent);
    font-family: var(--font-mono);
    font-weight: 400;
  }
  .waypoint-list strong {
    color: var(--accent);
    font-family: var(--font-mono);
  }
  .atlas-empty {
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
  .relations a:hover span {
    color: var(--accent);
  }
  .relations small {
    color: var(--text-muted);
    font-family: var(--font-mono);
  }
  .atlas-source {
    border-top: 1px solid var(--border);
    padding-top: 0.85rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.64rem;
  }
  @media (max-width: 700px) {
    .entry-hero,
    .entry-grid,
    .entry-grid-lower {
      grid-template-columns: 1fr;
    }
    .cover-frame {
      width: 8rem;
    }
  }
</style>
