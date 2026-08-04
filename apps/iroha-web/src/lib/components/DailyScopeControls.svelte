<script lang="ts">
  let {
    label,
    options,
    value,
    onChange,
  }: {
    label: string;
    options: { value: string; label: string }[];
    value: string;
    onChange: (value: string) => void;
  } = $props();
</script>

<div class="scope-controls" aria-label={`${label} scope`}>
  <label for="daily-scope">{label}</label>
  <select
    id="daily-scope"
    {value}
    onchange={(event) =>
      onChange((event.currentTarget as HTMLSelectElement).value)}
  >
    {#each options as option}
      <option value={option.value}>{option.label}</option>
    {/each}
  </select>
</div>

<style>
  .scope-controls {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    margin: 0 0 1rem;
    color: var(--text-muted);
    font-size: 0.74rem;
  }

  label {
    color: var(--accent);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  select {
    min-width: 9rem;
    border: 1px solid var(--border);
    border-radius: 0.25rem;
    padding: 0.4rem 0.6rem;
    background: var(--surface);
    color: var(--text);
    font: inherit;
  }
</style>
