<script lang="ts">
  import { onMount } from "svelte";
  import {
    getMediaAggregates,
    listMedia,
    type MediaAggregates,
    type MediaRow,
  } from "$lib/api";
  import StatTile from "$lib/components/StatTile.svelte";

  let aggregates = $state<MediaAggregates | null>(null);
  let items = $state<MediaRow[]>([]);
  let nextCursor = $state<string | null>(null);
  let hasMore = $state(false);
  let loadingMore = $state(false);
  let loading = $state(true);
  let error = $state<string | null>(null);

  // Only actively-in-progress items belong in the "continue" strip; paused /
  // on-hold entries keep status=in_progress but carry hidden_from_continue.
  const isContinuing = (item: MediaRow) =>
    item.status === "in_progress" && !item.hidden_from_continue;
  const continueItems = $derived(items.filter(isContinuing).slice(0, 6));
  const groupedItems = $derived(
    items
      .filter((item) => !isContinuing(item))
      .reduce(
        (groups, item) => {
          // Paused items share the in_progress status; give them their own shelf.
          const key =
            item.status === "in_progress" ? "paused" : item.status || "unknown";
          (groups[key] ??= []).push(item);
          return groups;
        },
        {} as Record<string, MediaRow[]>,
      ),
  );
  const completionMax = $derived(
    Math.max(
      ...(aggregates?.completions_by_year.map((bucket) => bucket.count) ?? [1]),
      1,
    ),
  );
  const scoreMax = $derived(
    Math.max(
      ...(aggregates?.score_distribution.map((bucket) => bucket.count) ?? [1]),
      1,
    ),
  );
  const typeMax = $derived(
    Math.max(
      ...(aggregates?.type_split.map((bucket) => bucket.count) ?? [1]),
      1,
    ),
  );

  onMount(() => {
    void load();
  });

  async function load() {
    loading = true;
    error = null;
    try {
      const [nextAggregates, page] = await Promise.all([
        getMediaAggregates(),
        listMedia({ limit: 100 }),
      ]);
      aggregates = nextAggregates;
      items = page.items;
      nextCursor = page.next_cursor;
      hasMore = page.has_more;
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    } finally {
      loading = false;
    }
  }

  async function loadMore() {
    if (!nextCursor || loadingMore) return;
    loadingMore = true;
    try {
      const page = await listMedia({ limit: 100, cursor: nextCursor });
      items = [...items, ...page.items];
      nextCursor = page.next_cursor;
      hasMore = page.has_more;
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    } finally {
      loadingMore = false;
    }
  }

  function mediaKind(type: string): string {
    if (
      type.includes("manga") ||
      type.includes("novel") ||
      type.includes("book")
    ) {
      return "Reading";
    }
    if (type.includes("anime")) return "Watching";
    return type.replaceAll("_", " ");
  }

  function statusLabel(status: string): string {
    return status.replaceAll("_", " ");
  }

  function progressValue(item: MediaRow): number {
    if (item.progress_percent != null)
      return Math.min(Math.max(item.progress_percent, 0), 100);
    if (item.position != null && item.total) {
      return Math.min(Math.max((item.position / item.total) * 100, 0), 100);
    }
    return 0;
  }

  function scoreFor(item: MediaRow): string {
    return item.rating == null ? "—" : item.rating.toFixed(1);
  }
</script>

<svelte:head>
  <title>Media · iroha</title>
</svelte:head>

<section class="media-shell">
  <header class="domain-header">
    <div>
      <p class="eyebrow">Media domain</p>
      <h1>Stories that stay with you.</h1>
      <p class="muted">
        A quiet index of what you watch, read, finish, and remember.
      </p>
    </div>
  </header>

  {#if loading}
    <p class="muted">Loading media history…</p>
  {:else if error}
    <p class="error">Failed to load media: {error}</p>
  {:else if aggregates}
    <div class="stat-strip" aria-label="Media summary">
      <StatTile
        label="Library"
        value={aggregates.totals.item_count.toLocaleString()}
        sub="Tracked items"
      />
      <StatTile
        label="Completed"
        value={aggregates.totals.completed_count.toLocaleString()}
        sub={`${aggregates.totals.this_year_completed} this year`}
      />
      <StatTile
        label="Average score"
        value={aggregates.totals.average_rating
          ? aggregates.totals.average_rating.toFixed(1)
          : "—"}
        sub="Normalized to 10"
      />
      <StatTile
        label="In progress"
        value={continueItems.length.toLocaleString()}
        sub="Watching or reading"
      />
    </div>

    <div class="analytics-grid">
      <section class="chart-card tile">
        <div class="section-heading">
          <div>
            <p class="eyebrow">Momentum</p>
            <h2>Completions by year</h2>
          </div>
          <span class="chart-total">{aggregates.totals.completed_count}</span>
        </div>
        {#if aggregates.completions_by_year.length}
          <div class="year-bars" aria-label="Completions by year">
            {#each aggregates.completions_by_year as bucket}
              <div class="year-bar">
                <span class="bar-value">{bucket.count}</span>
                <div class="bar-track">
                  <span
                    style={`height: ${(bucket.count / completionMax) * 100}%`}
                  ></span>
                </div>
                <span class="bar-label">{bucket.year}</span>
              </div>
            {/each}
          </div>
        {:else}
          <p class="empty-copy">No completed items yet.</p>
        {/if}
      </section>

      <section class="chart-card tile">
        <div class="section-heading">
          <div>
            <p class="eyebrow">Taste</p>
            <h2>Score distribution</h2>
          </div>
          <span class="chart-total">0–10</span>
        </div>
        {#if aggregates.score_distribution.length}
          <div class="score-bars" aria-label="Score distribution">
            {#each aggregates.score_distribution as bucket}
              <div class="score-bar">
                <span style={`height: ${(bucket.count / scoreMax) * 100}%`}
                ></span>
                <small>{bucket.score}</small>
              </div>
            {/each}
          </div>
        {:else}
          <p class="empty-copy">Ratings will appear here as you score items.</p>
        {/if}
      </section>

      <section class="chart-card type-card tile">
        <div class="section-heading">
          <div>
            <p class="eyebrow">Shape of the library</p>
            <h2>By kind</h2>
          </div>
        </div>
        {#if aggregates.type_split.length}
          <div class="type-list">
            {#each aggregates.type_split as bucket}
              <div class="type-row">
                <div class="type-meta">
                  <span>{mediaKind(bucket.type)}</span><strong
                    >{bucket.count}</strong
                  >
                </div>
                <div class="type-track">
                  <span style={`width: ${(bucket.count / typeMax) * 100}%`}
                  ></span>
                </div>
              </div>
            {/each}
          </div>
        {:else}
          <p class="empty-copy">Your collection will take shape here.</p>
        {/if}
      </section>
    </div>

    <section class="continue-section">
      <div class="section-heading">
        <div>
          <p class="eyebrow">Keep going</p>
          <h2>Watching & reading</h2>
        </div>
        <span class="muted">{continueItems.length} active</span>
      </div>
      {#if continueItems.length}
        <div class="continue-grid">
          {#each continueItems as item (item.id)}
            <a
              class="continue-card tile tile-interactive"
              href={`/media/${item.id}`}
            >
              {#if item.cover_image_url}
                <img src={item.cover_image_url} alt="" loading="lazy" />
              {:else}
                <div class="cover-placeholder" aria-hidden="true">
                  {item.title.slice(0, 1)}
                </div>
              {/if}
              <div class="continue-copy">
                <span class="card-kicker">{mediaKind(item.media_type)}</span>
                <h3>{item.title}</h3>
                <div class="progress-track">
                  <span style={`width: ${progressValue(item)}%`}></span>
                </div>
                <span class="progress-label"
                  >{Math.round(progressValue(item))}% complete</span
                >
              </div>
            </a>
          {/each}
        </div>
      {:else}
        <div class="empty-panel tile">Nothing in progress right now.</div>
      {/if}
    </section>

    <section class="collection-section">
      <div class="section-heading">
        <div>
          <p class="eyebrow">Collection</p>
          <h2>Everything in the index</h2>
        </div>
        <span class="muted">{items.length} shown</span>
      </div>
      {#if Object.keys(groupedItems).length}
        {#each Object.entries(groupedItems) as [status, group]}
          <div class="status-group">
            <div class="status-heading">
              <h3>{statusLabel(status)}</h3>
              <span>{group.length}</span>
            </div>
            <div class="cover-grid">
              {#each group as item (item.id)}
                <a
                  class="media-card tile tile-interactive"
                  href={`/media/${item.id}`}
                >
                  {#if item.cover_image_url}
                    <img src={item.cover_image_url} alt="" loading="lazy" />
                  {:else}
                    <div class="cover-placeholder" aria-hidden="true">
                      {item.title.slice(0, 1)}
                    </div>
                  {/if}
                  <div class="media-card-copy">
                    <h3>{item.title}</h3>
                    <span>{mediaKind(item.media_type)} · {scoreFor(item)}</span>
                  </div>
                </a>
              {/each}
            </div>
          </div>
        {/each}
      {:else}
        <div class="empty-panel tile">No media items found.</div>
      {/if}
      {#if hasMore}
        <button class="load-more" onclick={loadMore} disabled={loadingMore}>
          {loadingMore ? "Loading…" : "Load more"}
        </button>
      {/if}
    </section>
  {/if}
</section>

<style>
  .media-shell {
    display: grid;
    gap: 1.5rem;
  }
  .stat-strip {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 0.75rem;
  }
  .analytics-grid {
    display: grid;
    grid-template-columns: 1.2fr 1fr 0.9fr;
    gap: 0.75rem;
  }
  .chart-card {
    min-height: 14.5rem;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 1.1rem;
  }
  .section-heading {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1rem;
  }
  h2,
  h3,
  p {
    margin: 0;
  }
  h2 {
    font-size: 1.05rem;
    letter-spacing: -0.02em;
  }
  h3 {
    font-size: 0.9rem;
  }
  .eyebrow,
  .card-kicker {
    color: var(--text-muted);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.09em;
    text-transform: uppercase;
  }
  .chart-total {
    color: var(--accent);
    font-size: 0.8rem;
    font-weight: 750;
  }
  .year-bars {
    min-height: 9rem;
    display: flex;
    align-items: end;
    gap: clamp(0.45rem, 2vw, 1rem);
    padding: 0 0.2rem;
    border-bottom: 1px solid var(--border);
  }
  .year-bar {
    flex: 1;
    min-width: 0;
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: end;
    gap: 0.35rem;
  }
  .bar-track {
    height: 7rem;
    width: min(2.25rem, 100%);
    display: flex;
    align-items: end;
  }
  .bar-track span {
    width: 100%;
    min-height: 0.2rem;
    border-radius: 4px 4px 0 0;
    background: linear-gradient(
      180deg,
      var(--accent),
      color-mix(in srgb, var(--accent), var(--accent-2) 35%)
    );
  }
  .bar-value {
    color: var(--text);
    font-size: 0.72rem;
    font-weight: 700;
  }
  .bar-label,
  .score-bar small {
    color: var(--text-muted);
    font-size: 0.65rem;
  }
  .score-bars {
    min-height: 9rem;
    display: flex;
    align-items: end;
    gap: 0.35rem;
    padding: 0 0.2rem;
    border-bottom: 1px solid var(--border);
  }
  .score-bar {
    flex: 1;
    min-width: 0;
    height: 8rem;
    display: flex;
    align-items: center;
    flex-direction: column;
    justify-content: end;
    gap: 0.35rem;
  }
  .score-bar span {
    width: 100%;
    min-height: 0.2rem;
    border-radius: 3px 3px 0 0;
    background: var(--mark-magenta);
    opacity: 0.8;
  }
  .type-list {
    display: grid;
    gap: 1rem;
    margin-top: 0.35rem;
  }
  .type-row {
    display: grid;
    gap: 0.4rem;
  }
  .type-meta {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    font-size: 0.82rem;
  }
  .type-meta strong {
    color: var(--text-muted);
  }
  .type-track,
  .progress-track {
    height: 0.4rem;
    overflow: hidden;
    border-radius: 999px;
    background: var(--surface-2);
  }
  .type-track span,
  .progress-track span {
    display: block;
    height: 100%;
    border-radius: inherit;
    background: var(--mark-teal);
  }
  .continue-section,
  .collection-section {
    display: grid;
    gap: 0.85rem;
  }
  .continue-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 0.75rem;
  }
  .continue-card {
    min-width: 0;
    padding: 0.65rem;
    display: flex;
    gap: 0.75rem;
    color: var(--text);
  }
  .continue-card img,
  .continue-card .cover-placeholder {
    width: 3.4rem;
    height: 4.8rem;
    flex: 0 0 auto;
  }
  img {
    display: block;
    object-fit: cover;
    background: var(--surface-2);
  }
  .cover-placeholder {
    display: grid;
    place-items: center;
    border: 1px solid var(--border);
    background: linear-gradient(145deg, var(--surface-2), var(--surface));
    color: var(--accent);
    font-size: 1.5rem;
    font-weight: 800;
  }
  .continue-copy {
    min-width: 0;
    display: grid;
    align-content: center;
    gap: 0.42rem;
  }
  .continue-copy h3,
  .media-card-copy h3 {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .progress-label,
  .media-card-copy span {
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  .collection-section {
    padding-top: 0.5rem;
  }
  .status-group {
    display: grid;
    gap: 0.65rem;
  }
  .status-heading {
    display: flex;
    align-items: baseline;
    gap: 0.5rem;
  }
  .status-heading h3 {
    text-transform: capitalize;
  }
  .status-heading span {
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .cover-grid {
    display: grid;
    grid-template-columns: repeat(6, minmax(0, 1fr));
    gap: 0.75rem;
  }
  .media-card {
    min-width: 0;
    overflow: hidden;
    color: var(--text);
  }
  .media-card img,
  .media-card .cover-placeholder {
    width: 100%;
    aspect-ratio: 2 / 3;
  }
  .media-card-copy {
    min-width: 0;
    padding: 0.65rem;
    display: grid;
    gap: 0.3rem;
  }
  .media-card-copy h3 {
    font-size: 0.78rem;
  }
  .empty-panel,
  .empty-copy {
    color: var(--text-muted);
  }
  .empty-panel {
    padding: 1rem;
  }
  .load-more {
    justify-self: center;
    padding: 0.55rem 1.4rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--surface);
    color: var(--text);
    font-size: 0.8rem;
    font-weight: 650;
    cursor: pointer;
  }
  .load-more:hover:not(:disabled) {
    border-color: var(--accent);
    color: var(--accent);
  }
  .load-more:disabled {
    opacity: 0.6;
    cursor: default;
  }
  .error {
    color: var(--danger);
  }
  @media (max-width: 900px) {
    .analytics-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .type-card {
      grid-column: span 2;
    }
    .cover-grid {
      grid-template-columns: repeat(4, minmax(0, 1fr));
    }
  }
  @media (max-width: 640px) {
    .stat-strip,
    .analytics-grid,
    .continue-grid {
      grid-template-columns: 1fr 1fr;
    }
    .type-card {
      grid-column: span 2;
    }
    .cover-grid {
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 0.55rem;
    }
  }
</style>
