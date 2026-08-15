<script lang="ts">
  import { currentMonth, shiftMonthWithin } from "./month";
  import { formatCanonicalMonth } from "./format";
  import SelectControl, {
    type SelectControlOption,
  } from "./SelectControl.svelte";
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

  const yearSelectOptions = $derived<SelectControlOption[]>([
    ...(showAllYears ? [{ value: "", label: "Lifetime" }] : []),
    ...yearOptions.map((option) => ({
      value: option.value,
      label: monthOptionLabel(option),
    })),
  ]);

  const monthSelectOptions = $derived<SelectControlOption[]>([
    { value: "", label: "All months" },
    ...months.map((option) => ({
      value: option.value,
      label: monthOptionLabel(option),
    })),
  ]);

  function shiftPeriod(delta: number) {
    const maximumMonth = currentMonth();

    if (/^\d{4}-(?:0[1-9]|1[0-2])$/.test(month)) {
      onMonth(shiftMonthWithin(month, delta, maximumMonth));
      return;
    }
    if (!/^\d{4}$/.test(year)) return;

    if (!/^(?:[1-9]|1[0-2])$/.test(month)) {
      const nextYear = Number(year) + delta;
      onYear(
        delta > 0 && nextYear > Number(maximumMonth.slice(0, 4))
          ? maximumMonth.slice(0, 4)
          : String(nextYear),
      );
      onMonth("");
      return;
    }
    const current = `${year}-${String(Number(month)).padStart(2, "0")}`;
    const bounded = shiftMonthWithin(current, delta, maximumMonth);
    onYear(bounded.slice(0, 4));
    onMonth(String(Number(bounded.slice(5, 7))));
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
  <SelectControl
    label="Year"
    value={year}
    options={yearSelectOptions}
    appearance={appearance}
    ariaLabel="Filter by year"
    onChange={onYear}
  />
  <SelectControl
    label="Month"
    value={month}
    options={monthSelectOptions}
    appearance={appearance}
    ariaLabel="Filter by month"
    disabled={monthDisabled}
    onChange={onMonth}
  />
</div>

<style>
  .period-controls {
    display: flex;
    min-width: 0;
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

  @media (max-width: 640px) {
    .period-controls {
      flex-direction: column;
      align-items: stretch;
      gap: 0.55rem;
      width: 100%;
    }
  }
</style>
