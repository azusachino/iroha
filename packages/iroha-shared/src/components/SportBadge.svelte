<script lang="ts">
  import { sportColor, sportLabel } from "../domain/sport";
  import { sportIcon } from "../domain/sport-icons";

  let { sport }: { sport?: string | null } = $props();

  const label = $derived(sportLabel(sport));
  const color = $derived(sportColor(sport));
  const Icon = $derived(sportIcon(sport));
</script>

<span class="sport-badge" title={label} style:--sport={color}>
  {#if Icon}<Icon class="sport-icon" size={12} aria-hidden="true" />{:else}<span
      class="sport-dot"
    ></span>{/if}
  <span class="sport-name">{label}</span>
</span>

<style>
  .sport-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    max-width: 100%;
    min-width: 0;
    padding: 0.16rem 0.55rem 0.16rem 0.45rem;
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--sport) 15%, transparent);
    border: 1px solid color-mix(in srgb, var(--sport) 32%, transparent);
    color: var(--text);
    font-size: 0.8rem;
    font-weight: 650;
    line-height: 1;
    white-space: nowrap;
  }

  .sport-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }

  .sport-dot {
    width: 0.5rem;
    height: 0.5rem;
    flex: 0 0 auto;
    border-radius: 999px;
    background: var(--sport);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--sport) 22%, transparent);
  }

  :global(.sport-icon) {
    flex: 0 0 auto;
    color: var(--sport);
  }
</style>
