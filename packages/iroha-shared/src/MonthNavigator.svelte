<script lang="ts">
  import { currentMonth, formatMonth, shiftMonth } from "./month";
  import type { DesignLanguage } from "./themes";

  const monthNames = Array.from({ length: 12 }, (_, index) =>
    new Date(Date.UTC(2026, index, 1)).toLocaleDateString(undefined, {
      month: "short",
      timeZone: "UTC",
    }),
  );

  let {
    month,
    onMonth,
    disabled = false,
    appearance = "grapher",
  }: {
    month: string;
    onMonth: (value: string) => void;
    disabled?: boolean;
    appearance?: DesignLanguage;
  } = $props();

  let open = $state(false);
  let pickerYear = $state(0);
  let root: HTMLDivElement;

  function openPicker() {
    pickerYear = Number(month.slice(0, 4));
    open = true;
  }

  function choose(value: string) {
    open = false;
    onMonth(value);
  }

  function handleWindowClick(event: MouseEvent) {
    if (open && root && !root.contains(event.target as Node)) open = false;
  }

  function handleWindowKeydown(event: KeyboardEvent) {
    if (event.key === "Escape") {
      open = false;
      return;
    }
    handleNavigatorKeydown(event);
  }

  function handleNavigatorKeydown(event: KeyboardEvent) {
    if (!root?.contains(document.activeElement)) return;
    const target = event.target as HTMLElement | null;
    if (
      target?.matches("input, textarea, select, [contenteditable='true']")
    ) {
      return;
    }
    if (event.key === "ArrowLeft" || event.key === "ArrowRight") {
      event.preventDefault();
      choose(shiftMonth(month, event.key === "ArrowLeft" ? -1 : 1));
    }
  }
</script>

<svelte:window onclick={handleWindowClick} onkeydown={handleWindowKeydown} />

<div
  class="month-navigator"
  data-appearance={appearance}
  bind:this={root}
  aria-label="Month selector"
>
  <button
    class="step"
    type="button"
    aria-label="Previous month"
    {disabled}
    onclick={() => choose(shiftMonth(month, -1))}
  >
    ‹
  </button>
  <button
    class="month-trigger"
    type="button"
    aria-expanded={open}
    aria-haspopup="dialog"
    {disabled}
    onclick={(event) => {
      event.stopPropagation();
      openPicker();
    }}
  >
    <span>Month</span>
    <strong>{formatMonth(month)}</strong>
    <i class="chevron" aria-hidden="true"></i>
  </button>
  <button
    class="step"
    type="button"
    aria-label="Next month"
    {disabled}
    onclick={() => choose(shiftMonth(month, 1))}
  >
    ›
  </button>

  {#if open}
    <div class="month-popover" role="dialog" aria-label="Choose a month">
      <header>
        <button
          class="year-step"
          type="button"
          aria-label="Previous year"
          onclick={() => (pickerYear -= 1)}
        >
          ‹
        </button>
        <strong>{pickerYear}</strong>
        <button
          class="year-step"
          type="button"
          aria-label="Next year"
          onclick={() => (pickerYear += 1)}
        >
          ›
        </button>
      </header>
      <div class="month-grid">
        {#each monthNames as label, index}
          {@const value = `${pickerYear}-${String(index + 1).padStart(2, "0")}`}
          <button
            type="button"
            class:selected={value === month}
            class:current={value === currentMonth()}
            aria-current={value === month ? "date" : undefined}
            onclick={() => choose(value)}
          >
            {label}
          </button>
        {/each}
      </div>
      <button
        class="today"
        type="button"
        onclick={() => choose(currentMonth())}
      >
        Jump to today
      </button>
    </div>
  {/if}
</div>

<style>
  .month-navigator {
    position: relative;
    display: inline-flex;
    align-items: center;
    gap: 0.2rem;
    padding: 0.25rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface-2);
    box-shadow: var(--tile-shadow);
  }

  .month-navigator[data-appearance="atlas"] {
    border-width: 2px;
    background-image:
      linear-gradient(color-mix(in srgb, var(--accent) 8%, transparent) 1px, transparent 1px),
      linear-gradient(90deg, color-mix(in srgb, var(--accent) 8%, transparent) 1px, transparent 1px);
    background-size: 12px 12px;
  }

  .month-navigator[data-appearance="grapher"] {
    border-radius: 2px;
    border-bottom: 3px solid var(--accent);
    box-shadow: none;
    font-variant-numeric: tabular-nums;
  }

  .month-navigator[data-appearance="field-journal"] {
    border-style: dashed;
    box-shadow: 3px 3px 0 color-mix(in srgb, var(--accent-2) 24%, transparent);
  }

  .month-navigator[data-appearance="phenology"] {
    padding: 0.35rem;
    border-radius: 999px;
    background: color-mix(in srgb, var(--accent) 9%, var(--surface-2));
  }

  .month-navigator[data-appearance="sound-map"] {
    border-inline: 3px solid var(--accent-2);
    box-shadow: inset 0 -3px 0 color-mix(in srgb, var(--accent) 35%, transparent);
  }

  .month-navigator[data-appearance="archive"] {
    border: 3px double var(--border);
    border-radius: 0;
    box-shadow: 2px 2px 0 color-mix(in srgb, var(--text-muted) 35%, transparent);
  }

  button {
    font: inherit;
  }

  .step,
  .year-step {
    display: grid;
    place-items: center;
    border: 0;
    border-radius: calc(var(--radius) - 5px);
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
  }

  .step {
    width: 2rem;
    height: 2.25rem;
    font-size: 1.45rem;
    line-height: 1;
  }

  .step:hover,
  .step:focus-visible,
  .year-step:hover,
  .year-step:focus-visible {
    background: color-mix(in srgb, var(--accent) 12%, transparent);
    color: var(--accent);
  }

  .month-trigger {
    display: grid;
    min-width: 10rem;
    gap: 0.05rem;
    padding: 0.3rem 0.7rem;
    border: 0;
    border-radius: calc(var(--radius) - 5px);
    background: transparent;
    color: var(--text);
    cursor: pointer;
    text-align: center;
  }

  .month-trigger:hover,
  .month-trigger:focus-visible {
    background: color-mix(in srgb, var(--accent) 10%, transparent);
  }

  .month-trigger span {
    color: var(--text-muted);
    font-size: 0.62rem;
    font-weight: 700;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .month-trigger strong {
    font-size: 0.88rem;
  }

  .chevron {
    width: 0.46rem;
    height: 0.46rem;
    justify-self: center;
    border-right: 1.5px solid currentColor;
    border-bottom: 1.5px solid currentColor;
    color: var(--accent);
    transform: rotate(45deg) translateY(-1px);
    transition: transform 160ms ease;
  }
  .month-trigger[aria-expanded="true"] .chevron {
    transform: rotate(225deg) translate(-1px, -1px);
  }

  .month-popover {
    position: absolute;
    z-index: 20;
    top: calc(100% + 0.5rem);
    left: 50%;
    width: 15rem;
    padding: 0.75rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface-2);
    box-shadow: var(--shadow, 0 16px 40px rgb(0 0 0 / 25%));
    transform: translateX(-50%);
  }

  .month-popover header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 0.55rem;
    color: var(--text);
  }

  .year-step {
    width: 1.8rem;
    height: 1.8rem;
    font-size: 1.2rem;
  }

  .month-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 0.3rem;
  }

  .month-grid button,
  .today {
    min-height: 2.15rem;
    border: 1px solid transparent;
    border-radius: calc(var(--radius) - 6px);
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    font-size: 0.76rem;
  }

  .month-grid button:hover,
  .month-grid button:focus-visible,
  .month-grid button.current {
    border-color: color-mix(in srgb, var(--accent) 45%, transparent);
    color: var(--text);
  }

  .month-grid button.selected {
    border-color: var(--accent);
    background: var(--accent);
    color: var(--on-accent, #08131a);
    font-weight: 700;
  }

  .today {
    width: 100%;
    margin-top: 0.65rem;
    border-color: var(--border);
    color: var(--accent);
  }

  .today:hover,
  .today:focus-visible {
    background: color-mix(in srgb, var(--accent) 10%, transparent);
  }

  button:disabled {
    cursor: default;
    opacity: 0.45;
  }

  @media (max-width: 520px) {
    .month-popover {
      left: 0;
      transform: none;
    }
  }
</style>
