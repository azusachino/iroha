<script lang="ts">
  import { onMount } from "svelte";
  import {
    getMediaAggregates,
    listMedia,
    type MediaAggregates,
    type MediaRow,
  } from "$lib/api";
  import StatTile from "$lib/components/StatTile.svelte";
  import MediaBarChart from "$lib/components/MediaBarChart.svelte";

  let aggregates = $state<MediaAggregates | null>(null);
  let items = $state<MediaRow[]>([]);
  let nextCursor = $state<string | null>(null);
  let hasMore = $state(false);
  let loadingMore = $state(false);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let family = $state("");
  let status = $state("");
  let completedYear = $state("");
  let statusCounts = $state<Record<string, number>>({});
  let activeCount = $state(0);

  const FAMILIES = [
    { value: "", label: "All" },
    { value: "anime", label: "Anime" },
    { value: "manga_book", label: "Manga & books" },
    { value: "game", label: "Games" },
  ];

  // Only actively-in-progress items belong in the "continue" strip; paused /
  // on-hold entries keep status=in_progress but carry hidden_from_continue.
  const isContinuing = (item: MediaRow) =>
    item.status === "in_progress" && !item.hidden_from_continue;
  const continueItems = $derived(items.filter(isContinuing).slice(0, 6));

  const STATUS_ORDER = [
    "paused",
    "completed",
    "planned",
    "abandoned",
    "unknown",
  ];
  const groupedItems = $derived(
    Object.entries(
      items
        .filter((item) => !isContinuing(item))
        .reduce(
          (groups, item) => {
            // Paused items share the in_progress status; give them a shelf.
            const key =
              item.status === "in_progress"
                ? "paused"
                : item.status || "unknown";
            (groups[key] ??= []).push(item);
            return groups;
          },
          {} as Record<string, MediaRow[]>,
        ),
    ).sort(
      ([a], [b]) =>
        (STATUS_ORDER.indexOf(a) + 1 || 99) -
        (STATUS_ORDER.indexOf(b) + 1 || 99),
    ),
  );

  // The API splits by raw media_type (anime_season, manga, movie, ova…);
  // collapse those into display families for the "By kind" chart.
  function typeFamily(type: string): string {
    if (["manga", "one_shot", "light_novel", "book", "novel"].includes(type))
      return "Manga & books";
    if (["anime_season", "movie", "ona", "ova", "special"].includes(type))
      return "Anime";
    if (type === "game") return "Games";
    return "Other";
  }
  const typeFamilies = $derived(
    Object.entries(
      (aggregates?.type_split ?? []).reduce(
        (families, bucket) => {
          const key = typeFamily(bucket.type);
          families[key] = (families[key] ?? 0) + bucket.count;
          return families;
        },
        {} as Record<string, number>,
      ),
    )
      .map(([type, count]) => ({ type, count }))
      .sort((a, b) => b.count - a.count),
  );

  const completions = $derived(aggregates?.completions_by_year ?? []);
  const scores = $derived(aggregates?.score_distribution ?? []);
  const yearOptions = $derived(
    [...completions].sort((a, b) => b.year - a.year),
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
        listMedia({
          limit: 100,
          family: family || undefined,
          status: status || undefined,
          completed_year: completedYear ? Number(completedYear) : undefined,
        }),
      ]);
      aggregates = nextAggregates;
      items = page.items;
      nextCursor = page.next_cursor;
      hasMore = page.has_more;
      statusCounts = page.status_counts ?? {};
      activeCount = page.active_count ?? 0;
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    } finally {
      loading = false;
    }
  }

  async function selectFamily(value: string) {
    if (value === family) return;
    family = value;
    await reloadItems();
  }

  async function selectStatus() {
    await reloadItems();
  }

  async function selectYear() {
    await reloadItems();
  }

  async function reloadItems() {
    loading = true;
    error = null;
    try {
      const page = await listMedia({
        limit: 100,
        family: family || undefined,
        status: status || undefined,
        completed_year: completedYear ? Number(completedYear) : undefined,
      });
      items = page.items;
      nextCursor = page.next_cursor;
      hasMore = page.has_more;
      statusCounts = page.status_counts ?? {};
      activeCount = page.active_count ?? 0;
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
      const page = await listMedia({
        limit: 100,
        cursor: nextCursor,
        family: family || undefined,
        status: status || undefined,
        completed_year: completedYear ? Number(completedYear) : undefined,
      });
      items = [...items, ...page.items];
      nextCursor = page.next_cursor;
      hasMore = page.has_more;
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    } finally {
      loadingMore = false;
    }
  }

  const TYPE_LABELS: Record<string, string> = {
    anime_season: "Anime",
    movie: "Movie",
    ona: "ONA",
    ova: "OVA",
    special: "Special",
    manga: "Manga",
    one_shot: "One-shot",
    light_novel: "Light novel",
    book: "Book",
    game: "Game",
    real: "Live action",
    music: "Music",
  };
  function typeLabel(type: string): string {
    return TYPE_LABELS[type] ?? type.replaceAll("_", " ");
  }
  // Small family-colored dot so anime / manga-books / games read apart at a
  // glance; the text label still carries the meaning (color is not the only cue).
  function familyColor(type: string): string {
    const fam = typeFamily(type);
    if (fam === "Anime") return "var(--mark-teal)";
    if (fam === "Manga & books") return "var(--mark-magenta)";
    if (fam === "Games") return "var(--mark-amber)";
    return "var(--text-muted)";
  }

  function statusLabel(status: string): string {
    return status
      .replaceAll("_", " ")
      .replace(/^./, (char) => char.toUpperCase());
  }
  function statusTone(status: string): string {
    if (status === "completed") return "completed";
    if (status === "planned") return "planned";
    if (status === "abandoned") return "abandoned";
    if (status === "paused") return "paused";
    return "unknown";
  }

  function progressValue(item: MediaRow): number {
    if (item.progress_percent != null)
      return Math.min(Math.max(item.progress_percent, 0), 100);
    if (item.position != null && item.total)
      return Math.min(Math.max((item.position / item.total) * 100, 0), 100);
    return 0;
  }

  // Default to the native (Japanese) title; keep the English/romaji as a
  // secondary line when it differs.
  function primaryTitle(item: MediaRow): string {
    return item.native_title || item.title;
  }
  function altTitle(item: MediaRow): string {
    return item.native_title && item.native_title !== item.title
      ? item.title
      : "";
  }
  function initial(item: MediaRow): string {
    return primaryTitle(item).slice(0, 1);
  }
</script>

<svelte:head>
  <title>Media · iroha</title>
</svelte:head>

<section class="media-shell">
  <header class="domain-header">
    <p class="eyebrow">Media</p>
    <h1>Watchlist &amp; bookshelf</h1>
    <p class="muted">
      Everything you follow on AniList and Bangumi, on one shelf.
    </p>
  </header>

  <div class="filter-bar" role="tablist" aria-label="Filter by kind">
    {#each FAMILIES as f (f.value)}
      <button
        class="chip"
        class:active={family === f.value}
        role="tab"
        aria-selected={family === f.value}
        onclick={() => selectFamily(f.value)}
      >
        {f.label}
      </button>
    {/each}
  </div>

  <div class="filter-options" aria-label="Media filters">
    <label>
      <span>Status</span>
      <select bind:value={status} onchange={() => selectStatus()}>
        <option value="">All statuses</option>
        <option value="in_progress">In progress</option>
        <option value="completed">Completed</option>
        <option value="planned">Planned</option>
        <option value="abandoned">Abandoned</option>
      </select>
    </label>
    <label>
      <span>Completed year</span>
      <select bind:value={completedYear} onchange={() => selectYear()}>
        <option value="">All years</option>
        {#each yearOptions as option (option.year)}
          <option value={option.year}>{option.year}</option>
        {/each}
      </select>
    </label>
  </div>

  {#if loading}
    <p class="muted">Loading media history…</p>
  {:else if error}
    <p class="error">Failed to load media: {error}</p>
  {:else if aggregates}
    <div class="stat-strip">
      <StatTile
        label="Library"
        value={aggregates.totals.item_count.toLocaleString()}
        sub="Tracked titles"
      />
      <StatTile
        label="Completed"
        value={aggregates.totals.completed_count.toLocaleString()}
        sub={`${aggregates.totals.this_year_completed} this year`}
      />
      <StatTile
        label="Avg score"
        value={aggregates.totals.average_rating
          ? aggregates.totals.average_rating.toFixed(1)
          : "—"}
        sub="Out of 10"
      />
      <StatTile
        label="In progress"
        value={continueItems.length.toLocaleString()}
        sub="Watching or reading"
      />
    </div>

    <div class="analytics-grid">
      <section class="chart-card tile">
        <header class="chart-head">
          <h2>Completions by year</h2>
          <span class="chart-total">{aggregates.totals.completed_count}</span>
        </header>
        {#if completions.length}
          <MediaBarChart
            labels={completions.map((b) => b.year)}
            values={completions.map((b) => b.count)}
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
        {#if scores.length}
          <MediaBarChart
            labels={scores.map((b) => b.score)}
            values={scores.map((b) => b.count)}
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
        {#if typeFamilies.length}
          <MediaBarChart
            labels={typeFamilies.map((f) => f.type)}
            values={typeFamilies.map((f) => f.count)}
            color="--mark-teal"
            horizontal
          />
        {:else}
          <p class="empty-copy">Your collection will take shape here.</p>
        {/if}
      </section>
    </div>

    {#if continueItems.length}
      <section class="shelf">
        <header class="shelf-head">
          <div>
            <p class="eyebrow">Keep going</p>
            <h2>Watching &amp; reading</h2>
          </div>
          <span class="muted"
            >{activeCount} active{activeCount > 6 ? " · showing 6" : ""}</span
          >
        </header>
        <div class="continue-grid">
          {#each continueItems as item (item.id)}
            <a class="continue-card tile" href={`/media/${item.id}`}>
              <div class="thumb">
                {#if item.cover_image_url}
                  <img src={item.cover_image_url} alt="" loading="lazy" />
                {:else}
                  <span class="thumb-ph" aria-hidden="true"
                    >{initial(item)}</span
                  >
                {/if}
              </div>
              <div class="continue-copy">
                <span class="kicker">
                  <span
                    class="dot"
                    style={`background:${familyColor(item.media_type)}`}
                  ></span>{typeLabel(item.media_type)}
                </span>
                <h3>{primaryTitle(item)}</h3>
                {#if altTitle(item)}<span class="alt">{altTitle(item)}</span
                  >{/if}
                {#if item.total}
                  <div class="progress-track">
                    <span style={`width:${progressValue(item)}%`}></span>
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
        <span class="muted">{items.length} shown</span>
      </header>
      {#if groupedItems.length}
        {#each groupedItems as [status, group] (status)}
          <div class="status-group">
            <header class="status-head">
              <h3>
                <span
                  class={`status-dot ${statusTone(status)}`}
                  aria-hidden="true"
                ></span>
                {statusLabel(status)}
              </h3>
              <span>{statusCounts[status] ?? group.length}</span>
            </header>
            <div class="poster-grid">
              {#each group as item (item.id)}
                <a class="poster" href={`/media/${item.id}`}>
                  <div class="cover">
                    {#if item.cover_image_url}
                      <img src={item.cover_image_url} alt="" loading="lazy" />
                    {:else}
                      <span class="cover-ph" aria-hidden="true"
                        >{initial(item)}</span
                      >
                    {/if}
                    {#if item.rating != null}
                      <span class="score-badge">{item.rating.toFixed(1)}</span>
                    {/if}
                  </div>
                  <h3 title={primaryTitle(item)}>{primaryTitle(item)}</h3>
                  <span class="poster-sub">
                    <span
                      class="dot"
                      style={`background:${familyColor(item.media_type)}`}
                    ></span>{typeLabel(item.media_type)}
                  </span>
                </a>
              {/each}
            </div>
          </div>
        {/each}
        {#if hasMore}
          <button class="load-more" onclick={loadMore} disabled={loadingMore}>
            {loadingMore ? "Loading…" : "Load more"}
          </button>
        {/if}
      {:else}
        <div class="empty-panel tile">No media items found.</div>
      {/if}
    </section>
  {/if}
</section>

<style>
  .media-shell {
    display: grid;
    gap: 1.75rem;
  }
  h1,
  h2,
  h3,
  p {
    margin: 0;
  }
  .domain-header h1 {
    font-size: 1.5rem;
    letter-spacing: -0.02em;
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

  @media (max-width: 900px) {
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
