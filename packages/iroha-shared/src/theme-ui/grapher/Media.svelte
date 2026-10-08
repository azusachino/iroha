<script lang="ts">
  import type { MediaThemeProps } from "../../domain/media";
  import MediaBarChart from "../components/MediaBarChart.svelte";
  import MediaAssetCard from "../components/MediaAssetCard.svelte";
  import RetryNotice from "../components/RetryNotice.svelte";

  let {
    items,
    aggregates,
    aggregatesLoading,
    aggregatesReady,
    aggregatesError,
    aggregatesScope,
    recordsLoading,
    recordsReady,
    recordsError,
    recordsScope,
    onRetryAggregates,
    onRetryRecords,
    family,
    status,
    completedYear,
    yearOptions,
    typeFamilies,
    completions,
    scores,
    currentCompletedCount,
    activeCount,
    activeScope,
    theme,
    onFamily,
    onStatus,
    onYear,
    onLoadMore,
    hasMore,
    loadingMore,
    loadMoreError = null,
  }: MediaThemeProps = $props();

  let totalsPanel = $state<HTMLElement>();
  let recordsPanel = $state<HTMLElement>();
  let selectedYear = $state("");
  let yearSelect = $state<HTMLSelectElement>();
  $effect(() => {
    selectedYear = completedYear;
    if (yearSelect && yearSelect.value !== completedYear) {
      yearSelect.value = completedYear;
    }
  });

  const families = [
    { value: "", label: "All" },
    { value: "anime", label: "Anime" },
    { value: "manga_book", label: "Manga & light novels" },
    { value: "book", label: "Books" },
    { value: "game", label: "Games" },
  ];
</script>

<section class="grapher-media" aria-labelledby="grapher-media-title">
  <header class="domain-header" role="region" aria-label="Library header">
    <div class="header-copy">
      <h1 id="grapher-media-title">Library</h1>
      <p>Watch, reading, and game history</p>
    </div>
  </header>
  <nav class="tabs" aria-label="Media family">
    {#each families as option (option.value)}<button
        class:active={family === option.value}
        aria-pressed={family === option.value}
        type="button"
        onclick={() => onFamily(option.value)}>{option.label}</button
      >{/each}
  </nav>
  <div class="filters">
    <label
      >Status<select
        value={status}
        onchange={(event) =>
          onStatus((event.currentTarget as HTMLSelectElement).value)}
        ><option value="">All statuses</option><option value="in_progress"
          >In progress</option
        ><option value="completed">Completed</option><option value="planned"
          >Planned</option
        ><option value="abandoned">Abandoned</option></select
      ></label
    >
    <label
      >Completed year<select
        bind:this={yearSelect}
        bind:value={selectedYear}
        onchange={(event) =>
          onYear((event.currentTarget as HTMLSelectElement).value)}
        ><option value="">Lifetime</option
        >{#each yearOptions as option (option.year)}<option value={option.year}
            >{option.year}</option
          >{/each}</select
      ></label
    >
  </div>

  <section
    aria-label="Library totals read state"
    tabindex="-1"
    bind:this={totalsPanel}
  >
    {#if aggregatesError}<RetryNotice
        message={`Could not load library totals: ${aggregatesError}`}
        retryLabel="Retry library totals"
        onRetry={onRetryAggregates}
        focusTarget={totalsPanel}
      />{/if}
    {#if aggregatesLoading}<p role="status">
        {aggregatesReady
          ? "Updating library totals…"
          : "Loading library totals…"}
      </p>{/if}
    {#if aggregatesScope}<p>Observed totals: {aggregatesScope}</p>{/if}
    <p class="muted">
      <strong class="media-count"
        ><span>{aggregates?.totals.item_count ?? "—"}</span> titles</strong
      >
    </p>
  </section>
  <div class="chart-grid" aria-label="Library charts">
    <article class="chart-panel">
      <header>
        <p class="kicker">Time</p>
        <h2>Completions by year</h2>
      </header>
      {#if completions.length}<MediaBarChart
          title="Completions by year"
          labels={completions.map((bucket) => bucket.year)}
          values={completions.map((bucket) => bucket.count)}
          color="--accent"
        />{:else if aggregatesReady}<p class="empty">
          No completion records.
        </p>{:else}<p class="empty">Completion history unavailable.</p>{/if}
    </article>
    <article class="chart-panel">
      <header>
        <p class="kicker">Score</p>
        <h2>Score distribution</h2>
      </header>
      {#if scores.length}<MediaBarChart
          title="Score distribution"
          labels={scores.map((bucket) => bucket.score)}
          values={scores.map((bucket) => bucket.count)}
          color="--accent-2"
        />{:else if aggregatesReady}<p class="empty">
          No ratings recorded.
        </p>{:else}<p class="empty">Ratings unavailable.</p>{/if}
    </article>
    <article class="chart-panel kind-panel">
      <header>
        <p class="kicker">Composition</p>
        <h2>By kind</h2>
      </header>
      {#if typeFamilies.length}<MediaBarChart
          title="By kind"
          labels={typeFamilies.map((item) => item.type)}
          values={typeFamilies.map((item) => item.count)}
          color="--mark-teal"
          horizontal
        />{:else if aggregatesReady}<p class="empty">
          No kind breakdown.
        </p>{:else}<p class="empty">Kind breakdown unavailable.</p>{/if}
    </article>
  </div>

  <div class="stats" role="region" aria-label="Library summary">
    <div>
      <span>Completed</span><strong>{currentCompletedCount ?? "—"}</strong>
    </div>
    <div>
      <span>This year</span><strong
        >{aggregates?.totals.this_year_completed ?? "—"}</strong
      >
    </div>
    <div>
      <span>Average score</span><strong
        >{aggregates && scores.length
          ? aggregates.totals.average_rating.toFixed(1)
          : "—"}</strong
      >
    </div>
    <div>
      <span>In progress</span><strong>{activeCount ?? "—"}</strong>
      <small
        >Counts across statuses · {activeScope
          ? `Observed facet: ${activeScope}`
          : "Facet unavailable"}</small
      >
    </div>
  </div>
  <section
    class="records"
    aria-label="Library records read state"
    tabindex="-1"
    bind:this={recordsPanel}
  >
    {#if recordsError}<RetryNotice
        message={`Could not load library records: ${recordsError}`}
        retryLabel="Retry library records"
        onRetry={onRetryRecords}
        focusTarget={recordsPanel}
      />{/if}
    {#if recordsLoading}<p role="status">
        {recordsReady
          ? "Updating library records…"
          : "Loading library records…"}
      </p>{/if}
    {#if recordsScope}<p>Observed records: {recordsScope}</p>{/if}
    <header>
      <div>
        <p class="kicker">Exact records</p>
        <h2>{recordsReady ? items.length : "—"} visible titles</h2>
      </div>
      <span>Chart values stay above the rows.</span>
    </header>
    {#if items.length}<div class="record-grid">
        {#each items as item (item.id)}
          <MediaAssetCard {item} {theme} />
        {/each}
      </div>{:else if recordsReady && !recordsLoading && !recordsError}<p
        class="empty"
      >
        No titles match this selection.
      </p>{/if}
    {#if loadMoreError}
      <RetryNotice
        message={`Could not load the next library page: ${loadMoreError}. Observed titles and cursor are retained.`}
        retryLabel="Retry more titles"
        onRetry={onLoadMore}
        focusTarget={recordsPanel}
      />
    {/if}
    {#if hasMore}<button
        class="load-more"
        type="button"
        disabled={loadingMore}
        onclick={onLoadMore}>{loadingMore ? "Loading…" : "Load more"}</button
      >{/if}
  </section>
  <footer>
    {aggregatesReady ? completions.length : "—"} completion periods · {aggregatesReady
      ? scores.length
      : "—"} score buckets · source: imported provider records
  </footer>
</section>

<style>
  .grapher-media {
    display: grid;
    gap: 1rem;
    font-family: var(--font-mono);
    min-width: 0;
  }
  .grapher-media > * {
    min-width: 0;
  }
  h1,
  h2,
  p {
    margin: 0;
  }
  h1 {
    font-family: var(--font-sans);
    font-size: var(--grapher-utility-title-size);
    letter-spacing: -0.05em;
    line-height: 1;
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
  .domain-header {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    padding-bottom: 1.25rem;
    border-bottom: 1px solid var(--border);
    font-family: var(--font-sans);
  }
  .header-copy h1 {
    margin: 0;
    font-family: var(--font-sans);
    font-size: var(--type-title);
    line-height: 1.25;
  }
  .header-copy p {
    margin: 0.25rem 0 0;
    font-family: var(--font-sans);
    font-size: var(--type-caption);
    color: var(--text-muted);
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
    border-radius: 999px;
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
    border-radius: var(--radius, 14px);
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
    border-radius: var(--radius, 14px);
    background: var(--tile-surface, var(--surface));
    box-shadow: var(--tile-shadow);
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
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 0.65rem;
  }
  .stats div {
    display: grid;
    gap: 0.35rem;
    min-width: 0;
    padding: 0.8rem;
    border: 1px solid var(--border);
    border-radius: var(--radius, 14px);
    background: var(--tile-surface, var(--surface));
    box-shadow: var(--tile-shadow);
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
  @media (max-width: 768px) {
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
