<script lang="ts">
  import type { Snippet } from "svelte";
  import type { DesignLanguage } from "../theme/themes";

  let {
    title,
    eyebrow = "Period",
    ariaLabel = "Period controls",
    appearance = "grapher",
    children,
  }: {
    title?: string;
    eyebrow?: string;
    ariaLabel?: string;
    appearance?: DesignLanguage;
    children: Snippet;
  } = $props();
</script>

<section
  class="period-toolbar"
  data-appearance={appearance}
  aria-label={ariaLabel}
>
  {#if title}
    <div class="period-copy">
      <span>{eyebrow}</span>
      <strong>{title}</strong>
    </div>
  {/if}
  <div class="period-slot">
    {@render children()}
  </div>
</section>

<style>
  .period-toolbar {
    display: flex;
    min-width: 0;
    min-height: 5.25rem;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
    padding: var(--space-3) var(--space-4);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }

  .period-toolbar[data-appearance="grapher"] {
    border-radius: 2px;
  }

  .period-copy {
    display: grid;
    min-width: 0;
    gap: var(--space-1);
  }

  .period-copy span {
    color: var(--accent);
    font-size: var(--type-caption);
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .period-copy strong {
    font-size: var(--type-label);
  }

  .period-slot {
    display: flex;
    min-width: 0;
    margin-left: auto;
    justify-content: flex-end;
  }

  @media (max-width: 768px) {
    .period-toolbar {
      align-items: stretch;
      flex-direction: column;
    }

    .period-slot {
      width: 100%;
      margin-left: 0;
      justify-content: stretch;
    }

    .period-slot :global(.period-controls) {
      width: 100%;
    }
  }
</style>
