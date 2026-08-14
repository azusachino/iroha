<script lang="ts">
  import type { MonthlyReport } from "$lib/api";

  let { report }: { report: MonthlyReport } = $props();
</script>

<div class="report-coverage" aria-label="Canonical coverage">
  <span>Canonical coverage</span>
  {#each Object.entries(report.sections) as [domain, section]}
    <b class:available={section.state === "available"}>
      {domain.replace("daily_health", "health").replace("_", " ")} · {section.state}
    </b>
  {/each}
</div>

<style>
  .report-coverage {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.45rem;
  }

  .report-coverage span,
  .report-coverage b {
    font-size: 0.66rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .report-coverage span {
    margin-right: 0.25rem;
    color: var(--text-muted);
  }

  .report-coverage b {
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 0.35rem 0.55rem;
    color: var(--text-muted);
    font-weight: 650;
  }

  .report-coverage b.available {
    border-color: color-mix(in srgb, var(--accent) 55%, var(--border));
    color: var(--accent);
  }
</style>
