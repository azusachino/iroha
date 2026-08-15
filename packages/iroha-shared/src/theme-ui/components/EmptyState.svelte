<script lang="ts">
  let {
    eyebrow = "No records",
    title,
    description,
    actionHref,
    actionLabel,
  }: {
    eyebrow?: string;
    title: string;
    description: string;
    actionHref?: string;
    actionLabel?: string;
  } = $props();
</script>

<section class="empty-state tile" role="status" aria-live="polite">
  <div class="empty-mark" aria-hidden="true">—</div>
  <div class="empty-copy">
    <p class="empty-eyebrow">{eyebrow}</p>
    <h2>{title}</h2>
    <p>{description}</p>
    {#if actionHref && actionLabel}
      <a href={actionHref}>{actionLabel} →</a>
    {/if}
  </div>
</section>

<style>
  .empty-state {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr);
    align-items: center;
    gap: 1.25rem;
    min-height: 16rem;
    padding: clamp(1.25rem, 4vw, 2.5rem);
    border: 1px solid color-mix(in srgb, var(--accent) 28%, var(--border));
    background:
      radial-gradient(
        circle at 18% 50%,
        color-mix(in srgb, var(--accent) 13%, transparent),
        transparent 12rem
      ),
      var(--surface);
  }

  .empty-mark {
    display: grid;
    width: 4rem;
    height: 4rem;
    place-items: center;
    border: 1px solid color-mix(in srgb, var(--accent) 45%, var(--border));
    border-radius: 50%;
    color: var(--accent);
    font-size: 2rem;
    line-height: 1;
  }

  .empty-copy {
    min-width: 0;
  }

  .empty-eyebrow {
    margin: 0 0 0.45rem;
    color: var(--accent);
    font-size: 0.7rem;
    font-weight: 750;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }

  h2 {
    margin: 0;
    font-size: clamp(1.35rem, 3vw, 2.2rem);
    letter-spacing: -0.06em;
  }

  .empty-copy > p:not(.empty-eyebrow) {
    max-width: 42rem;
    margin: 0.6rem 0 0;
    color: var(--text-muted);
    line-height: 1.55;
  }

  a {
    display: inline-block;
    margin-top: 1rem;
    color: var(--accent);
    font-size: 0.82rem;
    font-weight: 650;
  }

  @media (max-width: 640px) {
    .empty-state {
      grid-template-columns: 1fr;
    }
  }
</style>
