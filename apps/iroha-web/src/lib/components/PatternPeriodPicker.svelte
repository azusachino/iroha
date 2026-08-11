<script lang="ts">
  type Gran = "day" | "month" | "year";

  let {
    gran,
    availableYears,
    availableMonths,
    selectedYear,
    selectedMonth,
    activeYear,
    activeMonth,
    scopeLabel,
    onYear,
    onMonth,
  }: {
    gran: Gran;
    availableYears: string[];
    availableMonths: string[];
    selectedYear: string;
    selectedMonth: string;
    activeYear: string;
    activeMonth: string;
    scopeLabel: string;
    onYear: (value: string) => void;
    onMonth: (value: string) => void;
  } = $props();

  const monthsInScope = $derived(
    availableMonths.filter((month) => month.startsWith(activeYear)),
  );
  const yearValue = $derived(gran === "year" ? selectedYear : activeYear);
  const monthValue = $derived(gran === "day" ? activeMonth : selectedMonth);

  function monthLabel(period: string): string {
    return new Date(`${period}-01T00:00:00Z`).toLocaleDateString(undefined, {
      month: "long",
      year: "numeric",
      timeZone: "UTC",
    });
  }
</script>

<details class="period-picker">
  <summary>
    <span>Browse period</span>
    <strong>{scopeLabel}</strong>
  </summary>
  <div class="period-menu">
    <label>
      <span>Year</span>
      <select
        value={yearValue}
        onchange={(event) =>
          onYear((event.currentTarget as HTMLSelectElement).value)}
      >
        {#if gran === "year"}<option value="">All years</option>{/if}
        {#each availableYears as year (year)}
          <option value={year}>{year}</option>
        {/each}
      </select>
    </label>

    {#if gran !== "year"}
      <label>
        <span>Month</span>
        <select
          value={monthValue}
          onchange={(event) =>
            onMonth((event.currentTarget as HTMLSelectElement).value)}
        >
          {#if gran === "month"}<option value="">All months</option>{/if}
          {#each monthsInScope as month (month)}
            <option value={month}>{monthLabel(month)}</option>
          {/each}
        </select>
      </label>
    {/if}

    <p>
      Choose a period to focus the patterns without changing the view scale.
    </p>
  </div>
</details>

<style>
  .period-picker {
    position: relative;
    z-index: 2;
  }

  summary {
    display: inline-flex;
    align-items: center;
    gap: 0.65rem;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    cursor: pointer;
    list-style: none;
  }

  summary::-webkit-details-marker {
    display: none;
  }

  summary::after {
    content: "⌄";
    color: var(--text-muted);
    font-size: 0.9rem;
  }

  details[open] summary {
    border-color: var(--accent);
    color: var(--accent);
  }

  details[open] summary::after {
    content: "⌃";
  }

  summary span {
    color: var(--text-muted);
    font-size: 0.78rem;
  }

  summary strong {
    font-size: 0.85rem;
    font-weight: 650;
  }

  .period-menu {
    position: absolute;
    top: calc(100% + 0.45rem);
    left: 0;
    display: grid;
    grid-template-columns: repeat(2, minmax(8rem, 1fr));
    gap: 0.7rem;
    width: min(26rem, calc(100vw - 2rem));
    padding: 0.8rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    box-shadow: var(--tile-shadow);
  }

  label {
    display: grid;
    gap: 0.3rem;
  }

  label span {
    color: var(--text-muted);
    font-size: 0.72rem;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }

  select {
    min-width: 0;
    padding: 0.45rem 0.5rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 2px);
    background: var(--surface-2);
    color: var(--text);
    font: inherit;
  }

  .period-menu p {
    grid-column: 1 / -1;
    margin: 0;
    color: var(--text-muted);
    font-size: 0.76rem;
    line-height: 1.4;
  }

  @media (max-width: 520px) {
    .period-menu {
      grid-template-columns: 1fr;
    }
  }
</style>
