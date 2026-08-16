<script lang="ts">
  import { mediaTypeColor, mediaTypeLabel } from "$lib/media";
  import StatTile from "@iroha/shared/StatTile.svelte";
  import MediaBarChart from "@iroha/shared/theme-ui/components/MediaBarChart.svelte";
  import RouteIntro from "$lib/components/RouteIntro.svelte";
  import { useTheme } from "$lib/themes/context.svelte";
  import ThemeRouteRenderer from "@iroha/shared/theme-ui/ThemeRouteRenderer.svelte";
  import { hasThemeRoute } from "$lib/themes/registry";
  import LoadingBoundary from "$lib/components/LoadingBoundary.svelte";
  import { createLibraryState } from "./library-state.svelte";

  const theme = useTheme();
  const l = createLibraryState();
</script>

<svelte:head>
  <title>Library · iroha</title>
</svelte:head>

<section class="media-shell">
  {#if hasThemeRoute(theme.definition(), "media")}
    <LoadingBoundary
      resource={l.libraryResource}
      preserveLayout
      label="Loading media history…"
    >
      {#if l.libraryResource.error}
        <p class="error" aria-live="assertive">
          {#if l.aggregates}
            Could not update media; showing the previous result: {l
              .libraryResource.error}
          {:else}
            Failed to load media: {l.libraryResource.error}
          {/if}
        </p>
      {/if}
      <ThemeRouteRenderer
        route="media"
        props={{
          items: l.items,
          aggregates: l.aggregatesForView,
          family: l.family,
          status: l.status,
          completedYear: l.completedYear,
          yearOptions: l.yearOptions,
          typeFamilies: l.typeFamilies,
          completions: l.completions,
          scores: l.scores,
          currentCompletedCount:
            l.aggregatesForView.totals.current_completed_count,
          activeCount: l.activeCount,
          onFamily: l.selectFamily,
          onStatus: l.selectStatus,
          onYear: l.selectYear,
          onLoadMore: l.loadMore,
          hasMore: l.hasMore,
          loadingMore: l.loadingMore,
        }}
      />
    </LoadingBoundary>
  {:else}
    <RouteIntro
      eyebrow="Library / things in orbit"
      title="A living personal library."
      description="Keep reading, watching, and playing visible without turning your interests into a backlog."
    />

    <div class="filter-bar" role="tablist" aria-label="Filter by kind">
      {#each l.FAMILIES as f (f.value)}
        <button
          type="button"
          class="chip"
          class:active={l.family === f.value}
          role="tab"
          aria-selected={l.family === f.value}
          aria-label={`Filter media by ${f.label}`}
          onclick={() => l.selectFamily(f.value)}
        >
          {f.label}
        </button>
      {/each}
    </div>

    <div class="filter-options" aria-label="Media filters">
      <label>
        <span>Status</span>
        <select
          value={l.status}
          onchange={(event) =>
            l.selectStatus((event.currentTarget as HTMLSelectElement).value)}
        >
          <option value="">All statuses</option>
          <option value="in_progress">In progress</option>
          <option value="completed">Completed</option>
          <option value="planned">Planned</option>
          <option value="abandoned">Abandoned</option>
        </select>
      </label>
      <label>
        <span>Completed year</span>
        <select
          bind:this={l.yearSelect}
          bind:value={l.selectedYear}
          onchange={(event) =>
            l.selectYear((event.currentTarget as HTMLSelectElement).value)}
        >
          <option value="">Lifetime</option>
          {#each l.yearOptions as option (option.year)}
            <option value={option.year}>{option.year}</option>
          {/each}
        </select>
      </label>
    </div>

    {#if !l.aggregates && l.libraryResource.loading}
      <p class="muted" aria-live="polite">Loading media history…</p>
    {:else if !l.aggregates && l.libraryResource.error}
      <p class="error" aria-live="assertive">
        Failed to load media: {l.libraryResource.error}
      </p>
    {:else if l.aggregates}
      <LoadingBoundary
        resource={l.libraryResource}
        label="Loading media history…"
      >
        {#if l.libraryResource.error}
          <p class="error" aria-live="assertive">
            Could not update media; showing the previous result: {l
              .libraryResource.error}
          </p>
        {/if}
        <div class="stat-strip">
          <StatTile
            label="Library"
            value={l.aggregates.totals.item_count.toLocaleString()}
            sub="Tracked titles"
          />
          <StatTile
            label="Completed"
            value={l.aggregates.totals.current_completed_count.toLocaleString()}
            sub={`${l.aggregates.totals.this_year_completed} this year`}
          />
          <StatTile
            label="Avg score"
            value={l.aggregates.totals.average_rating
              ? l.aggregates.totals.average_rating.toFixed(1)
              : "—"}
            sub="Out of 10"
          />
          <StatTile
            label="In progress"
            value={l.activeCount.toLocaleString()}
            sub="Watching or reading"
          />
        </div>

        <div class="analytics-grid">
          <section class="chart-card tile">
            <header class="chart-head">
              <h2>Completions by year</h2>
              <span class="chart-total"
                >{l.aggregates.totals.completed_count}</span
              >
            </header>
            {#if l.completions.length}
              <MediaBarChart
                labels={l.completions.map((b) => b.year)}
                values={l.completions.map((b) => b.count)}
                color="--accent"
              />
            {:else}
              <p class="empty-copy">No completed items yet.</p>
            {/if}
          </section>

          <section class="chart-card tile">
            <header class="chart-head">
              <h2>Score distribution</h2>
              <span class="chart-total">0–10</span>
            </header>
            {#if l.scores.length}
              <MediaBarChart
                labels={l.scores.map((b) => b.score)}
                values={l.scores.map((b) => b.count)}
                color="--accent-2"
              />
            {:else}
              <p class="empty-copy">No ratings yet.</p>
            {/if}
          </section>

          <section class="chart-card tile">
            <header class="chart-head">
              <h2>By kind</h2>
            </header>
            {#if l.typeFamilies.length}
              <MediaBarChart
                labels={l.typeFamilies.map((f) => f.type)}
                values={l.typeFamilies.map((f) => f.count)}
                color="--mark-teal"
                horizontal
              />
            {:else}
              <p class="empty-copy">Your collection will take shape here.</p>
            {/if}
          </section>
        </div>

        {#if l.continueItems.length}
          <section class="shelf">
            <header class="shelf-head">
              <div>
                <p class="eyebrow">Keep going</p>
                <h2>Watching &amp; reading</h2>
              </div>
              <span class="muted"
                >{l.activeCount} active{l.activeCount > 6
                  ? " · showing 6"
                  : ""}</span
              >
            </header>
            <div class="continue-grid">
              {#each l.continueItems as item (item.id)}
                <a class="continue-card tile" href={`/library/${item.id}`}>
                  <div class="thumb">
                    {#if item.cover_image_url}
                      <img src={item.cover_image_url} alt="" loading="lazy" />
                    {:else}
                      <span class="thumb-ph" aria-hidden="true"
                        >{l.initial(item)}</span
                      >
                    {/if}
                  </div>
                  <div class="continue-copy">
                    <span class="kicker">
                      <span
                        class="dot"
                        style={`background:${mediaTypeColor(item.media_type)}`}
                      ></span>{mediaTypeLabel(item.media_type)}
                    </span>
                    <h3>{l.primaryTitle(item)}</h3>
                    {#if l.altTitle(item)}<span class="alt"
                        >{l.altTitle(item)}</span
                      >{/if}
                    {#if item.total}
                      <div class="progress-track">
                        <span style={`width:${l.progressValue(item)}%`}></span>
                      </div>
                    {/if}
                    <span class="progress-label">
                      {item.position ?? 0}{item.total ? ` / ${item.total}` : ""}
                      {item.unit ?? ""}
                    </span>
                  </div>
                </a>
              {/each}
            </div>
          </section>
        {/if}

        <section class="shelf">
          <header class="shelf-head">
            <div>
              <p class="eyebrow">Collection</p>
              <h2>Everything in the index</h2>
            </div>
            <span class="muted">{l.items.length} shown</span>
          </header>
          {#if l.groupedItems.length}
            {#each l.groupedItems as [status, group] (status)}
              <div class="status-group">
                <header class="status-head">
                  <h3>
                    <span
                      class={`status-dot ${l.statusTone(status)}`}
                      aria-hidden="true"
                    ></span>
                    {l.statusLabel(status)}
                  </h3>
                  <span>{l.statusCounts[status] ?? group.length}</span>
                </header>
                <div class="poster-grid">
                  {#each group as item (item.id)}
                    <a class="poster" href={`/library/${item.id}`}>
                      <div class="cover">
                        {#if item.cover_image_url}
                          <img
                            src={item.cover_image_url}
                            alt=""
                            loading="lazy"
                          />
                        {:else}
                          <span class="cover-ph" aria-hidden="true"
                            >{l.initial(item)}</span
                          >
                        {/if}
                        {#if item.rating != null}
                          <span class="score-badge"
                            >{item.rating.toFixed(1)}</span
                          >
                        {/if}
                      </div>
                      <h3 title={l.primaryTitle(item)}>
                        {l.primaryTitle(item)}
                      </h3>
                      <span class="poster-sub">
                        <span
                          class="dot"
                          style={`background:${mediaTypeColor(item.media_type)}`}
                        ></span>{mediaTypeLabel(item.media_type)}
                      </span>
                    </a>
                  {/each}
                </div>
              </div>
            {/each}
            {#if l.hasMore}
              <button
                class="load-more"
                onclick={l.loadMore}
                disabled={l.loadingMore}
              >
                {l.loadingMore ? "Loading…" : "Load more"}
              </button>
            {/if}
          {:else}
            <div class="empty-panel tile">No media items found.</div>
          {/if}
        </section>
      </LoadingBoundary>
    {/if}
  {/if}
</section>

<style>
  .media-shell {
    display: grid;
    gap: 1.75rem;
  }
  h2,
  h3,
  p {
    margin: 0;
  }
  .eyebrow {
    color: var(--text-muted);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .muted {
    color: var(--text-muted);
  }
  .error {
    color: var(--danger);
  }

  /* Filter */
  .filter-bar {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    margin-top: -0.75rem;
  }
  .chip {
    padding: 0.32rem 0.85rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--surface);
    color: var(--text-muted);
    font-size: 0.78rem;
    font-weight: 600;
    cursor: pointer;
    transition:
      color 0.12s ease,
      border-color 0.12s ease,
      background 0.12s ease;
  }
  .chip:hover {
    color: var(--text);
    border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
  }
  .chip.active {
    color: var(--bg);
    background: var(--accent);
    border-color: var(--accent);
  }
  .filter-options {
    display: flex;
    flex-wrap: wrap;
    gap: 0.65rem;
    margin-top: -1rem;
  }
  .filter-options label {
    display: grid;
    gap: 0.25rem;
    color: var(--text-muted);
    font-size: 0.68rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }
  .filter-options select {
    min-width: 10rem;
    padding: 0.4rem 1.9rem 0.4rem 0.65rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    font: inherit;
    font-size: 0.78rem;
    letter-spacing: normal;
    text-transform: none;
  }
  .filter-options select:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
  }

  /* Stats */
  .stat-strip {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 0.75rem;
  }

  /* Analytics */
  .analytics-grid {
    display: grid;
    grid-template-columns: 1.15fr 1fr 0.85fr;
    gap: 0.85rem;
  }
  .chart-card {
    padding: 1rem 1.1rem 0.85rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .chart-head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 1rem;
  }
  .chart-head h2 {
    font-size: 0.9rem;
    letter-spacing: -0.01em;
  }
  .chart-total {
    color: var(--accent);
    font-size: 0.78rem;
    font-weight: 750;
  }
  .empty-copy {
    color: var(--text-muted);
    font-size: 0.82rem;
    padding: 2rem 0;
  }

  /* Shelves */
  .shelf {
    display: grid;
    gap: 1rem;
  }
  .shelf-head {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 1rem;
    border-bottom: 1px solid var(--border);
    padding-bottom: 0.6rem;
  }
  .shelf-head h2 {
    font-size: 1.1rem;
    letter-spacing: -0.02em;
  }

  /* Continue strip */
  .continue-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 0.85rem;
  }
  .continue-card {
    display: flex;
    gap: 0.85rem;
    padding: 0.75rem;
    min-width: 0;
    color: var(--text);
    transition:
      transform 0.12s ease,
      border-color 0.12s ease;
  }
  .continue-card:hover {
    transform: translateY(-2px);
    border-color: color-mix(in srgb, var(--accent) 50%, var(--border));
    text-decoration: none;
  }
  .thumb,
  .thumb-ph {
    width: 3.2rem;
    height: 4.6rem;
    flex: 0 0 auto;
    border-radius: 5px;
    overflow: hidden;
  }
  .continue-copy {
    min-width: 0;
    display: grid;
    align-content: center;
    gap: 0.28rem;
  }
  .kicker {
    color: var(--text-muted);
    font-size: 0.62rem;
    font-weight: 750;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .continue-copy h3 {
    font-size: 0.86rem;
    line-height: 1.25;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .alt {
    color: var(--text-muted);
    font-size: 0.68rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .progress-track {
    height: 0.35rem;
    border-radius: 999px;
    overflow: hidden;
    background: var(--surface-2);
    margin-top: 0.15rem;
  }
  .progress-track span {
    display: block;
    height: 100%;
    border-radius: inherit;
    background: linear-gradient(90deg, var(--accent), var(--accent-2));
  }
  .progress-label {
    color: var(--text-muted);
    font-size: 0.68rem;
  }

  /* Collection posters */
  .status-group {
    display: grid;
    gap: 0.75rem;
    margin-bottom: 1.25rem;
  }
  .status-head {
    display: flex;
    align-items: baseline;
    gap: 0.55rem;
  }
  .status-head h3 {
    display: flex;
    align-items: center;
    gap: 0.45rem;
    font-size: 0.82rem;
    font-weight: 700;
    text-transform: capitalize;
  }
  .status-head span {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .status-dot {
    width: 0.55rem;
    height: 0.55rem;
    flex: 0 0 auto;
    border-radius: 50%;
    background: var(--text-muted);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--text-muted) 14%, transparent);
  }
  .status-dot.completed {
    background: var(--mark-teal);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--mark-teal) 16%, transparent);
  }
  .status-dot.planned {
    background: var(--mark-amber);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--mark-amber) 16%, transparent);
  }
  .status-dot.abandoned {
    background: var(--danger);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--danger) 16%, transparent);
  }
  .status-dot.paused {
    background: var(--accent-2);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent-2) 16%, transparent);
  }
  .poster-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(7.5rem, 1fr));
    gap: 0.9rem 0.8rem;
  }
  .poster {
    display: grid;
    gap: 0.35rem;
    min-width: 0;
    color: var(--text);
  }
  .cover {
    position: relative;
    aspect-ratio: 2 / 3;
    border-radius: 6px;
    overflow: hidden;
    background: var(--surface-2);
    border: 1px solid var(--border);
    transition:
      transform 0.12s ease,
      box-shadow 0.12s ease;
  }
  .poster:hover .cover {
    transform: translateY(-3px);
    box-shadow: 0 12px 26px rgb(0 0 0 / 0.35);
  }
  .cover img,
  .cover-ph,
  .thumb img,
  .thumb-ph {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }
  .cover-ph,
  .thumb-ph {
    display: grid;
    place-items: center;
    background: linear-gradient(145deg, var(--surface-2), var(--surface));
    color: var(--accent);
    font-size: 1.4rem;
    font-weight: 800;
  }
  .score-badge {
    position: absolute;
    top: 0.35rem;
    right: 0.35rem;
    padding: 0.08rem 0.34rem;
    border-radius: 999px;
    background: color-mix(in srgb, var(--bg) 78%, transparent);
    backdrop-filter: blur(4px);
    color: var(--accent);
    font-size: 0.7rem;
    font-weight: 750;
    line-height: 1.4;
  }
  .poster h3 {
    font-size: 0.75rem;
    line-height: 1.25;
    font-weight: 600;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .poster-sub {
    color: var(--text-muted);
    font-size: 0.66rem;
  }
  .dot {
    display: inline-block;
    width: 0.42rem;
    height: 0.42rem;
    border-radius: 50%;
    margin-right: 0.35rem;
    vertical-align: middle;
  }

  .empty-panel {
    padding: 1.1rem;
    color: var(--text-muted);
  }
  .load-more {
    justify-self: center;
    padding: 0.55rem 1.5rem;
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

  @media (max-width: 1024px) {
    .stat-strip {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .analytics-grid {
      grid-template-columns: 1fr;
    }
    .continue-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
