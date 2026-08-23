<script lang="ts">
  let {
    label,
    period,
    onDrill,
  }: {
    label: string;
    period: string;
    onDrill: (period: string) => void;
  } = $props();
</script>

<button type="button" onclick={() => onDrill(period)}>{label}</button>

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
