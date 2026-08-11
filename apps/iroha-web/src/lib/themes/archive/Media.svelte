<script lang="ts">
  import type {
    MediaAggregates,
    MediaCompletionBucket,
    MediaRow,
    MediaScoreBucket,
  } from "$lib/api";
  import { formatProgressCount, progressPercent } from "$lib/format";
  import { mediaTypeLabel } from "$lib/media";

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

  // The completion-by-year record becomes a shelf core: each year is a
  // stratum, thickness real completion volume, tone the year's share of
  // the total -- the collection's own deposition rate, laid out like a
  // shelf of accession registers, one per year.
  const maxCompletion = $derived(
    Math.max(1, ...completions.map((bucket) => bucket.count)),
  );
  const completionRows = $derived(
    [...completions]
      .sort((a, b) => b.year - a.year)
      .map((bucket) => ({
        bucket,
        magnitude: Math.max(bucket.count / maxCompletion, 0.06),
        pct: (bucket.count / maxCompletion) * 100,
      })),
  );

  function tone(pct: number): string {
    const clamped = Math.max(0, Math.min(100, pct));
    return `color-mix(in srgb, var(--accent-2) ${clamped}%, var(--accent) ${100 - clamped}%)`;
  }
</script>

<section class="folio-media" aria-labelledby="folio-media-title">
  <header class="folio-head">
    <div>
      <p class="folio-kicker">Collection shelf / provenance</p>
      <h1 id="folio-media-title">Held in the collection.</h1>
      <p>
        Reading, watching, and playing as an accessioned holding -- not a
        backlog.
      </p>
    </div>
    <div class="folio-readout">
      <strong>{aggregates.totals.item_count}</strong><span>titles held</span>
    </div>
  </header>

  <nav class="folio-tabs" aria-label="Media family">
    {#each families as option (option.value)}
      <button
        class:active={family === option.value}
        type="button"
        onclick={() => onFamily(option.value)}>{option.label}</button
      >
    {/each}
  </nav>

  <div class="folio-filters">
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

  <div class="folio-stats catalog-card">
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

  {#if completionRows.length}
    <section class="folio-core catalog-card">
      <header>
        <div>
          <p class="folio-kicker">Deposition record</p>
          <h2>Completions by year</h2>
        </div>
        <span>{scores.length} score buckets on file</span>
      </header>
      <div
        class="core-log"
        role="img"
        aria-label="Completions by year, most recent at top"
      >
        <div class="core-strip">
          {#each completionRows as row (row.bucket.year)}
            <div
              class="core-band"
              style={`flex-grow: ${row.magnitude}; background: ${tone(row.pct)};`}
            ></div>
          {/each}
        </div>
        <div class="core-legend">
          {#each completionRows as row (row.bucket.year)}
            <div class="core-row" style={`flex-grow: ${row.magnitude};`}>
              <strong>{row.bucket.year}</strong>
              <span>{row.bucket.count} completed</span>
            </div>
          {/each}
        </div>
      </div>
    </section>
  {/if}

  <section class="folio-shelf">
    <header>
      <div>
        <p class="folio-kicker">Current shelf</p>
        <h2>{items.length} visible titles</h2>
      </div>
      <span
        >{typeFamilies
          .map((item) => `${item.type}: ${item.count}`)
          .join(" · ")}</span
      >
    </header>
    {#if items.length === 0}
      <p class="folio-empty">No titles match this shelf.</p>
    {:else}
      <div class="folio-grid">
        {#each items as item, index (item.id)}
          <a class="folio-card" href={`/library/${item.id}`}>
            <span class="folio-card-tag" title={mediaTypeLabel(item.media_type)}
              >{item.media_type.slice(0, 3).toUpperCase()}-{String(
                index + 1,
              ).padStart(4, "0")}</span
            >
            {#if item.cover_image_url}
              <img src={item.cover_image_url} alt="" loading="lazy" />
            {:else}
              <span class="folio-initial">{item.title.slice(0, 1)}</span>
            {/if}
            <strong>{item.native_title || item.title}</strong>
            <small class="folio-card-type"
              >{mediaTypeLabel(item.media_type)}</small
            >
            <small
              >{item.status || "unknown"} · {formatProgressCount(
                item.position,
                item.total,
                item.unit,
                item.status,
              )}</small
            >
            <i
              ><b
                style={`width: ${progressPercent(item.status, item.position, item.total, item.progress_percent)}%`}
              ></b></i
            >
          </a>
        {/each}
      </div>
    {/if}
  </section>
  {#if hasMore}
    <button
      class="folio-load-more"
      type="button"
      disabled={loadingMore}
      onclick={onLoadMore}>{loadingMore ? "Loading…" : "Load more"}</button
    >
  {/if}
  <footer class="folio-source">
    {completions.length} completion periods · {scores.length} score buckets · source:
    imported provider records
  </footer>
</section>

<style>
  .folio-media {
    display: grid;
    gap: 1.3rem;
  }
  .folio-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.64rem;
    letter-spacing: 0.15em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    font-family: var(--font-serif);
    font-weight: 700;
    letter-spacing: -0.01em;
  }
  h1 {
    max-width: 12ch;
    font-size: clamp(2.5rem, 6.5vw, 5rem);
    line-height: 0.95;
  }
  h2 {
    font-size: 1.55rem;
  }
  .folio-head {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.75rem;
  }
  .folio-head p:last-child {
    color: var(--text-muted);
    line-height: 1.6;
  }
  .folio-readout {
    display: grid;
    justify-items: end;
    color: var(--text-muted);
  }
  .folio-readout strong {
    color: var(--accent);
    font-family: var(--font-serif);
    font-size: 3.2rem;
    font-weight: 700;
    white-space: nowrap;
  }
  .folio-readout span {
    margin-top: 0.5rem;
    font-family: var(--font-mono);
    font-size: 0.64rem;
    text-transform: uppercase;
  }
  .folio-tabs {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
  }
  .folio-tabs button {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 0.45rem 0.85rem;
    background: transparent;
    color: var(--text-muted);
    font: inherit;
    font-family: var(--font-mono);
    font-size: 0.72rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    cursor: pointer;
  }
  .folio-tabs button.active,
  .folio-tabs button:hover {
    border-color: var(--accent);
    color: var(--accent);
  }
  .folio-filters {
    display: flex;
    gap: 0.75rem;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1rem;
  }
  label {
    display: grid;
    gap: 0.3rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
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
  .catalog-card {
    position: relative;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
    padding: 1.5rem 1.5rem 1.5rem 1.7rem;
  }
  .catalog-card::before {
    content: "";
    position: absolute;
    left: -1px;
    top: 1.15rem;
    width: 4px;
    height: 2.3rem;
    background: var(--accent-2);
    border-radius: 0 2px 2px 0;
  }
  .folio-stats {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    padding: 0;
  }
  .folio-stats::before {
    display: none;
  }
  .folio-stats div {
    display: grid;
    gap: 0.35rem;
    border-right: 1px solid var(--border);
    padding: 1.1rem;
  }
  .folio-stats div:last-child {
    border-right: 0;
  }
  .folio-stats span {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.68rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .folio-stats strong {
    font-family: var(--font-serif);
    font-size: 1.5rem;
    font-weight: 700;
  }
  .folio-core header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .folio-core header > span {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.68rem;
    text-align: right;
  }
  .core-log {
    display: flex;
    gap: 1rem;
    height: 14rem;
    margin-top: 1.4rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    overflow: hidden;
  }
  .core-strip {
    display: flex;
    flex-direction: column;
    width: 1.9rem;
    flex-shrink: 0;
  }
  .core-band {
    flex-shrink: 0;
    border-top: 1px solid var(--bg);
  }
  .core-band:first-child {
    border-top: 0;
  }
  .core-legend {
    display: flex;
    flex: 1;
    min-width: 0;
    flex-direction: column;
    overflow-y: auto;
  }
  .core-row {
    display: flex;
    flex-shrink: 0;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    min-height: 1.6rem;
    overflow: hidden;
    border-top: 1px solid var(--border);
    padding: 0 0.9rem 0 0.25rem;
  }
  .core-row:first-child {
    border-top: 0;
  }
  .core-row strong {
    font-family: var(--font-serif);
    font-size: 0.85rem;
    font-weight: 700;
  }
  .core-row span {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.68rem;
  }
  .folio-shelf {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 1.5rem;
    background: color-mix(in srgb, var(--surface) 90%, transparent);
  }
  .folio-shelf > header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .folio-shelf > header > span {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.7rem;
    text-align: right;
  }
  .folio-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(9rem, 1fr));
    gap: 1rem;
    margin-top: 1.25rem;
  }
  .folio-card {
    position: relative;
    display: grid;
    gap: 0.35rem;
    color: var(--text);
    text-decoration: none;
  }
  .folio-card-tag {
    position: absolute;
    top: 0.4rem;
    left: 0.4rem;
    z-index: 1;
    border: 1px solid color-mix(in srgb, var(--accent) 60%, transparent);
    border-radius: 2px;
    padding: 0.1rem 0.35rem;
    background: color-mix(in srgb, var(--bg) 55%, transparent);
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.58rem;
    letter-spacing: 0.03em;
  }
  .folio-card img,
  .folio-initial {
    display: block;
    width: 100%;
    aspect-ratio: 3 / 4;
    object-fit: cover;
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--accent) 12%, var(--surface));
  }
  .folio-initial {
    display: grid;
    place-items: center;
    color: var(--accent);
    font-family: var(--font-serif);
    font-size: 2.4rem;
    font-weight: 700;
  }
  .folio-card strong {
    overflow: hidden;
    font-family: var(--font-serif);
    font-size: 0.88rem;
    font-weight: 700;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .folio-card small {
    overflow: hidden;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.64rem;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .folio-card i {
    display: block;
    height: 0.2rem;
    background: var(--border);
    font-style: normal;
  }
  .folio-card i b {
    display: block;
    height: 100%;
    background: var(--accent-2);
  }
  .folio-empty {
    color: var(--text-muted);
  }
  .folio-load-more {
    justify-self: center;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    padding: 0.55rem 1.1rem;
    background: transparent;
    color: var(--accent);
    font: inherit;
    font-family: var(--font-mono);
    font-size: 0.75rem;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    cursor: pointer;
  }
  .folio-load-more:disabled {
    opacity: 0.5;
  }
  .folio-source {
    border-top: 1px solid var(--border);
    padding-top: 0.8rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.66rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  @media (max-width: 680px) {
    .folio-head,
    .folio-shelf > header,
    .folio-core header {
      display: block;
    }
    .folio-readout {
      display: block;
      margin-top: 1.5rem;
    }
    .folio-shelf > header > span,
    .folio-core header > span {
      display: block;
      margin-top: 0.8rem;
      text-align: left;
    }
    .folio-filters,
    .folio-stats {
      grid-template-columns: 1fr;
    }
    .folio-filters {
      display: grid;
    }
    .folio-stats div {
      border-right: 0;
      border-bottom: 1px solid var(--border);
    }
    .folio-stats div:last-child {
      border-bottom: 0;
    }
  }
</style>
