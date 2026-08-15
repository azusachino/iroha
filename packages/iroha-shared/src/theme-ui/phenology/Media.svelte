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

  const families = [
    { value: "", label: "All" },
    { value: "anime", label: "Anime" },
    { value: "manga_book", label: "Manga & books" },
    { value: "game", label: "Games" },
  ];
</script>

<section class="bloom-shelf" aria-labelledby="bloom-shelf-title">
  <header class="shelf-opening">
    <div>
      <p class="bloom-kicker">◔ Attention, in season</p>
      <h1 id="bloom-shelf-title">Kept in orbit.</h1>
      <p>
        Reading, watching, and playing as a record of attention — each title
        ripening at its own pace.
      </p>
    </div>
    <div class="shelf-count">
      <strong>{aggregates.totals.item_count}</strong>
      <span>titles</span>
    </div>
  </header>

  <nav class="shelf-families" aria-label="Media family">
    {#each families as option (option.value)}
      <button
        class:active={family === option.value}
        type="button"
        onclick={() => onFamily(option.value)}>{option.label}</button
      >
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
        value={completedYear}
        onchange={(event) =>
          onYear((event.currentTarget as HTMLSelectElement).value)}
      >
        <option value="" selected={completedYear === ""}>Lifetime</option>
        {#each yearOptions as option (option.year)}
          <option
            value={option.year}
            selected={completedYear === String(option.year)}
          >{option.year}</option>
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

  <section class="shelf-panel">
    <header>
      <div>
        <p class="bloom-kicker">Current shelf</p>
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
      class="load-more"
      type="button"
      disabled={loadingMore}
      onclick={onLoadMore}>{loadingMore ? "Gathering…" : "Load more"}</button
    >
  {/if}

  <footer class="bloom-source">
    <span
      >{completions.length} completion periods · {scores.length} score buckets</span
    >
    <span>Source: imported provider records</span>
  </footer>
</section>

<style>
  .bloom-shelf {
    display: grid;
    gap: 1.5rem;
    font-family: var(--font-serif);
    min-width: 0;
  }
  .bloom-shelf > * {
    min-width: 0;
  }
  .bloom-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.15em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    font-weight: 400;
    letter-spacing: -0.02em;
  }
  h1 {
    max-width: 12ch;
    font-size: clamp(2.4rem, 6vw, 4.9rem);
    line-height: 0.95;
  }
  h2 {
    font-size: 1.45rem;
  }
  .shelf-opening {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.75rem;
  }
  .shelf-opening p:last-child {
    max-width: 34rem;
    color: var(--text-muted);
    line-height: 1.65;
  }
  .shelf-count {
    display: grid;
    justify-items: end;
    color: var(--text-muted);
  }
  .shelf-count strong {
    color: var(--accent);
    font-style: italic;
    font-size: 3.1rem;
    font-weight: 400;
  }
  .shelf-count span {
    margin-top: 0.4rem;
    font-size: 0.68rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
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
    border-radius: calc(var(--radius) * 0.4);
    padding: 0.5rem;
    background: var(--surface);
    color: var(--text);
    font: inherit;
    font-size: 0.8rem;
  }
  .shelf-summary {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    margin: 0;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
  }
  .shelf-summary div {
    display: grid;
    gap: 0.35rem;
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
    font-style: italic;
    font-size: 1.3rem;
  }
  .shelf-panel {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 1.5rem;
    background: color-mix(in srgb, var(--surface) 90%, transparent);
  }
  .shelf-panel > header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .shelf-panel > header > span {
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
    grid-template-columns: repeat(auto-fill, minmax(9.5rem, 1fr));
    gap: 1.25rem;
    margin-top: 1.4rem;
  }
  .load-more {
    justify-self: center;
    border: 1px solid var(--accent);
    border-radius: 999px;
    padding: 0.55rem 1.2rem;
    background: transparent;
    color: var(--accent);
    font: inherit;
    cursor: pointer;
  }
  .load-more:disabled {
    opacity: 0.5;
  }
  .bloom-source {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding-top: 0.85rem;
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  @media (max-width: 768px) {
    .shelf-opening,
    .shelf-panel > header,
    .bloom-source {
      display: block;
    }
    .shelf-count {
      display: block;
      margin-top: 1.5rem;
    }
    .shelf-panel > header > span {
      display: block;
      margin-top: 0.6rem;
      text-align: left;
    }
    .shelf-filters,
    .shelf-summary {
      grid-template-columns: 1fr;
    }
    .shelf-filters {
      display: grid;
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
