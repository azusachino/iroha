<script lang="ts">
  // A compact month-grid day picker. Days that actually have data are dotted,
  // so scrubbing lands on real days. Emits the chosen 'YYYY-MM-DD'.
  let {
    value,
    days,
    max,
    onselect,
  }: {
    value: string;
    days: Set<string>;
    max?: string;
    onselect: (day: string) => void;
  } = $props();

  // Shown month follows the selected value, unless the user browsed months.
  import { formatMonth } from "$lib/format";
  let monthOverride = $state<string | null>(null);
  $effect(() => {
    void value;
    monthOverride = null; // a new selection re-centers on its month
  });
  const view = $derived(monthOverride ?? value.slice(0, 7)); // 'YYYY-MM'

  const WEEKDAYS = ["S", "M", "T", "W", "T", "F", "S"];

  const cells = $derived.by<(string | null)[]>(() => {
    const [y, m] = view.split("-").map(Number);
    const startDow = new Date(Date.UTC(y, m - 1, 1)).getUTCDay();
    const daysInMonth = new Date(Date.UTC(y, m, 0)).getUTCDate();
    const out: (string | null)[] = [];
    for (let i = 0; i < startDow; i++) out.push(null);
    for (let d = 1; d <= daysInMonth; d++)
      out.push(`${view}-${String(d).padStart(2, "0")}`);
    return out;
  });
  const monthLabel = $derived(formatMonth(view));
  const focusableDay = $derived(value);
  let dayGrid: HTMLDivElement;

  function dayLabel(day: string): string {
    return new Date(`${day}T00:00:00Z`).toLocaleDateString(undefined, {
      weekday: "long",
      month: "long",
      day: "numeric",
      year: "numeric",
      timeZone: "UTC",
    });
  }

  function handleDayKeydown(event: KeyboardEvent, day: string) {
    const index = cells.indexOf(day);
    if (index < 0) return;
    const delta =
      event.key === "ArrowLeft"
        ? -1
        : event.key === "ArrowRight"
          ? 1
          : event.key === "ArrowUp"
            ? -7
            : event.key === "ArrowDown"
              ? 7
              : event.key === "Home"
                ? -(index % 7)
                : event.key === "End"
                  ? 6 - (index % 7)
                  : 0;
    if (delta === 0) return;
    event.preventDefault();
    event.stopPropagation();
    const nextDay = cells[index + delta];
    if (!nextDay || (max && nextDay > max)) return;
    dayGrid
      ?.querySelector<HTMLButtonElement>(`button[data-day="${nextDay}"]`)
      ?.focus();
  }

  function shiftMonth(delta: number) {
    const [y, m] = view.split("-").map(Number);
    monthOverride = new Date(Date.UTC(y, m - 1 + delta, 1))
      .toISOString()
      .slice(0, 7);
  }
</script>

<div class="picker" role="group" aria-label={`Choose a day in ${monthLabel}`}>
  <div class="pk-head" aria-live="polite">
    <button
      type="button"
      aria-label="Previous month"
      onclick={() => shiftMonth(-1)}>‹</button
    >
    <span role="heading" aria-level="2">{monthLabel}</span>
    <button type="button" aria-label="Next month" onclick={() => shiftMonth(1)}
      >›</button
    >
  </div>
  <div
    class="pk-grid"
    bind:this={dayGrid}
    role="grid"
    aria-label={`Calendar for ${monthLabel}`}
  >
    {#each WEEKDAYS as w, index}
      <span
        class="dow"
        role="columnheader"
        aria-label={index === 0
          ? "Sunday"
          : index === 1
            ? "Monday"
            : index === 2
              ? "Tuesday"
              : index === 3
                ? "Wednesday"
                : index === 4
                  ? "Thursday"
                  : index === 5
                    ? "Friday"
                    : "Saturday"}>{w}</span
      >
    {/each}
    {#each cells as d}
      {#if d == null}
        <span role="gridcell" aria-hidden="true"></span>
      {:else}
        <button
          type="button"
          role="gridcell"
          class="day"
          class:selected={d === value}
          class:has={days.has(d)}
          disabled={max ? d > max : false}
          data-day={d}
          tabindex={d === focusableDay ? 0 : -1}
          aria-label={dayLabel(d)}
          aria-selected={d === value}
          aria-current={d === value ? "date" : undefined}
          onkeydown={(event) => handleDayKeydown(event, d)}
          onclick={() => onselect(d)}
        >
          {Number(d.slice(8))}
        </button>
      {/if}
    {/each}
  </div>
</div>

<style>
  .picker {
    width: 15rem;
    padding: 0.75rem;
  }
  .pk-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 0.5rem;
    font-weight: 650;
    color: var(--text);
  }
  .pk-head button {
    appearance: none;
    border: 0;
    background: transparent;
    color: var(--text-muted);
    font-size: 1.1rem;
    width: 1.8rem;
    height: 1.8rem;
    border-radius: var(--radius);
    cursor: pointer;
  }
  .pk-head button:hover {
    color: var(--accent);
    background: color-mix(in srgb, var(--accent) 12%, transparent);
  }
  .pk-grid {
    display: grid;
    grid-template-columns: repeat(7, 1fr);
    gap: 2px;
  }
  .dow {
    text-align: center;
    font-size: 0.68rem;
    color: var(--text-muted);
    padding-bottom: 0.25rem;
  }
  .day {
    appearance: none;
    border: 0;
    background: transparent;
    color: var(--text);
    aspect-ratio: 1;
    border-radius: var(--radius);
    font-size: 0.8rem;
    cursor: pointer;
    position: relative;
  }
  .day:hover:not(:disabled) {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
  }
  .day:disabled {
    color: color-mix(in srgb, var(--text-muted) 45%, transparent);
    cursor: default;
  }
  /* dot marks days that actually have data */
  .day.has::after {
    content: "";
    position: absolute;
    bottom: 3px;
    left: 50%;
    transform: translateX(-50%);
    width: 4px;
    height: 4px;
    border-radius: 50%;
    background: var(--accent);
  }
  .day.selected {
    background: var(--accent);
    color: #08131a;
    font-weight: 700;
    box-shadow: var(--accent-glow);
  }
  .day.selected.has::after {
    background: #08131a;
  }
</style>
