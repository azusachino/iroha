<script lang="ts">
  import type {
    MediaAggregates,
    MediaCompletionBucket,
    MediaRow,
    MediaScoreBucket,
  } from "$lib/api";
  import { boundPercent, formatProgressCount } from "$lib/format";

  let {
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

  const families = [
    { value: "", label: "All" },
    { value: "anime", label: "Anime" },
    { value: "manga_book", label: "Manga & books" },
    { value: "game", label: "Games" },
  ];
</script>

<section class="mix-media" aria-labelledby="mix-media-title">
  <header class="mix-head">
    <div>
      <p class="mix-kicker">Playlist / indexed shelf</p>
      <h1 id="mix-media-title">Things kept in rotation.</h1>
      <p>
        Reading, watching, and playing as a record of attention—not a backlog.
      </p>
    </div>
    <div class="mix-readout">
      <strong>{aggregates.totals.item_count}</strong><span>titles</span>
    </div>
  </header>

  <nav class="mix-tabs" aria-label="Media family">
    {#each families as option (option.value)}
      <button
        class:active={family === option.value}
        type="button"
        onclick={() => onFamily(option.value)}>{option.label}</button
      >
    {/each}
  </nav>

  <div class="mix-filters">
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
        ><option value="">All years</option
        >{#each yearOptions as option (option.year)}<option value={option.year}
            >{option.year}</option
          >{/each}</select
      ></label
    >
  </div>

  <div class="mix-stats">
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

  <section class="mix-shelf">
    <header>
      <div>
        <p class="mix-kicker">Current rotation</p>
        <h2>{items.length} visible titles</h2>
      </div>
      <span
        >{typeFamilies
          .map((item) => `${item.type}: ${item.count}`)
          .join(" · ")}</span
      >
    </header>
    {#if items.length === 0}
      <p class="mix-empty">No titles match this rotation.</p>
    {:else}
      <div class="mix-grid">
        {#each items as item (item.id)}
          <a class="mix-card" href={`/media/${item.id}`}>
            {#if item.cover_image_url}
              <img src={item.cover_image_url} alt="" loading="lazy" />
            {:else}
              <span class="mix-initial">{item.title.slice(0, 1)}</span>
            {/if}
            <strong>{item.native_title || item.title}</strong>
            <small
              >{item.status || "unknown"} · {formatProgressCount(
                item.position,
                item.total,
                item.unit,
              )}</small
            >
            <div class="mix-scrub">
              <i style={`width: ${boundPercent(item.progress_percent)}%`}></i>
              <b style={`left: ${boundPercent(item.progress_percent)}%`}></b>
            </div>
          </a>
        {/each}
      </div>
    {/if}
  </section>
  {#if hasMore}
    <button
      class="mix-load-more"
      type="button"
      disabled={loadingMore}
      onclick={onLoadMore}>{loadingMore ? "Loading…" : "Load more"}</button
    >
  {/if}
  <footer class="mix-source">
    <span
      >{completions.length} completion periods · {scores.length} score buckets</span
    >
    <span>Source: imported provider records</span>
  </footer>
</section>

<style>
  .mix-media {
    display: grid;
    gap: 1.35rem;
  }
  .mix-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    font-weight: 700;
    letter-spacing: -0.03em;
  }
  h1 {
    max-width: 13ch;
    font-size: clamp(2.3rem, 6vw, 4.4rem);
    line-height: 0.98;
    text-transform: uppercase;
  }
  h2 {
    font-size: 1.65rem;
  }
  .mix-head {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.75rem;
  }
  .mix-head p:last-child {
    color: var(--text-muted);
    line-height: 1.6;
  }
  .mix-readout {
    display: grid;
    justify-items: end;
    padding: 0.6rem 0.9rem;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    color: var(--text-muted);
  }
  .mix-readout strong {
    color: var(--accent);
    font-size: 2.5rem;
    font-weight: 700;
    line-height: 1;
    font-variant-numeric: tabular-nums;
  }
  .mix-readout span {
    margin-top: 0.4rem;
    font-size: 0.62rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .mix-tabs {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
  }
  .mix-tabs button {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 0.45rem 0.85rem;
    background: transparent;
    color: var(--text-muted);
    font: inherit;
    font-size: 0.72rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    cursor: pointer;
  }
  .mix-tabs button.active,
  .mix-tabs button:hover {
    border-color: var(--accent);
    color: var(--accent);
  }
  .mix-filters {
    display: flex;
    gap: 0.75rem;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1rem;
  }
  label {
    display: grid;
    gap: 0.3rem;
    color: var(--text-muted);
    font-size: 0.64rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  select {
    min-width: 9rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 0.5rem;
    background: var(--surface);
    color: var(--text);
    font: inherit;
    font-size: 0.8rem;
  }
  .mix-stats {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }
  .mix-stats div {
    display: grid;
    gap: 0.35rem;
    border-right: 1px solid var(--border);
    padding: 1.1rem;
  }
  .mix-stats div:last-child {
    border-right: 0;
  }
  .mix-stats span {
    color: var(--text-muted);
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  .mix-stats strong {
    font-size: 1.5rem;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
  }
  .mix-shelf {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 1.4rem;
    background: color-mix(in srgb, var(--surface) 90%, transparent);
  }
  .mix-shelf > header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .mix-shelf > header > span {
    color: var(--text-muted);
    font-size: 0.7rem;
    text-align: right;
  }
  .mix-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(9rem, 1fr));
    gap: 1rem;
    margin-top: 1.25rem;
  }
  .mix-card {
    display: grid;
    gap: 0.35rem;
    color: var(--text);
    text-decoration: none;
  }
  .mix-card img,
  .mix-initial {
    display: block;
    width: 100%;
    aspect-ratio: 3 / 4;
    object-fit: cover;
    border-radius: calc(var(--radius) * 0.6);
    background: color-mix(in srgb, var(--accent) 12%, var(--surface));
  }
  .mix-initial {
    display: grid;
    place-items: center;
    color: var(--accent);
    font-size: 2.2rem;
    font-weight: 700;
  }
  .mix-card strong {
    overflow: hidden;
    font-size: 0.86rem;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .mix-card small {
    overflow: hidden;
    color: var(--text-muted);
    font-size: 0.64rem;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .mix-scrub {
    position: relative;
    height: 0.2rem;
    margin-top: 0.15rem;
    background: var(--border);
  }
  .mix-scrub i {
    display: block;
    height: 100%;
    background: var(--accent);
  }
  .mix-scrub b {
    position: absolute;
    top: 50%;
    width: 0.45rem;
    height: 0.45rem;
    border-radius: 50%;
    background: var(--accent-2);
    transform: translate(-50%, -50%);
  }
  .mix-empty {
    color: var(--text-muted);
  }
  .mix-load-more {
    justify-self: center;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    padding: 0.55rem 1.1rem;
    background: transparent;
    color: var(--accent);
    font: inherit;
    font-size: 0.75rem;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    cursor: pointer;
  }
  .mix-load-more:disabled {
    opacity: 0.5;
  }
  .mix-source {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding-top: 0.8rem;
    color: var(--text-muted);
    font-size: 0.66rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  @media (max-width: 680px) {
    .mix-head,
    .mix-shelf > header,
    .mix-source {
      display: block;
    }
    .mix-readout {
      display: flex;
      align-items: baseline;
      justify-items: initial;
      gap: 0.5rem;
      margin-top: 1.5rem;
    }
    .mix-readout strong {
      font-size: 2.1rem;
    }
    .mix-shelf > header > span {
      display: block;
      margin-top: 0.8rem;
      text-align: left;
    }
    .mix-filters,
    .mix-stats {
      grid-template-columns: 1fr;
      display: grid;
    }
    .mix-stats div {
      border-right: 0;
      border-bottom: 1px solid var(--border);
    }
    .mix-stats div:last-child {
      border-bottom: 0;
    }
  }
</style>
