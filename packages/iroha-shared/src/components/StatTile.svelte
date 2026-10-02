<script lang="ts">
  let {
    label,
    value,
    sub,
  }: {
    label: string;
    value: string;
    sub?: string;
  } = $props();
</script>

<section class="stat-tile tile">
  <div class="stat-label">{label}</div>
  <div class="stat-value">{value}</div>
  {#if sub}<div class="stat-sub">{sub}</div>{/if}
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

  .stat-sub {
    color: var(--text-muted);
    font-size: var(--type-label);
    line-height: 1.35;
  }
</style>
