<script lang="ts">
  import PanelFrame from "../../components/PanelFrame.svelte";
  import BarChart from "../components/BarChart.svelte";
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
  let trendRange = $state("recent");
  let trendScroller: HTMLDivElement | undefined = $state();
  const yearly = $derived.by(() => {
    const counts = new Map<string, number>();
    for (const book of books) {
      const year = completionDate(book)?.slice(0, 4);
      if (year) counts.set(year, (counts.get(year) ?? 0) + 1);
    }
    return [...counts].sort(([a], [b]) => b.localeCompare(a));
  });
  const undated = $derived(books.filter((book) => !completionDate(book)));
  const selectedYear = $derived(selection || yearly[0]?.[0] || "undated");
  const visible = $derived(
    books
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
  const decades = $derived([
    ...new Set(yearly.map(([year]) => Math.floor(Number(year) / 10) * 10)),
  ]);
  const latestYear = $derived(Number(yearly[0]?.[0] ?? 0));
  const earliestYear = $derived(Number(yearly[yearly.length - 1]?.[0] ?? 0));
  const trendStart = $derived(
    trendRange === "recent"
      ? Math.max(earliestYear, latestYear - 9)
      : Math.max(0, Number(trendRange)),
  );
  const trendEnd = $derived(
    trendRange === "recent"
      ? latestYear
      : Math.min(latestYear, Number(trendRange) + 9),
  );
  const trend = $derived.by(() => {
    if (!yearly.length) return [];
    const counts = new Map(yearly);
    const end = trendEnd;
    return Array.from({ length: end - trendStart + 1 }, (_, index) => {
      const year = String(trendStart + index).padStart(4, "0");
      return { year, count: counts.get(year) ?? 0 };
    });
  });
  function chooseYear(year: string) {
    selection = year;
    if (
      year !== "undated" &&
      (Number(year) < trendStart || Number(year) > trendEnd)
    ) {
      trendRange = String(Math.floor(Number(year) / 10) * 10);
    }
  }
  function chooseTrendRange(range: string) {
    trendRange = range;
    const start =
      range === "recent"
        ? Math.max(earliestYear, latestYear - 9)
        : Math.max(0, Number(range));
    const end = range === "recent" ? latestYear : Number(range) + 9;
    if (
      selectedYear === "undated" ||
      Number(selectedYear) < start ||
      Number(selectedYear) > end
    ) {
      selection =
        yearly.find(
          ([year]) => Number(year) >= start && Number(year) <= end,
        )?.[0] ?? selection;
    }
  }
  $effect(() => {
    const scroller = trendScroller;
    const index = trend.findIndex(({ year }) => year === selectedYear);
    const length = trend.length;
    if (!scroller || index < 0) return;
    const frame = requestAnimationFrame(() => {
      const chart = scroller.querySelector<HTMLElement>(".bar-chart");
      if (!chart) return;
      const point =
        48 + ((chart.clientWidth - 64) * index) / Math.max(1, length - 1);
      if (
        point < scroller.scrollLeft + 40 ||
        point > scroller.scrollLeft + scroller.clientWidth - 40
      ) {
        scroller.scrollLeft = Math.max(0, point - scroller.clientWidth / 2);
      }
    });
    return () => cancelAnimationFrame(frame);
  });
  const maxMonth = $derived(Math.max(1, ...monthly.map(({ count }) => count)));
</script>

<PanelFrame label="Reading list">
  {#if books.length === 0}
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
        <strong>{books.length}</strong><span
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
          onchange={(event) => chooseYear(event.currentTarget.value)}
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
      <div class="trend-heading">
        <div>
          <h4>Books finished by year</h4>
          <p>
            Recorded completions by year. Years with no recorded completions are
            shown as zero. Choose a year above to see its books; scroll the
            chart to see more years.
          </p>
          <p>
            Highest yearly total: {Math.max(
              ...trend.map(({ count }) => count),
            )}.
          </p>
        </div>
        {#if Number(yearly[0][0]) - Number(yearly[yearly.length - 1][0]) >= 10}
          <label class="year-select"
            >Trend range
            <select
              aria-label="Trend range"
              value={trendRange}
              onchange={(event) => chooseTrendRange(event.currentTarget.value)}
            >
              <option value="recent">Recent 10 years</option>
              {#each decades as decade (decade)}
                <option value={String(decade)}
                  >{String(decade).padStart(4, "0")}–{String(
                    decade + 9,
                  ).padStart(4, "0")}</option
                >
              {/each}
            </select>
          </label>
        {/if}
      </div>
      <div
        class="trend-scroll"
        bind:this={trendScroller}
        role="region"
        aria-label="Yearly trend chart, scroll horizontally to see all years"
        tabindex="0"
      >
        <div style={`--trend-width: ${Math.max(420, trend.length * 55)}px`}>
          <BarChart
            categories={trend.map(({ year }) => year)}
            primary={{
              name: "Books finished",
              values: trend.map(({ count }) => count),
              color: "var(--accent)",
              axis: { unit: "books", minimumInterval: 1 },
            }}
            primaryType="line"
            showDataTable={false}
            activeIndex={selectedYear === "undated"
              ? undefined
              : trend.findIndex(({ year }) => year === selectedYear)}
            height={230}
          />
        </div>
      </div>
      <details class="trend-data">
        <summary>View chart data</summary>
        <table>
          <caption>Books finished by category</caption>
          <thead
            ><tr
              ><th scope="col">Year</th><th scope="col">Books finished</th></tr
            ></thead
          >
          <tbody>
            {#each trend as { year, count } (year)}
              <tr><th scope="row">{year}</th><td>{count}</td></tr>
            {/each}
          </tbody>
        </table>
      </details>
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
  .trend-heading {
    display: flex;
    justify-content: space-between;
    align-items: end;
    gap: 1rem;
    margin-top: 1.5rem;
  }
  .trend-heading p {
    margin-top: 0.3rem;
    color: var(--text-muted);
    font-size: 0.8rem;
    line-height: 1.4;
  }
  .trend-scroll {
    max-width: 100%;
    overflow-x: auto;
    padding: 0.5rem 0;
  }
  .trend-scroll :global(.bar-chart) {
    min-width: var(--trend-width);
  }
  .trend-data {
    margin-top: 0.5rem;
    color: var(--text-muted);
    font-size: var(--type-label);
  }
  .trend-data summary {
    width: fit-content;
    cursor: pointer;
  }
  .trend-data table {
    width: 100%;
    margin-top: 0.5rem;
    border-collapse: collapse;
    color: var(--text);
    font-variant-numeric: tabular-nums;
  }
  .trend-data th,
  .trend-data td {
    padding: 0.35rem 0.5rem;
    border-bottom: 1px solid var(--border);
    text-align: right;
  }
  .trend-data th:first-child {
    text-align: left;
  }
  .trend-data thead th {
    color: var(--text-muted);
    font-weight: 650;
  }
  @media (max-width: 640px) {
    .trend-heading {
      align-items: stretch;
      flex-direction: column;
    }
  }
  .trend-scroll:focus-visible,
  select:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
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
