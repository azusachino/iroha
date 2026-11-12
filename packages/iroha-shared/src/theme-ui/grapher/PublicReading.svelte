<script lang="ts">
  import PanelFrame from "../../components/PanelFrame.svelte";
  import type { PublicBook } from "../../domain/public-book";

  let { books }: { books: PublicBook[] } = $props();

  const months = [
    "January",
    "February",
    "March",
    "April",
    "May",
    "June",
    "July",
    "August",
    "September",
    "October",
    "November",
    "December",
  ];

  // Completion dates are calendar dates supplied by the public projection.
  // Never use a start or update time to invent a completion date.
  function completionDate(book: PublicBook): string | null {
    const date = book.finished_at?.slice(0, 10);
    if (!date || !/^\d{4}-\d{2}-\d{2}$/.test(date)) return null;
    const parsed = new Date(`${date}T00:00:00Z`);
    return Number.isNaN(parsed.getTime()) ||
      parsed.toISOString().slice(0, 10) !== date
      ? null
      : date;
  }

  let selection = $state("");
  const finished = $derived(
    books.filter((book) => book.status === "completed"),
  );
  const yearly = $derived.by(() => {
    const counts = new Map<string, number>();
    for (const book of finished) {
      const year = completionDate(book)?.slice(0, 4);
      if (year) counts.set(year, (counts.get(year) ?? 0) + 1);
    }
    return [...counts].sort(([a], [b]) => b.localeCompare(a));
  });
  const undated = $derived(finished.filter((book) => !completionDate(book)));
  const selectedYear = $derived(selection || yearly[0]?.[0] || "undated");
  const visible = $derived(
    finished
      .filter((book) => {
        const date = completionDate(book);
        return selectedYear === "undated"
          ? !date
          : date?.startsWith(`${selectedYear}-`);
      })
      .toSorted(
        (a, b) =>
          (completionDate(b) ?? "").localeCompare(completionDate(a) ?? "") ||
          a.title.localeCompare(b.title),
      ),
  );
  const monthly = $derived(
    months.map((month, index) => ({
      month,
      count: visible.filter(
        (book) => Number(completionDate(book)?.slice(5, 7)) === index + 1,
      ).length,
    })),
  );
  const maxYear = $derived(Math.max(1, ...yearly.map(([, count]) => count)));
  const maxMonth = $derived(Math.max(1, ...monthly.map(({ count }) => count)));
</script>

<PanelFrame label="Reading list">
  {#if finished.length === 0}
    <div class="empty">
      <p class="eyebrow">THE READING ARCHIVE</p>
      <h3>No finished books yet.</h3>
      <p>Finished books will appear here when they are recorded.</p>
    </div>
  {:else}
    <header class="reading-header">
      <div>
        <p class="eyebrow">THE READING ARCHIVE</p>
        <h3>A life in books<span>.</span></h3>
        <p class="intro">
          A record of the books I've finished, one year at a time.
        </p>
      </div>
      <div class="total">
        <strong>{finished.length}</strong><span
          >books finished<br />all time</span
        >
      </div>
    </header>

    <div class="year-heading">
      <div>
        <p class="eyebrow">EXPLORE THE YEARS</p>
        <h4>Reading through time</h4>
      </div>
      <label class="year-select"
        >Year
        <select
          aria-label="Reading year"
          value={selectedYear}
          onchange={(event) => (selection = event.currentTarget.value)}
        >
          {#each yearly as [year, count] (year)}
            <option value={year}
              >{year} · {count} {count === 1 ? "book" : "books"}</option
            >
          {/each}
          {#if undated.length}<option value="undated"
              >Date unavailable · {undated.length}</option
            >{/if}
        </select>
      </label>
    </div>

    {#if yearly.length}
      <div class="year-chart" aria-label="Finished books by year">
        {#each yearly as [year, count] (year)}
          <button
            type="button"
            class:chosen={selectedYear === year}
            aria-pressed={selectedYear === year}
            aria-label={`${year}: ${count} ${count === 1 ? "book" : "books"} finished`}
            onclick={() => (selection = year)}
          >
            <span class="year-count">{count}</span>
            <span class="year-track"
              ><span
                class="year-bar"
                style={`--fill:${(count / maxYear) * 100}%`}
              ></span></span
            >
            <span class="year-label">{year}</span>
          </button>
        {/each}
      </div>
    {/if}

    <div class="reading-content">
      <div class="section-title">
        <div>
          <p class="eyebrow">
            {selectedYear === "undated" ? "DATE UNAVAILABLE" : selectedYear}
          </p>
          <h4>
            {selectedYear === "undated" ? "Undated books" : "The year in books"}
          </h4>
        </div>
        <p>
          <strong>{visible.length}</strong>
          {visible.length === 1 ? "book" : "books"} finished
        </p>
      </div>
      {#if selectedYear !== "undated"}
        <div
          class="month-chart"
          role="img"
          aria-label={`Books finished each month in ${selectedYear}: ${monthly.map(({ month, count }) => `${month} ${count}`).join(", ")}`}
        >
          {#each monthly as { month, count } (month)}
            <div class="month-column" aria-hidden="true">
              <span class="month-count">{count || "·"}</span>
              <span class="month-track"
                ><span
                  class="month-bar"
                  style={`--fill:${(count / maxMonth) * 100}%`}
                ></span></span
              >
              <span class="month-label">{month.slice(0, 1)}</span>
            </div>
          {/each}
        </div>
      {/if}
      <div class="list-heading">
        <h4>Finished books</h4>
        <span
          >{selectedYear === "undated"
            ? "Date unavailable"
            : selectedYear}</span
        >
      </div>
      <ul class="book-list">
        {#each visible as book (book.id)}
          <li class="book-row">
            <div>
              <span class="book-title">{book.title}</span
              >{#if book.authors.length}<span class="book-authors"
                  >{book.authors.join(", ")}</span
                >{/if}
            </div>
            {#if completionDate(book)}<time
                class="book-date"
                datetime={completionDate(book) ?? undefined}
                >{completionDate(book)}</time
              >{:else}<span class="book-date">Date unavailable</span>{/if}
          </li>
        {/each}
      </ul>
    </div>
  {/if}
</PanelFrame>

<style>
  .reading-header,
  .year-heading,
  .section-title,
  .list-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: end;
  }
  .reading-header {
    padding: 0.4rem 0 1.5rem;
    border-bottom: 1px solid var(--border);
  }
  .eyebrow {
    margin: 0 0 0.5rem;
    color: var(--text-muted);
    font-size: 0.69rem;
    font-weight: 700;
    letter-spacing: 0.12em;
  }
  h3,
  h4,
  p {
    margin: 0;
  }
  h3 {
    font-size: clamp(1.7rem, 4vw, 2.7rem);
    line-height: 1.08;
    letter-spacing: -0.04em;
  }
  h3 span {
    color: var(--accent);
  }
  h4 {
    font-size: 1.08rem;
    letter-spacing: -0.02em;
  }
  .intro,
  .empty > p:last-child {
    margin-top: 0.65rem;
    color: var(--text-muted);
    line-height: 1.5;
  }
  .total {
    display: flex;
    gap: 0.65rem;
    align-items: center;
    flex-shrink: 0;
  }
  .total strong {
    color: var(--accent);
    font-size: clamp(2.2rem, 5vw, 3.4rem);
    letter-spacing: -0.06em;
    line-height: 1;
    font-variant-numeric: tabular-nums;
  }
  .total span {
    color: var(--text-muted);
    font-size: 0.75rem;
    line-height: 1.35;
  }
  .year-heading {
    margin-top: 0.75rem;
  }
  .year-select {
    display: grid;
    gap: 0.3rem;
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  select {
    max-width: 100%;
    min-height: 44px;
    padding: 0.45rem 2rem 0.45rem 0.65rem;
    color: var(--text);
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    font: inherit;
    font-size: 0.85rem;
  }
  .year-chart {
    display: flex;
    overflow-x: auto;
    gap: 0.4rem;
    padding: 0.5rem 0 0.35rem;
  }
  .year-chart button {
    flex: 1 0 48px;
    min-width: 48px;
    display: grid;
    gap: 0.35rem;
    padding: 0.5rem 0.3rem;
    border: 1px solid transparent;
    border-radius: 8px;
    color: var(--text-muted);
    background: transparent;
    cursor: pointer;
    font: inherit;
  }
  .year-chart button:hover,
  .year-chart button.chosen {
    background: color-mix(in srgb, var(--accent) 9%, transparent);
    color: var(--text);
  }
  .year-chart button.chosen {
    border-color: color-mix(in srgb, var(--accent) 55%, var(--border));
  }
  .year-chart button:focus-visible,
  select:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
  }
  .year-count {
    font-size: 0.85rem;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
  }
  .year-track {
    height: 5px;
    background: var(--border);
    border-radius: 5px;
    overflow: hidden;
  }
  .year-bar {
    display: block;
    height: 100%;
    width: var(--fill);
    background: var(--accent);
    border-radius: inherit;
  }
  .year-label {
    font-size: 0.7rem;
    font-variant-numeric: tabular-nums;
  }
  .reading-content {
    margin-top: 0.65rem;
    padding-top: 1.3rem;
    border-top: 1px solid var(--border);
  }
  .section-title p:last-child,
  .list-heading span {
    color: var(--text-muted);
    font-size: 0.8rem;
    white-space: nowrap;
  }
  .section-title p strong {
    color: var(--text);
    font-size: 1.2rem;
  }
  .month-chart {
    display: grid;
    grid-template-columns: repeat(12, minmax(0, 1fr));
    gap: clamp(3px, 1vw, 12px);
    align-items: end;
    margin: 1.25rem 0 1.7rem;
  }
  .month-column {
    display: grid;
    gap: 0.4rem;
    text-align: center;
  }
  .month-count,
  .month-label {
    color: var(--text-muted);
    font-size: 0.68rem;
    font-variant-numeric: tabular-nums;
  }
  .month-track {
    height: 78px;
    display: flex;
    align-items: end;
    justify-content: center;
    border-bottom: 1px solid var(--border);
  }
  .month-bar {
    width: min(100%, 22px);
    height: var(--fill);
    background: var(--accent);
    opacity: 0.8;
    border-radius: 4px 4px 0 0;
  }
  .list-heading {
    margin: 1rem 0 0.35rem;
  }
  .book-list {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .book-row {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: baseline;
    padding: 0.8rem 0;
    border-bottom: 1px solid var(--border);
  }
  .book-row:last-child {
    border-bottom: 0;
  }
  .book-row > div {
    min-width: 0;
    overflow-wrap: anywhere;
  }
  .book-title {
    display: block;
    font-weight: 600;
    line-height: 1.35;
  }
  .book-authors {
    display: block;
    margin-top: 0.18rem;
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  .book-date {
    flex-shrink: 0;
    color: var(--text-muted);
    font-size: 0.75rem;
    white-space: nowrap;
    font-variant-numeric: tabular-nums;
  }
  .empty {
    padding: 1rem 0;
  }
  @media (max-width: 640px) {
    .reading-header {
      align-items: start;
      flex-direction: column;
    }
    .year-heading {
      align-items: start;
      flex-direction: column;
    }
    .year-select {
      width: 100%;
    }
    .book-row {
      flex-wrap: wrap;
      gap: 0.35rem 1rem;
    }
  }
</style>
