<script lang="ts">
  let {
    eyebrow,
    title,
    description,
    actionHref,
    actionLabel,
  }: {
    eyebrow: string;
    title: string;
    description: string;
    actionHref?: string;
    actionLabel?: string;
  } = $props();
</script>

<header class="route-intro">
  <div class="route-intro-copy">
    <p class="eyebrow">{eyebrow}</p>
    <h1>{title}</h1>
    <p class="description">{description}</p>
  </div>
  {#if actionHref && actionLabel}
    <a class="route-action" href={actionHref}>{actionLabel}</a>
  {/if}
</header>

<style>
  .route-intro {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.75rem 0 0.5rem;
  }

  .route-intro-copy {
    min-width: 0;
  }

  .eyebrow {
    margin: 0 0 0.4rem;
    color: var(--accent);
    font-size: 0.72rem;
    font-weight: 750;
    letter-spacing: 0.11em;
    text-transform: uppercase;
  }

  h1 {
    max-width: 13ch;
    margin: 0;
    color: var(--text);
    font-size: clamp(2.25rem, 6vw, 4rem);
    letter-spacing: -0.07em;
    line-height: 0.98;
  }

  .description {
    max-width: 42rem;
    margin: 0.7rem 0 0;
    color: var(--text-muted);
    font-size: 1rem;
    line-height: 1.5;
  }

  .route-action {
    flex: 0 0 auto;
    padding: 0.55rem 0.8rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    color: var(--text);
    font-size: 0.84rem;
    white-space: nowrap;
  }

  .route-action:hover {
    border-color: var(--accent);
    text-decoration: none;
  }

  @media (max-width: 560px) {
    .route-intro {
      flex-direction: column;
    }

    .route-action {
      width: 100%;
      text-align: center;
    }
  }
</style>
