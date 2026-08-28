<script lang="ts">
  import type { PanelRow } from "./metric-panel";

  // A DOM/CSS bar list, not an ECharts BarChart, for two reasons: a
  // coverage ratio (observed/expected days) is a percentage, so every bar
  // shares one honest 0-100% scale by construction instead of an
  // axis auto-scaled to whatever row happens to be largest; and real DOM
  // means a row's icon (Svelte component) can actually render here --
  // ECharts draws to canvas, so a component can't sit inside its axis
  // labels. MetricTable remains the accessible/exact-values source (this
  // is presentational, matching BarChart's own aria: {enabled:false}).
  let { rows }: { rows: PanelRow[] } = $props();

  function clampPct(value: number | null): number {
    if (value == null) return 0;
    return Math.min(100, Math.max(0, value));
  }
</script>

<ul class="coverage-bars" aria-hidden="true">
  {#each rows as row}
    <li>
      <span class="coverage-icon" style:color={`var(${row.colorVar ?? "--accent"})`}>
        {#if row.icon}<row.icon size={15} />{/if}
      </span>
      <span class="coverage-label">{row.label}</span>
      <span class="coverage-track">
        <span
          class="coverage-fill"
          style:width={`${clampPct(row.value)}%`}
          style:background={`var(${row.colorVar ?? "--accent"})`}
        ></span>
      </span>
      <span class="coverage-value">{row.display}</span>
    </li>
  {/each}
</ul>

<style>
  .coverage-bars {
    display: grid;
    gap: 0.55rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .coverage-bars li {
    display: grid;
    grid-template-columns: 1.1rem minmax(6rem, 9rem) 1fr 2.6rem;
    align-items: center;
    gap: 0.6rem;
  }

  .coverage-icon {
    display: grid;
    place-items: center;
  }

  .coverage-label {
    overflow: hidden;
    color: var(--text-muted);
    font-size: 0.78rem;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .coverage-track {
    position: relative;
    height: 0.5rem;
    overflow: hidden;
    border-radius: 999px;
    background: var(--border);
  }

  .coverage-fill {
    position: absolute;
    inset: 0;
    border-radius: inherit;
    transition: width var(--motion-quick-state);
  }

  .coverage-value {
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 0.72rem;
    text-align: right;
  }

  @media (max-width: 640px) {
    .coverage-bars li {
      grid-template-columns: 1.1rem 5rem 1fr 2.6rem;
    }
  }
</style>
