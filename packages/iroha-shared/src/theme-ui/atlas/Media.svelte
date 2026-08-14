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
    { value: "", label: "All regions" },
    { value: "anime", label: "Anime" },
    { value: "manga_book", label: "Manga & books" },
    { value: "game", label: "Games" },
  ];
</script>

<section class="atlas-shelf" aria-labelledby="atlas-shelf-title">
  <header class="shelf-header">
    <div>
      <p class="atlas-kicker">Collection atlas · charted attention</p>
      <h1 id="atlas-shelf-title">The shelf, charted.</h1>
      <p class="shelf-sub">
        Reading, watching, and playing plotted as territory covered, not a
        backlog to clear.
      </p>
    </div>
    <div class="grid-ref">
      <span>{aggregates.totals.item_count}</span>
      <small>titles charted</small>
    </div>
  </header>

  <nav class="shelf-regions" aria-label="Media family">
    {#each families as option (option.value)}<button
        class:active={family === option.value}
        type="button"
        onclick={() => onFamily(option.value)}>{option.label}</button
      >{/each}
  </nav>

  <div class="shelf-filters">
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
        ><option value="">Lifetime</option>{#each yearOptions as option}<option
            value={option.year}>{option.year}</option
          >{/each}</select
      ></label
    >
  </div>

  <div class="shelf-summary">
    <div class="atlas-plate">
      <p class="atlas-kicker">Completed</p>
      <strong>{aggregates.totals.completed_count}</strong>
    </div>
    <div class="atlas-plate">
      <p class="atlas-kicker">This year</p>
      <strong>{aggregates.totals.this_year_completed}</strong>
    </div>
    <div class="atlas-plate">
      <p class="atlas-kicker">Average score</p>
      <strong
        >{aggregates.totals.average_rating
          ? aggregates.totals.average_rating.toFixed(1)
          : "—"}</strong
      >
    </div>
    <div class="atlas-plate">
      <p class="atlas-kicker">In progress</p>
      <strong>{activeCount}</strong>
    </div>
  </div>

  <section class="atlas-plate shelf-plate">
    <header class="shelf-plate-heading">
      <div>
        <p class="atlas-kicker">Current sheet</p>
        <h2>{items.length} visible titles</h2>
      </div>
      <span
        >{typeFamilies
          .map((item) => `${item.type}: ${item.count}`)
          .join(" · ")}</span
      >
    </header>
    <div class="shelf-grid">
      {#each items as item (item.id)}
        <MediaAssetCard {item} {theme} />
      {/each}
    </div>
    {#if items.length === 0}<p class="atlas-empty">
        No titles match this region.
      </p>{/if}
  </section>
  {#if hasMore}<button
      class="load-more"
      type="button"
      disabled={loadingMore}
      onclick={onLoadMore}
      >{loadingMore ? "Extending the survey…" : "Extend the survey"}</button
    >{/if}
  <footer class="atlas-source">
    {completions.length} completion periods · {scores.length} score buckets · source:
    imported provider records
  </footer>
</section>

<style>
  .atlas-shelf {
    display: grid;
    gap: 1.5rem;
    font-family: var(--font-sans);
    min-width: 0;
  }
  .atlas-shelf > * {
    min-width: 0;
  }
  .atlas-kicker {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.66rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .atlas-kicker::before {
    content: "⌖";
  }
  h1,
  h2 {
    margin: 0;
    font-weight: 600;
    letter-spacing: -0.03em;
  }
  h1 {
    max-width: 12ch;
    font-size: clamp(2.5rem, 6vw, 4.6rem);
    line-height: 1;
  }
  h2 {
    font-size: 1.45rem;
  }
  .shelf-header {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.75rem;
  }
  .shelf-sub {
    max-width: 40rem;
    margin: 1rem 0 0;
    color: var(--text-muted);
    line-height: 1.65;
  }
  .grid-ref {
    display: grid;
    justify-items: end;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    padding: 0.6rem 0.9rem;
    color: var(--accent);
    font-family: var(--font-mono);
    text-align: right;
  }
  .grid-ref span {
    font-size: 1.6rem;
  }
  .grid-ref small {
    margin-top: 0.2rem;
    color: var(--text-muted);
    font-size: 0.6rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  .shelf-regions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
  }
  .shelf-regions button {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 0.45rem 0.85rem;
    background: transparent;
    color: var(--text-muted);
    font: inherit;
    font-size: 0.78rem;
    cursor: pointer;
  }
  .shelf-regions button.active,
  .shelf-regions button:hover {
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
    font-family: var(--font-mono);
    font-size: 0.62rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  select {
    min-width: 9rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 0.5rem;
    background: var(--surface-1);
    color: var(--text);
    font: inherit;
    font-size: 0.8rem;
  }
  .atlas-plate {
    position: relative;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface-1) 88%, transparent);
  }
  .atlas-plate::before,
  .atlas-plate::after {
    content: "";
    position: absolute;
    width: 0.7rem;
    height: 0.7rem;
    opacity: 0.7;
  }
  .atlas-plate::before {
    top: -1px;
    left: -1px;
    border-top: 2px solid var(--accent);
    border-left: 2px solid var(--accent);
  }
  .atlas-plate::after {
    right: -1px;
    bottom: -1px;
    border-right: 2px solid var(--accent);
    border-bottom: 2px solid var(--accent);
  }
  .shelf-summary {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1rem;
  }
  .shelf-summary .atlas-plate {
    padding: 1.1rem;
  }
  .shelf-summary strong {
    display: block;
    margin-top: 0.3rem;
    font-family: var(--font-mono);
    font-size: 1.4rem;
    font-weight: 600;
  }
  .shelf-plate {
    padding: 1.5rem;
  }
  .shelf-plate-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .shelf-plate-heading > span {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.7rem;
    text-align: right;
  }
  .shelf-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(9rem, 1fr));
    gap: 1rem;
    margin-top: 1.25rem;
  }
  .atlas-empty {
    color: var(--text-muted);
  }
  .load-more {
    justify-self: center;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    padding: 0.55rem 1.1rem;
    background: transparent;
    color: var(--accent);
    font: inherit;
    font-size: 0.82rem;
    cursor: pointer;
  }
  .load-more:disabled {
    opacity: 0.5;
  }
  .atlas-source {
    border-top: 1px solid var(--border);
    padding-top: 0.8rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.66rem;
  }
  @media (max-width: 768px) {
    .shelf-header,
    .shelf-plate-heading {
      display: block;
    }
    .grid-ref {
      margin-top: 1.5rem;
      justify-items: start;
      text-align: left;
    }
    .shelf-plate-heading > span {
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
    .shelf-summary {
      display: grid;
    }
  }
</style>
