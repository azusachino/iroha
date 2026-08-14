<script lang="ts">
  import { shiftMonth } from "./month";
  import { formatCanonicalMonth } from "./format";
  import type { DesignLanguage } from "./themes";

  export type PeriodOption = { value: string; label: string };

  let {
    year,
    month,
    years,
    months,
    monthDisabled = false,
    showAllYears = true,
    appearance = "grapher",
    surface = "panel",
    onYear,
    onMonth,
  }: {
    year: string;
    month: string;
    years: (PeriodOption | string)[];
    months: PeriodOption[];
    monthDisabled?: boolean;
    showAllYears?: boolean;
    surface?: "panel" | "inline";
    appearance?: DesignLanguage;
    onYear: (value: string) => void;
    onMonth: (value: string) => void;
  } = $props();

  const yearOptions = $derived(
    years.map((option) =>
      typeof option === "string" ? { value: option, label: option } : option,
    ),
  );

  function monthOptionLabel(option: PeriodOption): string {
    if (/^\d{4}-\d{1,2}$/.test(option.value)) {
      return formatCanonicalMonth(option.value);
    }
    if (/^\d{1,2}$/.test(option.value) && /^\d{4}$/.test(year)) {
      return formatCanonicalMonth(`${year}-${option.value}`);
    }
    return option.label;
  }

  function shiftPeriod(delta: number) {
    if (/^\d{4}-(?:0[1-9]|1[0-2])$/.test(month)) {
      onMonth(shiftMonth(month, delta));
      return;
    }
    if (!/^\d{4}$/.test(year)) return;

    if (!/^(?:[1-9]|1[0-2])$/.test(month)) {
      const nextYear = Number(year) + delta;
      onYear(String(nextYear));
      onMonth("");
      return;
    }
    const next = new Date(Date.UTC(Number(year), Number(month) - 1 + delta, 1));
    onYear(String(next.getUTCFullYear()));
    onMonth(String(next.getUTCMonth() + 1));
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
    if (event.altKey || event.ctrlKey || event.metaKey || event.shiftKey)
      return;
    const target = event.target as Element | null;
    if (target?.matches("input, textarea, select, [contenteditable='true']"))
      return;
    event.preventDefault();
    shiftPeriod(event.key === "ArrowLeft" ? -1 : 1);
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div
  class:panel={surface === "panel"}
  class:inline={surface === "inline"}
  class="period-controls"
  data-appearance={appearance}
  aria-label="Period filters"
>
  <label>
    <span>Year</span>
    <select
      aria-label="Filter by year"
      value={year}
      onchange={(event) =>
        onYear((event.currentTarget as HTMLSelectElement).value)}
    >
      {#if showAllYears}<option value="">Lifetime</option>{/if}
      {#each yearOptions as option (option.value)}
        <option value={option.value}>{monthOptionLabel(option)}</option>
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
        <option value={option.value}>{monthOptionLabel(option)}</option>
      {/each}
    </select>
  </label>
</div>

<style>
  .period-controls {
    display: flex;
    flex-wrap: wrap;
    align-items: end;
    gap: 0.75rem;
  }

  .period-controls.panel {
    padding: 0.65rem 0.8rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--tile-surface);
    box-shadow: var(--tile-shadow);
  }

  .period-controls.inline {
    padding: 0;
    border: 0;
    background: transparent;
    box-shadow: none;
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

  .period-controls[data-appearance="atlas"] select {
    border-width: 2px;
    border-radius: 2px;
    background-image:
      linear-gradient(
        color-mix(in srgb, var(--accent) 8%, transparent) 1px,
        transparent 1px
      ),
      linear-gradient(
        90deg,
        color-mix(in srgb, var(--accent) 8%, transparent) 1px,
        transparent 1px
      );
    background-size: 10px 10px;
  }

  .period-controls[data-appearance="grapher"] select {
    border-radius: 2px;
    border-bottom-width: 2px;
    font-variant-numeric: tabular-nums;
  }

  .period-controls[data-appearance="field-journal"] select {
    border-style: dashed;
    box-shadow: 2px 2px 0 color-mix(in srgb, var(--accent-2) 20%, transparent);
  }

  .period-controls[data-appearance="phenology"] select {
    border-radius: 999px;
    background: color-mix(in srgb, var(--accent) 9%, var(--surface-2));
  }

  .period-controls[data-appearance="sound-map"] select {
    border-inline-width: 2px;
    box-shadow: inset 0 -2px 0
      color-mix(in srgb, var(--accent) 35%, transparent);
  }

  .period-controls[data-appearance="archive"] select {
    border: 3px double var(--border);
    border-radius: 0;
  }

  @media (max-width: 520px) {
    .period-controls,
    label,
    select {
      width: 100%;
    }
  }
</style>
