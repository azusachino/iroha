<script lang="ts">
  import type { PanelRow } from "./metric-panel";

  // A DOM/CSS bar list, not an ECharts BarChart, for two reasons: a row
  // with a real reference range needs per-row scaling (its own min-max, not
  // one shared axis across metrics with different units), and real DOM
  // means a row's icon (Svelte component) can actually render here --
  // ECharts draws to canvas, so a component can't sit inside its axis
  // labels. MetricTable remains the accessible/exact-values source (this
  // is presentational, matching BarChart's own aria: {enabled:false}).
  let { rows }: { rows: PanelRow[] } = $props();

  // Rows WITHOUT a range have no common unit to share a scale against (bpm
  // vs. steps vs. km) -- each such row falls back to its own single-row
  // relative bar (full width at its own value) rather than a shared list
  // max, which would silently compare unrelated units.
  function barPct(row: PanelRow): number {
    if (row.value == null) return 0;
    if (row.range) {
      const { min, max } = row.range;
      return Math.min(100, Math.max(0, ((row.value - min) / (max - min)) * 100));
    }
    return 100;
  }

  function isOutOfRange(row: PanelRow): boolean {
    return (
      row.range != null &&
      row.value != null &&
      (row.value < row.range.min || row.value > row.range.max)
    );
  }

  function barColorVar(row: PanelRow): string {
    if (isOutOfRange(row)) return "--danger";
    return row.colorVar ?? "--accent";
  }

  function hoverTitle(row: PanelRow): string {
    const parts = [
      row.breakdown
        ? `${row.label}: ${row.display} · ${row.breakdown}`
        : `${row.label}: ${row.display}`,
    ];
    if (row.range) {
      parts.push(`Healthy range: ${row.range.min}–${row.range.max} · ${row.range.source}`);
      if (row.range.caveat) parts.push(row.range.caveat);
    }
    return parts.join(" · ");
  }
</script>

<ul class="coverage-bars" aria-hidden="true">
  {#each rows as row}
    <li title={hoverTitle(row)}>
      <span class="coverage-icon" style:color={`var(${row.colorVar ?? "--accent"})`}>
        {#if row.icon}<row.icon size={15} />{/if}
      </span>
      <span class="coverage-label">{row.label}</span>
      <span class="coverage-track" class:ranged={!!row.range}>
        <span
          class="coverage-fill"
          style:width={`${barPct(row)}%`}
          style:background={`var(${barColorVar(row)})`}
        ></span>
      </span>
      <span class="coverage-value" class:danger={isOutOfRange(row)}>{row.display}</span>
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

  /* A ranged track has real min/max bounds (see healthMetricRange) -- the
     inset ring marks it as a gauge, distinct from the plain relative bars
     used for metrics with no established healthy range. */
  .coverage-track.ranged {
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--text) 12%, transparent);
  }

  .coverage-fill {
    position: absolute;
    inset: 0;
    border-radius: inherit;
    transition:
      width var(--motion-quick-state),
      background var(--motion-quick-state);
  }

  .coverage-value {
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 0.72rem;
    text-align: right;
  }

  .coverage-value.danger {
    color: var(--danger);
    font-weight: 600;
  }

  @media (max-width: 640px) {
    .coverage-bars li {
      grid-template-columns: 1.1rem 5rem 1fr 2.6rem;
    }
  }
</style>
