<script lang="ts">
  import type { DesignLanguage } from "../theme/themes";

  export type SelectControlOption = {
    value: string;
    label: string;
  };

  let {
    label,
    value,
    options,
    ariaLabel = label,
    disabled = false,
    markerColor,
    appearance = "grapher",
    onChange,
  }: {
    label: string;
    value: string;
    options: SelectControlOption[];
    ariaLabel?: string;
    disabled?: boolean;
    markerColor?: string;
    appearance?: DesignLanguage;
    onChange: (value: string) => void;
  } = $props();
</script>

<label class="select-control" data-appearance={appearance}>
  <span class="select-label">
    {#if markerColor}
      <i
        class="select-marker"
        style={"background:" + markerColor}
        aria-hidden="true"
      ></i>
    {/if}
    {label}
  </span>
  <select
    aria-label={ariaLabel}
    {disabled}
    value={value}
    onchange={(event) =>
      onChange((event.currentTarget as HTMLSelectElement).value)}
  >
    {#each options as option (option.value)}
      <option value={option.value}>{option.label}</option>
    {/each}
  </select>
</label>

<style>
  .select-control {
    display: grid;
    min-width: 9rem;
    gap: var(--space-1);
  }

  .select-label {
    display: inline-flex;
    align-items: center;
    min-height: 0.85rem;
    color: var(--text-muted);
    font-size: var(--type-caption);
    font-weight: 700;
    letter-spacing: 0.08em;
    line-height: 1;
    text-transform: uppercase;
  }

  .select-marker {
    width: 0.55rem;
    height: 0.55rem;
    margin-right: var(--space-1);
    border-radius: 50%;
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--surface-2) 85%, transparent);
  }

  select {
    width: 100%;
    min-height: 2rem;
    padding: var(--space-1) var(--space-2);
    border: 1px solid var(--border);
    border-radius: 7px;
    background: var(--surface-2);
    color: var(--text);
    font: inherit;
    font-size: var(--type-label);
  }

  select:disabled {
    cursor: not-allowed;
    opacity: 0.45;
  }

  .select-control[data-appearance="grapher"] select {
    border-radius: 2px;
    border-bottom-width: 2px;
    font-variant-numeric: tabular-nums;
  }

  @media (max-width: 640px) {
    .select-control,
    select {
      width: 100%;
    }
  }
</style>
