<script lang="ts">
  import type { Snippet } from "svelte";
  let {
    label,
    value,
    sub,
    context,
    children,
  }: {
    label: string;
    value: string;
    sub?: string;
    context?: string;
    children?: Snippet;
  } = $props();
</script>

<section class="stat-tile tile" class:has-context={!!context} aria-label={context ? label : undefined}>
  <div class="stat-label">{label}</div>
  <div class="stat-value">{value}</div>
  {#if context}<div class="stat-context">{context}</div>{/if}
  {#if sub}<div class="stat-sub">{sub}</div>{/if}
  {#if children}<div class="stat-actions">{@render children()}</div>{/if}
</section>

<style>
  .stat-tile {
    position: relative;
    overflow: hidden;
    container-type: inline-size;
    min-height: 7rem;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    gap: var(--space-3);
    padding: var(--space-4);
    border: 1px solid var(--border);
    border-radius: var(--radius, 14px);
    background:
      radial-gradient(
        120% 140% at 100% 0%,
        color-mix(in srgb, var(--accent) 12%, transparent),
        transparent 55%
      ),
      var(--tile-surface, var(--surface));
  }

  .stat-tile::before {
    content: "";
    position: absolute;
    inset: 0 0 auto;
    height: 2px;
    background: linear-gradient(90deg, var(--accent), transparent 65%);
    opacity: 0.75;
  }

  .stat-label {
    color: var(--text-muted);
    font-size: var(--type-caption);
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .stat-value {
    color: var(--text);
    font-size: var(--type-title);
    font-weight: 800;
    font-variant-numeric: tabular-nums;
    line-height: 1.05;
    letter-spacing: -0.01em;
    white-space: nowrap;
  }

  .has-context .stat-value {
    min-height: 2.2em;
    white-space: normal;
    overflow-wrap: anywhere;
  }
  .stat-context { color: var(--text-muted); font-size: var(--type-caption); line-height: 1.4; }
  .stat-actions :global(a) { display: inline-flex; align-items: center; min-height: 24px; font-size: var(--type-label); font-weight: 650; }
  .stat-sub {
    color: var(--text-muted);
    font-size: var(--type-label);
    line-height: 1.35;
  }
</style>
