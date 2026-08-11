<script lang="ts">
  import type {
    MediaAggregates,
    MediaCompletionBucket,
    MediaRow,
    MediaScoreBucket,
  } from "$lib/api";
  import { boundPercent, formatDateOnly, formatPercent } from "$lib/format";

  export type MediaVariant =
    "atlas" | "field-journal" | "phenology" | "sound-map" | "archive";
  let {
    variant,
    items,
    aggregates,
    family,
    status,
    completedYear,
    yearOptions,
    typeFamilies,
    completions,
    scores,
    onFamily,
    onStatus,
    onYear,
    onLoadMore,
    hasMore,
    loadingMore,
  }: {
    variant: MediaVariant;
    items: MediaRow[];
    aggregates: MediaAggregates;
    family: string;
    status: string;
    completedYear: string;
    yearOptions: MediaCompletionBucket[];
    typeFamilies: { type: string; count: number }[];
    completions: MediaCompletionBucket[];
    scores: MediaScoreBucket[];
    onFamily: (value: string) => void;
    onStatus: () => void;
    onYear: () => void;
    onLoadMore: () => void;
    hasMore: boolean;
    loadingMore: boolean;
  } = $props();
</script>

<section
  class={`theme-media theme-media-${variant}`}
  aria-labelledby="theme-media-title"
>
  <header class="media-head">
    <div>
      <p class="theme-kicker">Library / indexed shelf</p>
      <h1 id="theme-media-title">Things kept in orbit.</h1>
      <p>
        Reading, watching, and playing as a record of attention—not a backlog.
      </p>
    </div>
    <strong>{aggregates.totals.item_count}<small> titles</small></strong>
  </header>
  <nav class="media-families" aria-label="Media family">
    {#each [{ value: "", label: "All" }, { value: "anime", label: "Anime" }, { value: "manga_book", label: "Manga & books" }, { value: "game", label: "Games" }] as option}<button
        class:active={family === option.value}
        type="button"
        onclick={() => onFamily(option.value)}>{option.label}</button
      >{/each}
  </nav>
  <div class="media-filters">
    <label
      >Status<select value={status} onchange={onStatus}
        ><option value="">All statuses</option><option value="in_progress"
          >In progress</option
        ><option value="completed">Completed</option><option value="planned"
          >Planned</option
        ><option value="abandoned">Abandoned</option></select
      ></label
    ><label
      >Completed year<select value={completedYear} onchange={onYear}
        ><option value="">All years</option>{#each yearOptions as option}<option
            value={option.year}>{option.year}</option
          >{/each}</select
      ></label
    >
  </div>
  <div class="media-stats">
    <div>
      <span>Completed</span><strong>{aggregates.totals.completed_count}</strong>
    </div>
    <div>
      <span>This year</span><strong
        >{aggregates.totals.this_year_completed}</strong
      >
    </div>
    <div>
      <span>Average score</span><strong
        >{aggregates.totals.average_rating
          ? aggregates.totals.average_rating.toFixed(1)
          : "—"}</strong
      >
    </div>
  </div>
  <section class="media-shelf">
    <header>
      <div>
        <p class="theme-kicker">Current shelf</p>
        <h2>{items.length} visible titles</h2>
      </div>
      <span
        >{typeFamilies
          .map((item) => `${item.type}: ${item.count}`)
          .join(" · ")}</span
      >
    </header>
    <div class="media-grid">
      {#each items as item (item.id)}<a
          class="media-card"
          href={`/library/${item.id}`}
          >{#if item.cover_image_url}<img
              src={item.cover_image_url}
              alt=""
              loading="lazy"
            />{:else}<span class="media-initial">{item.title.slice(0, 1)}</span
            >{/if}<strong>{item.native_title || item.title}</strong><small
            >{item.status || "unknown"} · {formatPercent(
              item.progress_percent ?? 0,
            )}</small
          ><i
            ><b style={`width: ${boundPercent(item.progress_percent)}%`}></b></i
          ></a
        >{/each}
    </div>
    {#if items.length === 0}<p class="media-empty">
        No titles match this shelf.
      </p>{/if}
  </section>
  {#if hasMore}<button
      class="load-more"
      type="button"
      disabled={loadingMore}
      onclick={onLoadMore}>{loadingMore ? "Loading…" : "Load more"}</button
    >{/if}
  <footer class="media-source">
    {completions.length} completion periods · {scores.length} score buckets · source:
    imported provider records
  </footer>
</section>

<style>
  .theme-media {
    display: grid;
    gap: 1.25rem;
  }
  .theme-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.14em;
    text-transform: uppercase;
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
    font-size: clamp(2.8rem, 7vw, 6rem);
    line-height: 0.9;
  }
  h2 {
    font-size: 1.7rem;
  }
  .media-head {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 2rem;
  }
  .media-head p:last-child {
    color: var(--text-muted);
    line-height: 1.6;
  }
  .media-head > strong {
    color: var(--accent);
    font-family: Georgia, serif;
    font-size: 3.4rem;
    font-weight: 400;
    white-space: nowrap;
  }
  .media-head > strong small {
    color: var(--text-muted);
    font-family: inherit;
    font-size: 0.8rem;
  }
  .media-families {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
  }
  .media-families button {
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 0.45rem 0.8rem;
    background: transparent;
    color: var(--text-muted);
    font: inherit;
    font-size: 0.75rem;
    cursor: pointer;
  }
  .media-families button.active,
  .media-families button:hover {
    border-color: var(--accent);
    color: var(--accent);
  }
  .media-filters {
    display: flex;
    gap: 0.75rem;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1rem;
  }
  label {
    display: grid;
    gap: 0.3rem;
    color: var(--text-muted);
    font-size: 0.65rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  select {
    min-width: 9rem;
    border: 1px solid var(--border);
    padding: 0.5rem;
    background: var(--surface-1);
    color: var(--text);
    font: inherit;
    font-size: 0.8rem;
  }
  .media-stats {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    border: 1px solid var(--border);
  }
  .media-stats div {
    display: grid;
    gap: 0.35rem;
    border-right: 1px solid var(--border);
    padding: 1.1rem;
  }
  .media-stats div:last-child {
    border-right: 0;
  }
  .media-stats span {
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  .media-stats strong {
    font-family: Georgia, serif;
    font-size: 1.5rem;
    font-weight: 400;
  }
  .media-shelf {
    border: 1px solid var(--border);
    padding: 1.5rem;
    background: color-mix(in srgb, var(--surface-1) 88%, transparent);
  }
  .media-shelf > header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .media-shelf > header > span {
    color: var(--text-muted);
    font-size: 0.7rem;
    text-align: right;
  }
  .media-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(9rem, 1fr));
    gap: 1rem;
    margin-top: 1.25rem;
  }
  .media-card {
    display: grid;
    gap: 0.35rem;
    color: var(--text);
    text-decoration: none;
  }
  .media-card img,
  .media-initial {
    display: block;
    width: 100%;
    aspect-ratio: 3 / 4;
    object-fit: cover;
    background: color-mix(in srgb, var(--accent) 12%, var(--surface-1));
  }
  .media-initial {
    display: grid;
    place-items: center;
    color: var(--accent);
    font-family: Georgia, serif;
    font-size: 2.5rem;
  }
  .media-card strong {
    overflow: hidden;
    font-family: Georgia, serif;
    font-size: 0.9rem;
    font-weight: 400;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .media-card small {
    overflow: hidden;
    color: var(--text-muted);
    font-size: 0.65rem;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .media-card i {
    display: block;
    height: 0.2rem;
    background: var(--border);
  }
  .media-card i b {
    display: block;
    height: 100%;
    background: var(--accent);
  }
  .media-empty {
    color: var(--text-muted);
  }
  .load-more {
    justify-self: center;
    border: 1px solid var(--accent);
    padding: 0.55rem 1rem;
    background: transparent;
    color: var(--accent);
    font: inherit;
    cursor: pointer;
  }
  .load-more:disabled {
    opacity: 0.5;
  }
  .media-source {
    border-top: 1px solid var(--border);
    padding-top: 0.8rem;
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  .theme-media-atlas {
    font-family: "Avenir Next", sans-serif;
  }
  .theme-media-atlas .media-shelf {
    border-top: 0.35rem solid var(--accent);
  }
  .theme-media-phenology h1,
  .theme-media-phenology h2 {
    font-style: italic;
  }
  .theme-media-phenology .media-card img,
  .theme-media-phenology .media-initial {
    border-radius: 1rem 0.2rem;
  }
  .theme-media-sound-map {
    font-family: "IBM Plex Mono", monospace;
  }
  .theme-media-sound-map h1,
  .theme-media-sound-map h2 {
    font-family: inherit;
  }
  .theme-media-archive .media-shelf {
    box-shadow: 0.45rem 0.45rem 0
      color-mix(in srgb, var(--accent) 18%, transparent);
  }
  @media (max-width: 680px) {
    .media-head,
    .media-shelf > header {
      display: block;
    }
    .media-head > strong {
      display: block;
      margin-top: 1.5rem;
      font-size: 2.6rem;
    }
    .media-shelf > header > span {
      display: block;
      margin-top: 0.8rem;
      text-align: left;
    }
    .media-filters,
    .media-stats {
      grid-template-columns: 1fr;
    }
    .media-filters {
      display: grid;
    }
    .media-stats {
      display: grid;
    }
    .media-stats div {
      border-right: 0;
      border-bottom: 1px solid var(--border);
    }
    .media-stats div:last-child {
      border-bottom: 0;
    }
  }
</style>
