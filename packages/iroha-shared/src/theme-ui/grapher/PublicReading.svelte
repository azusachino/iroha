<script lang="ts">
  import PanelFrame from "../../components/PanelFrame.svelte";
  import {
    PUBLIC_BOOK_STATUS_LABELS,
    type PublicBook,
  } from "../../domain/public-book";

  let { books }: { books: PublicBook[] } = $props();

  const groups = $derived.by(() => {
    const years = new Map<string, PublicBook[]>();
    for (const book of books) {
      const year = (
        book.finished_at ??
        book.last_read_at ??
        book.started_at ??
        ""
      ).slice(0, 4);
      const list = years.get(year) ?? [];
      list.push(book);
      years.set(year, list);
    }
    return [...years].sort(([a], [b]) => b.localeCompare(a));
  });
</script>

<PanelFrame label="Reading list">
  {#each groups as [year, list] (year)}
    {#if year}<h3 class="book-year">{year}</h3>{/if}
    <ul class="book-list">
      {#each list as book (book.id)}
        <li class="book-row">
          <span class="book-title">{book.title}</span>
          {#if book.authors.length > 0}
            <span class="book-authors">{book.authors.join(", ")}</span>
          {/if}
          <span class="book-status" data-status={book.status}>
            {PUBLIC_BOOK_STATUS_LABELS[book.status] ?? book.status}
          </span>
          {#if book.finished_at ?? book.last_read_at ?? book.started_at}
            <time
              class="book-date"
              datetime={book.finished_at ??
                book.last_read_at ??
                book.started_at}
            >
              {(
                book.finished_at ??
                book.last_read_at ??
                book.started_at
              )?.slice(0, 10)}
            </time>
          {/if}
        </li>
      {/each}
    </ul>
  {/each}
</PanelFrame>

<style>
  .book-year {
    margin: 1.1rem 0 0.35rem;
    font-size: var(--type-label);
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--text-muted);
  }
  .book-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 0.35rem;
  }
  .book-row {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: 0.25rem 0.75rem;
    padding: 0.3rem 0;
    border-bottom: 1px solid var(--border);
  }
  .book-row:last-child {
    border-bottom: 0;
  }
  .book-title {
    font-weight: 600;
  }
  .book-authors {
    color: var(--text-muted);
    font-size: var(--type-label);
  }
  .book-status {
    margin-left: auto;
    padding: 0.1rem 0.55rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    font-size: var(--type-caption);
    color: var(--text-muted);
    white-space: nowrap;
  }
  .book-status[data-status="completed"] {
    color: var(--accent);
    border-color: var(--accent);
  }
  .book-date {
    color: var(--text-muted);
    font-size: var(--type-label);
    white-space: nowrap;
  }
</style>
