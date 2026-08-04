<script lang="ts">
  let {
    label,
    options,
    value,
    summary,
    onChange,
  }: {
    label: string;
    options: { value: string; label: string }[];
    value: string;
    summary: string;
    onChange: (value: string) => void;
  } = $props();
</script>

<div class="scope-controls" aria-label={`${label} scope`}>
  <div class="scope-heading">
    <span>View window</span>
    <strong>{label}</strong>
  </div>
  <div class="select-wrap">
    <label for="daily-scope">Choose {label.toLowerCase()}</label>
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
  <span class="scope-summary">{summary}</span>
</div>

<style>
  .scope-controls {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 0.75rem;
    width: fit-content;
    max-width: 100%;
    margin: 0 0 1.5rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 0.45rem 0.55rem 0.45rem 0.8rem;
    background: color-mix(in srgb, var(--surface) 90%, var(--accent) 10%);
    box-shadow: 0 0.5rem 1.5rem color-mix(in srgb, var(--text) 8%, transparent);
    color: var(--text-muted);
  }

  .scope-heading {
    display: grid;
    gap: 0.08rem;
    line-height: 1.1;
  }

  .scope-heading span {
    color: var(--accent);
    font-size: 0.6rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .scope-heading strong {
    color: var(--text);
    font-size: 0.78rem;
  }

  .select-wrap {
    position: relative;
  }

  .select-wrap label {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
    white-space: nowrap;
  }

  select {
    min-width: 10rem;
    appearance: none;
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 0.48rem 2rem 0.48rem 0.8rem;
    background: var(--surface);
    color: var(--text);
    font: inherit;
    font-size: 0.78rem;
    cursor: pointer;
  }

  .select-wrap::after {
    position: absolute;
    top: 50%;
    right: 0.75rem;
    color: var(--accent);
    content: "⌄";
    pointer-events: none;
    transform: translateY(-55%);
  }

  select:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
  }

  .scope-summary {
    padding-right: 0.35rem;
    color: var(--text-muted);
    font-size: 0.7rem;
    white-space: nowrap;
  }
</style>
