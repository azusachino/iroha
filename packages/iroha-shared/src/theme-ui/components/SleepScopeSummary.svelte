<script lang="ts">
  import type { DesignLanguage } from "../../theme/themes";
  import type { SleepAggregateBucket } from "../../domain/sleep";
  import { formatCanonicalMonth } from "../../format/format";

  let {
    summary = null,
    scope = "",
    theme,
  }: {
    summary?: SleepAggregateBucket | null;
    scope?: string;
    theme: DesignLanguage;
  } = $props();

  function scopeLabel(value: string): string {
    if (/^\d{4}-\d{2}$/.test(value)) return formatCanonicalMonth(value);
    return value || "Lifetime";
  }
</script>

<div
  class="sleep-scope-summary"
  data-theme={theme}
  aria-label={`Sleep totals for ${scopeLabel(scope)}`}
>
  <span class="scope-label">
    {scopeLabel(scope)}
    {#if summary}
      · {summary.observed_wake_dates.toLocaleString()}
      {summary.observed_wake_dates === 1 ? "wake date" : "wake dates"}
    {/if}
  </span>
  <div class="kind-legend" aria-label="Sleep session types">
    <span><i class="main-dot"></i>Main sleep</span>
    <span><i class="nap-dot"></i>Nap</span>
  </div>
</div>

<style>
  .sleep-scope-summary {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2) var(--space-4);
    color: var(--text-muted);
    font-size: var(--type-caption);
  }

  .scope-label {
    font-weight: 600;
  }

  .kind-legend {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-3);
  }

  .kind-legend span {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
  }

  .kind-legend i {
    width: 0.55rem;
    height: 0.55rem;
    border-radius: 50%;
  }

  .main-dot {
    background: var(--accent);
  }

  .nap-dot {
    background: var(--accent-2);
  }
</style>
