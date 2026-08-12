<script lang="ts">
  import { pointValue, type SharedMetricSeries } from "./metric-series";

  let {
    series,
    unit,
  }: {
    series: SharedMetricSeries[];
    unit: string;
  } = $props();

  const dimensionLabel = (dimensions: Record<string, string>) =>
    Object.entries(dimensions)
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([key, value]) => `${key}: ${value}`)
      .join(", ") || "—";
</script>

<div class="metric-table-wrap">
  <table>
    <caption class="sr-only">Metric values</caption>
    <thead><tr><th>Period</th><th>Breakdown</th><th>Value ({unit})</th><th>Observed days</th></tr></thead>
    <tbody>
      {#each series as item}
        {#each item.points as point}
          <tr>
            <th scope="row">{point.period}</th>
            <td>{dimensionLabel(item.dimensions)}</td>
            <td>{pointValue(point) ?? "—"}</td>
            <td>{point.observed_days}</td>
          </tr>
        {/each}
      {/each}
    </tbody>
  </table>
</div>

<style>
  .metric-table-wrap { overflow-x: auto; }
  table { width: 100%; border-collapse: collapse; color: var(--text, #1b2430); font-size: 0.78rem; }
  th, td { padding: 0.55rem 0.65rem; border-bottom: 1px solid var(--border, #d6dbe3); text-align: left; }
  thead th { color: var(--text-muted, #778090); font-size: 0.68rem; letter-spacing: 0.06em; text-transform: uppercase; }
  tbody th { font-weight: 650; }
  .sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }
</style>
