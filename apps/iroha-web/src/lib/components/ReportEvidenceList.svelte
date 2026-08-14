<script lang="ts">
  export type ReportEvidenceRow = {
    label: string;
    value: string;
    detail?: string;
  };

  let { rows }: { rows: ReportEvidenceRow[] } = $props();
</script>

<ul class="report-evidence">
  {#each rows as row (row.label)}
    <li>
      <span>{row.label}</span>
      <strong>{row.value}</strong>
      {#if row.detail}<small>{row.detail}</small>{/if}
    </li>
  {/each}
</ul>

<style>
  .report-evidence {
    display: grid;
    gap: 0.4rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  li {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 0.7rem;
    border-bottom: 1px solid var(--border);
    padding: 0.45rem 0;
  }

  span,
  small {
    color: var(--text-muted);
    font-size: 0.76rem;
  }

  strong {
    font-size: 0.78rem;
    text-align: right;
  }

  small {
    grid-column: 1 / -1;
    line-height: 1.35;
  }
</style>
