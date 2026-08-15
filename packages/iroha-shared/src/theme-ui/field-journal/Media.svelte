<script lang="ts">
  import type { MediaThemeProps } from "../../media";
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

  let selectedYear = $state("");
  $effect(() => {
    selectedYear = completedYear;
  });

  const families = [
    { value: "", label: "All" },
    { value: "anime", label: "Anime" },
    { value: "manga_book", label: "Manga & books" },
    { value: "game", label: "Games" },
  ];
</script>

<section class="journal-shelf" aria-labelledby="journal-shelf-title">
  <header class="shelf-opening">
    <div>
      <p class="journal-kicker">Reading &amp; watching log</p>
      <h1 id="journal-shelf-title">Kept in the margins.</h1>
      <p>A record of attention, entered title by title, not a backlog.</p>
    </div>
    <div class="shelf-stamp" aria-label="Titles kept">
      <strong>{aggregates.totals.item_count}</strong>
      <span>titles</span>
    </div>
  </header>

  <div class="journal-rule"><span>the shelf</span></div>

  <nav class="shelf-families" aria-label="Media family">
    {#each families as option (option.value)}
      <button
        class:active={family === option.value}
        type="button"
        onclick={() => onFamily(option.value)}
      >
        {option.label}
      </button>
    {/each}
  </nav>

  <div class="shelf-filters">
    <label>
      <span>Status</span>
      <select
        value={status}
        onchange={(event) =>
          onStatus((event.currentTarget as HTMLSelectElement).value)}
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
        bind:value={selectedYear}
        onchange={(event) =>
          onYear((event.currentTarget as HTMLSelectElement).value)}
      >
        <option value="">Lifetime</option>
        {#each yearOptions as option (option.year)}
          <option value={option.year}>{option.year}</option>
        {/each}
      </select>
    </label>
  </div>

  <dl class="shelf-summary">
    <div>
      <dt>Completed</dt>
      <dd>{aggregates.totals.completed_count}</dd>
    </div>
    <div>
      <dt>This year</dt>
      <dd>{aggregates.totals.this_year_completed}</dd>
    </div>
    <div>
      <dt>Average score</dt>
      <dd>
        {aggregates.totals.average_rating
          ? aggregates.totals.average_rating.toFixed(1)
          : "—"}
      </dd>
    </div>
    <div>
      <dt>In progress</dt>
      <dd>{activeCount}</dd>
    </div>
  </dl>

  <section class="shelf-card">
    <header>
      <div>
        <p class="journal-kicker">Current shelf</p>
        <h2>{items.length} visible titles</h2>
      </div>
      <span
        >{typeFamilies
          .map((item) => `${item.type}: ${item.count}`)
          .join(" · ")}</span
      >
    </header>
    {#if items.length === 0}
      <p class="shelf-empty">No titles match this shelf.</p>
    {:else}
      <div class="shelf-grid">
        {#each items as item (item.id)}
          <MediaAssetCard {item} {theme} />
        {/each}
      </div>
    {/if}
  </section>

  {#if hasMore}
    <button
      class="log-continue"
      type="button"
      disabled={loadingMore}
      onclick={onLoadMore}
    >
      {loadingMore ? "Turning the page…" : "Turn the page for more entries"}
    </button>
  {/if}

  <footer class="journal-source">
    <span
      >{completions.length} completion periods · {scores.length} score buckets</span
    >
    <span>Source: imported provider records</span>
  </footer>
</section>

<style>
  .journal-shelf {
    display: grid;
    gap: 1.5rem;
    min-width: 0;
  }
  .journal-shelf > * {
    min-width: 0;
  }
  .journal-kicker {
    margin: 0 0 0.55rem;
    color: var(--accent);
    font-size: 0.68rem;
    letter-spacing: 0.16em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    font-family: var(--font-serif);
    font-weight: 400;
    letter-spacing: -0.04em;
  }
  h1 {
    max-width: 13ch;
    font-size: clamp(2.6rem, 6vw, 5rem);
    line-height: 0.92;
  }
  h2 {
    margin: 0.25rem 0 0.5rem;
    font-size: 1.5rem;
  }
  .shelf-opening {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
  }
  .shelf-opening p:last-child {
    max-width: 36rem;
    color: var(--text-muted);
    line-height: 1.7;
  }
  .shelf-stamp {
    display: grid;
    justify-items: end;
    color: var(--text-muted);
  }
  .shelf-stamp strong {
    color: var(--accent);
    font-family: var(--font-serif);
    font-size: 3.2rem;
    font-weight: 400;
    line-height: 0.85;
  }
  .shelf-stamp span {
    margin-top: 0.5rem;
    font-size: 0.68rem;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .journal-rule {
    display: flex;
    align-items: center;
    gap: 1rem;
    color: var(--text-muted);
    font-family: var(--font-serif);
    font-size: 0.8rem;
    font-style: italic;
  }
  .journal-rule::after {
    content: "";
    height: 1px;
    flex: 1;
    background: var(--border);
  }
  .shelf-families {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
  }
  .shelf-families button {
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 0.45rem 0.85rem;
    background: transparent;
    color: var(--text-muted);
    font: inherit;
    font-size: 0.75rem;
    cursor: pointer;
  }
  .shelf-families button.active,
  .shelf-families button:hover {
    border-color: var(--accent);
    color: var(--accent);
  }
  .shelf-filters {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1rem;
  }
  .shelf-filters label {
    display: grid;
    gap: 0.3rem;
  }
  .shelf-filters span {
    color: var(--text-muted);
    font-size: 0.62rem;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .shelf-filters select {
    min-width: 9rem;
    border: 1px solid var(--border);
    padding: 0.5rem;
    background: color-mix(in srgb, var(--surface-1) 84%, transparent);
    color: var(--text);
    font-family: var(--font-serif);
    font-size: 0.85rem;
  }
  .shelf-summary {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    margin: 0;
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface-1) 84%, transparent);
  }
  .shelf-summary div {
    display: grid;
    gap: 0.4rem;
    border-right: 1px solid var(--border);
    padding: 1.1rem;
  }
  .shelf-summary div:last-child {
    border-right: 0;
  }
  dt {
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  dd {
    margin: 0;
    font-family: var(--font-serif);
    font-size: 1.35rem;
  }
  .shelf-card {
    border: 1px solid var(--border);
    padding: 1.5rem;
    background: color-mix(in srgb, var(--surface-1) 84%, transparent);
  }
  .shelf-card > header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .shelf-card > header > span {
    color: var(--text-muted);
    font-size: 0.7rem;
    text-align: right;
  }
  .shelf-empty {
    margin-top: 1rem;
    color: var(--text-muted);
  }
  .shelf-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(9rem, 1fr));
    gap: 1.1rem;
    margin-top: 1.25rem;
  }
  .log-continue {
    justify-self: center;
    border: 1px solid var(--border);
    padding: 0.6rem 1.2rem;
    background: transparent;
    color: var(--text-muted);
    font-family: var(--font-serif);
    font-style: italic;
    font-size: 0.85rem;
    cursor: pointer;
  }
  .log-continue:hover:not(:disabled) {
    border-color: var(--accent);
    color: var(--accent);
  }
  .log-continue:disabled {
    opacity: 0.5;
    cursor: default;
  }
  .journal-source {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding-top: 1rem;
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  @media (max-width: 768px) {
    .shelf-opening,
    .shelf-card > header,
    .journal-source {
      display: block;
    }
    .shelf-stamp {
      align-items: start;
      justify-items: start;
      margin-top: 1.5rem;
    }
    .shelf-card > header > span {
      display: block;
      margin-top: 0.6rem;
      text-align: left;
    }
    .shelf-summary {
      grid-template-columns: 1fr;
    }
    .shelf-summary div {
      border-right: 0;
      border-bottom: 1px solid var(--border);
    }
    .shelf-summary div:last-child {
      border-bottom: 0;
    }
  }
</style>
