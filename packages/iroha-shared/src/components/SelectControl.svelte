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
    gap: 0.3rem;
  }

  .select-label {
    display: inline-flex;
    align-items: center;
    min-height: 0.85rem;
    color: var(--text-muted);
    font-size: 0.68rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    line-height: 1;
    text-transform: uppercase;
  }

  .select-marker {
    width: 0.55rem;
    height: 0.55rem;
    margin-right: 0.35rem;
    border-radius: 50%;
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--surface-2) 85%, transparent);
  }

  select {
    width: 100%;
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

  .select-control[data-appearance="atlas"] select {
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

  .select-control[data-appearance="grapher"] select {
    border-radius: 2px;
    border-bottom-width: 2px;
    font-variant-numeric: tabular-nums;
  }

  .select-control[data-appearance="field-journal"] select {
    border-style: dashed;
    box-shadow: 2px 2px 0 color-mix(in srgb, var(--accent-2) 20%, transparent);
  }

  .select-control[data-appearance="phenology"] select {
    border-radius: 999px;
    background: color-mix(in srgb, var(--accent) 9%, var(--surface-2));
  }

  .select-control[data-appearance="sound-map"] select {
    border-inline-width: 2px;
    box-shadow: inset 0 -2px 0
      color-mix(in srgb, var(--accent) 35%, transparent);
  }

  .select-control[data-appearance="archive"] select {
    border: 3px double var(--border);
    border-radius: 0;
  }

  @media (max-width: 640px) {
    .select-control,
    select {
      width: 100%;
    }
  }
</style>
