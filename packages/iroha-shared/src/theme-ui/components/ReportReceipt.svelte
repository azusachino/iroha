<script lang="ts">
  import type { ReportEvidenceRow } from "../../domain/report";
  import type { DesignLanguage } from "../../theme/themes";

  let {
    rows,
    theme,
  }: {
    rows: ReportEvidenceRow[];
    theme: DesignLanguage;
  } = $props();
</script>

<section class="report-receipt" data-theme={theme} aria-label="Exact evidence">
  <div class="receipt-edge" aria-hidden="true"></div>
  <ul>
    {#each rows as row (row.label)}
      <li>
        <span>{row.label}</span>
        <strong>{row.value}</strong>
        {#if row.detail}<small>{row.detail}</small>{/if}
      </li>
    {/each}
  </ul>
</section>

<style>
  .report-receipt {
    --receipt-paper: color-mix(in srgb, var(--surface-1) 92%, var(--accent));
    position: relative;
    overflow: hidden;
    border: 1px solid color-mix(in srgb, var(--accent) 35%, var(--border));
    border-radius: var(--radius);
    background: var(--receipt-paper);
    box-shadow: 0 0.7rem 1.8rem color-mix(in srgb, var(--bg) 70%, transparent);
  }

  .receipt-edge {
    height: 0.28rem;
    background: repeating-linear-gradient(
      90deg,
      var(--accent) 0 0.8rem,
      var(--accent-2) 0.8rem 1.6rem,
      transparent 1.6rem 2.4rem
    );
    opacity: 0.85;
  }

  ul {
    display: grid;
    gap: 0;
    margin: 0;
    padding: 0.45rem 1rem 0.65rem;
    list-style: none;
  }

  li {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 0.7rem;
    border-bottom: 1px dashed color-mix(in srgb, var(--accent) 28%, var(--border));
    padding: 0.7rem 0;
  }

  li:last-child {
    border-bottom: 0;
  }

  span,
  small {
    color: var(--text-muted);
    font-size: 0.72rem;
  }

  strong {
    color: var(--text);
    font-size: 0.8rem;
    font-variant-numeric: tabular-nums;
    text-align: right;
  }

  small {
    grid-column: 1 / -1;
    line-height: 1.35;
  }

  .report-receipt[data-theme="atlas"] {
    border-radius: 0;
    border-top-width: 3px;
  }

  .report-receipt[data-theme="atlas"] li {
    border-bottom-style: solid;
  }

  .report-receipt[data-theme="grapher"] {
    border-radius: 0.2rem;
    font-family: var(--font-mono);
  }

  .report-receipt[data-theme="grapher"] .receipt-edge {
    background: repeating-linear-gradient(
      90deg,
      var(--accent) 0 0.25rem,
      transparent 0.25rem 0.5rem
    );
  }

  .report-receipt[data-theme="field-journal"] {
    border-radius: 0;
    background:
      linear-gradient(
        90deg,
        color-mix(in srgb, var(--accent) 13%, transparent) 0 0.25rem,
        transparent 0.25rem
      ),
      repeating-linear-gradient(
        0deg,
        transparent 0 1.75rem,
        color-mix(in srgb, var(--accent) 14%, transparent) 1.75rem 1.8rem
      ),
      var(--receipt-paper);
    font-family: var(--font-serif);
  }

  .report-receipt[data-theme="phenology"] {
    border-radius: 1.2rem;
    border-color: color-mix(in srgb, var(--accent-2) 42%, var(--border));
  }

  .report-receipt[data-theme="phenology"] .receipt-edge {
    border-radius: 999px;
    margin: 0.3rem;
  }

  .report-receipt[data-theme="cadence"] {
    border-left: 3px solid var(--accent);
    border-radius: 0.25rem;
  }

  .report-receipt[data-theme="cadence"] li {
    border-bottom-style: dotted;
  }

  .report-receipt[data-theme="archive"] {
    border-radius: 0;
    border-style: double;
    border-width: 3px;
    font-family: var(--font-mono);
  }

  .report-receipt[data-theme="archive"] .receipt-edge {
    height: 0.18rem;
    background: var(--accent-2);
  }
</style>
