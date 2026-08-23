<script lang="ts">
  let {
    label,
    period,
    value,
    onDrill,
  }: {
    label: string;
    period: string;
    value?: number | null;
    onDrill: (period: string) => void;
  } = $props();

  const evidence = $derived(
    value === undefined
      ? ""
      : value === null
        ? "no steps recorded"
        : `${value.toLocaleString()} steps`,
  );
</script>

<button
  type="button"
  aria-label={evidence ? `${label}, ${evidence}` : undefined}
  data-period-drill
  data-period={label}
  data-evidence={evidence || undefined}
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
