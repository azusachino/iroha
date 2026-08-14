<script lang="ts">
  export type ReportFact = {
    label: string;
    value: string;
    detail?: string;
  };

  let { facts }: { facts: ReportFact[] } = $props();
</script>

<dl class="report-facts">
  {#each facts as fact (fact.label)}
    <div>
      <dt>{fact.label}</dt>
      <dd>{fact.value}</dd>
      {#if fact.detail}<small>{fact.detail}</small>{/if}
    </div>
  {/each}
</dl>

<style>
  .report-facts {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 0.65rem;
    margin: 0;
  }

  .report-facts div {
    display: grid;
    align-content: start;
    gap: 0.2rem;
    min-width: 0;
  }

  dt,
  small {
    color: var(--text-muted);
    font-size: 0.7rem;
  }

  dd {
    margin: 0;
    color: var(--text);
    font-size: clamp(1.05rem, 3vw, 1.5rem);
    font-weight: 750;
    line-height: 1.05;
  }

  small {
    line-height: 1.3;
  }

  @media (max-width: 640px) {
    .report-facts {
      grid-template-columns: 1fr 1fr;
    }
  }
</style>
