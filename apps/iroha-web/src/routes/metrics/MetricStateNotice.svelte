<script lang="ts">
  let {
    kind,
    labels = [],
    metricLabel = "",
    month = "",
    dimensionSummary = "",
  }: {
    kind: "required" | "empty";
    labels?: string[];
    metricLabel?: string;
    month?: string;
    dimensionSummary?: string;
  } = $props();
</script>

<section class="selection-state" role="status" aria-live="polite">
  {#if kind === "required"}
    <p class="eyebrow">Selection required</p>
    <h2>Choose {labels.join(" and ")}</h2>
    <p>
      This metric requires an explicit breakdown before Iroha can request or
      draw a truthful series.
    </p>
  {:else}
    <p class="eyebrow">No observations</p>
    <h2>No {metricLabel.toLowerCase()} values in this window.</h2>
    <p>
      The 12-month window ending {month} has no recorded values{dimensionSummary
        ? ` for ${dimensionSummary}`
        : ""}. The explicit selection remains unchanged.
    </p>
  {/if}
</section>

<style>
  .selection-state {
    display: grid;
    gap: 0.55rem;
  }

  .selection-state > p:last-child {
    color: var(--text-muted);
  }
</style>
