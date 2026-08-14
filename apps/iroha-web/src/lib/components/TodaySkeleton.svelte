<script lang="ts">
  let { label }: { label: string } = $props();
</script>

<section class="today-skeleton" aria-busy="true" aria-label={label}>
  <div class="skeleton-status" role="status" aria-live="polite">{label}</div>

  <div class="skeleton-hero" aria-hidden="true">
    <div class="skeleton-copy">
      <span class="skeleton-line eyebrow-line"></span>
      <span class="skeleton-line title-line"></span>
      <span class="skeleton-line copy-line"></span>
      <span class="skeleton-line note-line"></span>
    </div>
    <span class="skeleton-orbit"></span>
  </div>

  <div class="skeleton-kpis" aria-hidden="true">
    {#each Array(4) as _}
      <div class="skeleton-kpi">
        <span class="skeleton-line label-line"></span>
        <span class="skeleton-line value-line"></span>
        <span class="skeleton-bar"></span>
      </div>
    {/each}
  </div>

  <div class="skeleton-grid" aria-hidden="true">
    <div class="skeleton-card tall-card">
      <span></span><span></span><span></span>
    </div>
    <div class="skeleton-card"><span></span><span></span><span></span></div>
    <div class="skeleton-card"><span></span><span></span><span></span></div>
    <div class="skeleton-card wide-card">
      <span></span><span></span><span></span>
    </div>
    <div class="skeleton-card wide-card">
      <span></span><span></span><span></span>
    </div>
  </div>
</section>

<style>
  .today-skeleton {
    display: grid;
    position: relative;
    gap: 1rem;
    min-width: 0;
  }

  .skeleton-status {
    position: absolute;
    inset: 0.75rem 0.9rem auto auto;
    z-index: 1;
    padding: 0.35rem 0.6rem;
    border: 1px solid color-mix(in srgb, var(--accent) 38%, var(--border));
    border-radius: 999px;
    background: color-mix(in srgb, var(--surface) 88%, transparent);
    color: var(--accent);
    font-size: 0.7rem;
    font-weight: 650;
  }

  .skeleton-hero,
  .skeleton-kpi,
  .skeleton-card {
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface) 94%, var(--accent));
  }

  .skeleton-hero {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1.5rem;
    min-height: 22rem;
    padding: 2.5rem;
    overflow: hidden;
    border-radius: var(--radius);
  }

  .skeleton-copy {
    display: grid;
    align-content: end;
    gap: 0.8rem;
    width: min(34rem, 65%);
  }

  .skeleton-line,
  .skeleton-card span,
  .skeleton-bar {
    display: block;
    border-radius: 999px;
    background: color-mix(in srgb, var(--text-muted) 17%, var(--surface-2));
    animation: skeleton-wave 1.4s ease-in-out infinite;
  }

  .eyebrow-line {
    width: 9rem;
    height: 0.55rem;
  }

  .title-line {
    width: min(27rem, 90%);
    height: clamp(3.2rem, 8vw, 6rem);
    border-radius: 0.35rem;
  }

  .copy-line {
    width: 80%;
    height: 0.8rem;
  }

  .note-line {
    width: 10rem;
    height: 0.55rem;
  }

  .skeleton-orbit {
    display: block;
    flex: 0 0 13rem;
    width: 13rem;
    height: 13rem;
    border: 1px solid color-mix(in srgb, var(--accent) 38%, var(--border));
    border-radius: 50%;
    background: radial-gradient(
      circle,
      color-mix(in srgb, var(--accent) 14%, transparent),
      transparent 65%
    );
    animation: skeleton-pulse 1.8s ease-in-out infinite;
  }

  .skeleton-kpis {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 0.75rem;
  }

  .skeleton-kpi {
    display: grid;
    gap: 0.6rem;
    min-height: 6.1rem;
    padding: 0.9rem;
    border-radius: calc(var(--radius) - 4px);
  }

  .label-line {
    width: 45%;
    height: 0.55rem;
  }

  .value-line {
    width: 65%;
    height: 1.5rem;
    border-radius: 0.3rem;
  }

  .skeleton-bar {
    width: 100%;
    height: 0.22rem;
  }

  .skeleton-grid {
    display: grid;
    grid-template-columns: repeat(12, minmax(0, 1fr));
    gap: 1rem;
  }

  .skeleton-card {
    display: grid;
    align-content: start;
    gap: 0.85rem;
    min-height: 11.5rem;
    padding: 1.1rem;
    border-radius: calc(var(--radius) + 2px);
  }

  .skeleton-card span:nth-child(1) {
    width: 32%;
    height: 0.65rem;
  }

  .skeleton-card span:nth-child(2) {
    width: 84%;
    height: 1.25rem;
    border-radius: 0.3rem;
  }

  .skeleton-card span:nth-child(3) {
    width: 62%;
    height: 0.7rem;
  }

  .tall-card {
    grid-row: span 2;
    min-height: 24rem;
  }

  .tall-card,
  .skeleton-card:nth-child(2),
  .skeleton-card:nth-child(3) {
    grid-column: span 5;
  }

  .skeleton-card:nth-child(2),
  .skeleton-card:nth-child(3) {
    grid-column: span 7;
  }

  .wide-card {
    grid-column: span 7;
    min-height: 16rem;
  }

  @keyframes skeleton-wave {
    0%,
    100% {
      opacity: 0.48;
    }
    50% {
      opacity: 0.9;
    }
  }

  @keyframes skeleton-pulse {
    0%,
    100% {
      opacity: 0.55;
      transform: scale(0.98);
    }
    50% {
      opacity: 0.95;
      transform: scale(1);
    }
  }

  @media (max-width: 1024px) {
    .skeleton-hero {
      align-items: start;
      flex-direction: column;
      min-height: 25rem;
      padding: 1.5rem;
    }

    .skeleton-copy {
      width: 100%;
    }

    .skeleton-orbit {
      align-self: center;
    }

    .skeleton-kpis {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .skeleton-grid {
      grid-template-columns: 1fr;
    }

    .tall-card,
    .skeleton-card:nth-child(2),
    .skeleton-card:nth-child(3),
    .wide-card {
      grid-column: auto;
      grid-row: auto;
    }
  }

  @media (max-width: 640px) {
    .skeleton-hero {
      min-height: 23rem;
    }

    .skeleton-kpis {
      gap: 0.5rem;
    }

    .skeleton-kpi {
      min-height: 5.6rem;
      padding: 0.7rem;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .skeleton-line,
    .skeleton-card span,
    .skeleton-bar,
    .skeleton-orbit {
      animation: none;
    }
  }
</style>
