<script lang="ts">
  import { formatMonth, shiftMonth } from "./month";

  let {
    month,
    onMonth,
    disabled = false,
  }: {
    month: string;
    onMonth: (value: string) => void;
    disabled?: boolean;
  } = $props();
</script>

<div class="month-navigator" aria-label="Month selector">
  <button
    type="button"
    aria-label="Previous month"
    {disabled}
    onclick={() => onMonth(shiftMonth(month, -1))}
  >
    <span aria-hidden="true">←</span>
  </button>
  <label>
    <span>Month</span>
    <input
      aria-label="Select month"
      type="month"
      value={month}
      {disabled}
      onchange={(event) =>
        onMonth((event.currentTarget as HTMLInputElement).value)}
    />
    <strong>{formatMonth(month)}</strong>
  </label>
  <button
    type="button"
    aria-label="Next month"
    {disabled}
    onclick={() => onMonth(shiftMonth(month, 1))}
  >
    <span aria-hidden="true">→</span>
  </button>
</div>

<style>
  .month-navigator {
    display: inline-flex;
    align-items: center;
    gap: 0.55rem;
    padding: 0.35rem;
    border: 1px solid var(--border);
    border-radius: 0.75rem;
    background: var(--surface-2);
  }

  button {
    display: grid;
    width: 2rem;
    height: 2rem;
    place-items: center;
    border: 0;
    border-radius: 0.45rem;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
  }

  button:hover,
  button:focus-visible {
    background: var(--surface-1, var(--surface));
    color: var(--text);
  }

  label {
    position: relative;
    display: grid;
    min-width: 10rem;
    gap: 0.08rem;
    text-align: center;
  }

  label span {
    color: var(--text-muted);
    font-size: 0.62rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  input {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    opacity: 0;
    cursor: pointer;
  }

  strong {
    font-size: 0.86rem;
  }
</style>
