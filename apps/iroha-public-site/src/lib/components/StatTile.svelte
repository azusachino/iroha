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
  {#if sub}
    <div class="stat-sub">{sub}</div>
  {/if}
</section>

<style>
  .stat-tile {
    position: relative;
    overflow: hidden;
    /* Size the value against the tile's own width (cqi), so a long number
		   never overruns a narrow column regardless of viewport. */
    container-type: inline-size;
    min-height: 7rem;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    gap: 0.6rem;
    /* Faint accent wash lifts the KPI off the flat surface. */
    background:
      radial-gradient(
        120% 140% at 100% 0%,
        color-mix(in srgb, var(--accent) 12%, transparent),
        transparent 55%
      ),
      var(--tile-surface);
  }

  /* Accent hairline along the top edge for a touch of definition. */
  .stat-tile::before {
    content: "";
    position: absolute;
    inset: 0 0 auto 0;
    height: 2px;
    background: linear-gradient(90deg, var(--accent), transparent 65%);
    opacity: 0.75;
  }

  .stat-label {
    color: var(--text-muted);
    font-size: 0.72rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .stat-value {
    color: var(--text);
    font-size: clamp(1.2rem, 13cqi, 1.9rem);
    font-weight: 800;
    line-height: 1.05;
    letter-spacing: -0.01em;
    white-space: nowrap;
  }

  .stat-sub {
    color: var(--text-muted);
    font-size: 0.84rem;
    line-height: 1.35;
  }
</style>
