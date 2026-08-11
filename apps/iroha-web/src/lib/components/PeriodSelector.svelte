<script lang="ts">
  export type PeriodOption = { value: string; label: string };

  let {
    year,
    month,
    years,
    months,
    monthDisabled = false,
    showAllYears = true,
    onYear,
    onMonth,
  }: {
    year: string;
    month: string;
    years: (PeriodOption | string)[];
    months: PeriodOption[];
    monthDisabled?: boolean;
    showAllYears?: boolean;
    onYear: (value: string) => void;
    onMonth: (value: string) => void;
  } = $props();

  const yearOptions = $derived(
    years.map((option) =>
      typeof option === "string" ? { value: option, label: option } : option,
    ),
  );
</script>

<div class="period-controls" aria-label="Period filters">
  <label>
    <span>Year</span>
    <select
      aria-label="Filter by year"
      value={year}
      onchange={(event) =>
        onYear((event.currentTarget as HTMLSelectElement).value)}
    >
      {#if showAllYears}<option value="">All years</option>{/if}
      {#each yearOptions as option (option.value)}
        <option value={option.value}>{option.label}</option>
      {/each}
    </select>
  </label>
  <label>
    <span>Month</span>
    <select
      aria-label="Filter by month"
      value={month}
      disabled={monthDisabled}
      onchange={(event) =>
        onMonth((event.currentTarget as HTMLSelectElement).value)}
    >
      <option value="">All months</option>
      {#each months as option (option.value)}
        <option value={option.value}>{option.label}</option>
      {/each}
    </select>
  </label>
</div>

<style>
  .period-controls {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
  }

  label {
    display: grid;
    gap: 0.3rem;
  }

  label span {
    color: var(--text-muted);
    font-size: 0.68rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  select {
    min-width: 9rem;
    min-height: 2rem;
    padding: 0.35rem 0.55rem;
    border: 1px solid var(--border);
    border-radius: 7px;
    background: var(--surface-2);
    color: var(--text);
    font: inherit;
    font-size: 0.76rem;
  }

  select:disabled {
    cursor: not-allowed;
    opacity: 0.45;
  }

  @media (max-width: 520px) {
    .period-controls,
    label,
    select {
      width: 100%;
    }
  }
</style>
