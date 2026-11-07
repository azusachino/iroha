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
  <div class="period-slot">
    {@render children()}
  </div>
</section>

<style>
  .period-toolbar {
    display: flex;
    min-width: 0;
    min-height: 3.25rem;
    align-items: center;
    justify-content: flex-end;
    gap: var(--space-3);
    padding: var(--space-2) var(--space-4);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }

  .period-toolbar[data-appearance="grapher"] {
    border-radius: 2px;
  }

  .period-slot {
    display: flex;
    min-width: 0;
    width: 100%;
    align-items: center;
    justify-content: flex-end;
  }

  @media (max-width: 768px) {
    .period-toolbar {
      align-items: stretch;
      flex-direction: column;
      padding: var(--space-3);
    }

    .period-slot {
      width: 100%;
      justify-content: stretch;
    }

    .period-slot :global(.period-controls) {
      width: 100%;
    }
  }
</style>
