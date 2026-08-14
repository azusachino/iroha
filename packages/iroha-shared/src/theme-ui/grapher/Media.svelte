<script lang="ts">
  import type { MediaThemeProps } from "../../media";
  import MediaBarChart from "../components/MediaBarChart.svelte";
  import MediaAssetCard from "../components/MediaAssetCard.svelte";

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
    activeCount,
    theme,
    onFamily,
    onStatus,
    onYear,
    onLoadMore,
    hasMore,
    loadingMore,
  }: MediaThemeProps = $props();

  const families = [
    { value: "", label: "All" },
    { value: "anime", label: "Anime" },
    { value: "manga_book", label: "Manga & books" },
    { value: "game", label: "Games" },
  ];
</script>

<section class="grapher-media" aria-labelledby="grapher-media-title">
  <header class="media-header">
    <div>
      <p class="kicker">Library / distributions</p>
      <h1 id="grapher-media-title">The attention record.</h1>
      <p>
        Compare completion, score, and kind before opening the exact shelf rows.
      </p>
    </div>
    <strong>{aggregates.totals.item_count}<small> titles</small></strong>
  </header>
  <nav class="tabs" aria-label="Media family">
    {#each families as option (option.value)}<button
        class:active={family === option.value}
        type="button"
        onclick={() => onFamily(option.value)}>{option.label}</button
      >{/each}
  </nav>
  <div class="filters">
    <label
      >Status<select value={status} onchange={onStatus}
        ><option value="">All statuses</option><option value="in_progress"
          >In progress</option
        ><option value="completed">Completed</option><option value="planned"
          >Planned</option
        ><option value="abandoned">Abandoned</option></select
      ></label
    >
    <label
      >Completed year<select value={completedYear} onchange={onYear}
        ><option value="">Lifetime</option
        >{#each yearOptions as option (option.year)}<option value={option.year}
            >{option.year}</option
          >{/each}</select
      ></label
    >
  </div>

  <div class="chart-grid" aria-label="Library charts">
    <article class="chart-panel">
      <header>
        <p class="kicker">Time</p>
        <h2>Completions by year</h2>
      </header>
      {#if completions.length}<MediaBarChart
          labels={completions.map((bucket) => bucket.year)}
          values={completions.map((bucket) => bucket.count)}
          color="--accent"
        />{:else}<p class="empty">No completion records.</p>{/if}
    </article>
    <article class="chart-panel">
      <header>
        <p class="kicker">Score</p>
        <h2>Score distribution</h2>
      </header>
      {#if scores.length}<MediaBarChart
          labels={scores.map((bucket) => bucket.score)}
          values={scores.map((bucket) => bucket.count)}
          color="--accent-2"
        />{:else}<p class="empty">No ratings recorded.</p>{/if}
    </article>
    <article class="chart-panel kind-panel">
      <header>
        <p class="kicker">Composition</p>
        <h2>By kind</h2>
      </header>
      {#if typeFamilies.length}<MediaBarChart
          labels={typeFamilies.map((item) => item.type)}
          values={typeFamilies.map((item) => item.count)}
          color="--mark-teal"
          horizontal
        />{:else}<p class="empty">No kind breakdown.</p>{/if}
    </article>
  </div>

  <div class="stats">
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
    <div>
      <span>In progress</span><strong>{activeCount}</strong>
    </div>
  </div>
  <section class="records">
    <header>
      <div>
        <p class="kicker">Exact records</p>
        <h2>{items.length} visible titles</h2>
      </div>
      <span>Chart values stay above the rows.</span>
    </header>
    {#if items.length}<div class="record-grid">
        {#each items as item (item.id)}
          <MediaAssetCard {item} {theme} />
        {/each}
      </div>{:else}<p class="empty">No titles match this selection.</p>{/if}
  </section>
  {#if hasMore}<button
      class="load-more"
      type="button"
      disabled={loadingMore}
      onclick={onLoadMore}>{loadingMore ? "Loading…" : "Load more"}</button
    >{/if}
  <footer>
    {completions.length} completion periods · {scores.length} score buckets · source:
    imported provider records
  </footer>
</section>

<style>
  .grapher-media {
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
    max-width: 12ch;
    font-family: var(--font-sans);
    font-size: clamp(3rem, 8vw, 7rem);
    letter-spacing: -0.12em;
    line-height: 0.82;
  }
  h2 {
    font-family: var(--font-sans);
    font-size: 1.1rem;
    letter-spacing: -0.04em;
  }
  .kicker {
    margin-bottom: 0.45rem;
    color: var(--accent);
    font-size: 0.64rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .media-header {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 3px solid var(--text);
    padding-bottom: 1.5rem;
  }
  .media-header p:last-child {
    max-width: 40rem;
    margin-top: 1rem;
    color: var(--text-muted);
    font-family: var(--font-sans);
    line-height: 1.55;
  }
  .media-header > strong {
    color: var(--accent);
    font-family: var(--font-sans);
    font-size: 3.5rem;
    letter-spacing: -0.1em;
    white-space: nowrap;
  }
  .media-header small {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.7rem;
    letter-spacing: 0;
  }
  .tabs {
    display: flex;
    flex-wrap: wrap;
    gap: 0.35rem;
    border-bottom: 1px solid var(--border);
    padding-bottom: 0.7rem;
  }
  button {
    border: 1px solid var(--border);
    border-radius: 0;
    padding: 0.45rem 0.7rem;
    background: transparent;
    color: var(--text-muted);
    font: inherit;
    font-size: 0.7rem;
    cursor: pointer;
  }
  button.active,
  button:hover {
    border-color: var(--accent);
    color: var(--accent);
  }
  .filters {
    display: flex;
    flex-wrap: wrap;
    gap: 0.8rem;
  }
  label {
    display: grid;
    gap: 0.3rem;
    color: var(--text-muted);
    font-size: 0.64rem;
    text-transform: uppercase;
  }
  select {
    min-width: 9rem;
    border-radius: 0;
    font: inherit;
    font-size: 0.75rem;
  }
  .chart-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
  }
  .chart-panel,
  .records {
    border: 1px solid var(--border);
    background: var(--surface);
    padding: 1rem;
  }
  .chart-panel {
    border-top: 4px solid var(--accent);
  }
  .chart-panel header {
    margin-bottom: 0.5rem;
  }
  .kind-panel {
    grid-column: 1 / -1;
  }
  .stats {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    border-block: 1px solid var(--border);
  }
  .stats div {
    display: grid;
    gap: 0.35rem;
    padding: 0.8rem;
    border-right: 1px solid var(--border);
  }
  .stats div:last-child {
    border: 0;
  }
  .stats span,
  .records header > span,
  footer,
  .empty {
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  .stats strong {
    font-family: var(--font-sans);
    font-size: 1.35rem;
    letter-spacing: -0.06em;
  }
  .records header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: end;
  }
  .record-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0 1rem;
    margin-top: 0.75rem;
  }
  .load-more {
    width: 100%;
  }
  footer {
    border-top: 1px solid var(--border);
    padding-top: 0.7rem;
  }
  @media (max-width: 700px) {
    .media-header {
      display: grid;
    }
    .chart-grid,
    .record-grid {
      grid-template-columns: 1fr;
    }
    .kind-panel {
      grid-column: auto;
    }
    .stats {
      grid-template-columns: 1fr;
    }
    .stats div {
      border-right: 0;
      border-bottom: 1px solid var(--border);
    }
    .stats div:last-child {
      border-bottom: 0;
    }
  }
</style>
