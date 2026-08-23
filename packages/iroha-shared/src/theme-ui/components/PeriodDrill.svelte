<script module lang="ts">
  export function periodDrillAccessibleName(
    label: string,
    value: number | null,
  ): string {
    const evidence =
      value === null ? "no steps recorded" : `${value.toLocaleString()} steps`;
    return `${label}, ${evidence}`;
  }
</script>

<script lang="ts">
  let {
    label,
    period,
    value,
    onDrill,
  }: {
    label: string;
    period: string;
    value: number | null;
    onDrill: (period: string) => void;
  } = $props();

  const accessibleName = $derived(periodDrillAccessibleName(label, value));
</script>

<button
  type="button"
  aria-label={accessibleName}
  data-period-drill
  onclick={() => onDrill(period)}>{label}</button
>

<style>
  button {
    min-height: var(--control-target-min);
    padding: 0;
    border: 0;
    background: transparent;
    color: inherit;
    font: inherit;
    cursor: pointer;
    text-decoration: underline;
    text-underline-offset: 0.2em;
  }
</style>
