<script lang="ts">
  import type { Snippet } from "svelte";
  import { useTheme } from "$lib/themes/context.svelte";

  let {
    title,
    eyebrow = "Period",
    ariaLabel = "Period controls",
    children,
  }: {
    title?: string;
    eyebrow?: string;
    ariaLabel?: string;
    children: Snippet;
  } = $props();

  const theme = useTheme();
</script>

<section
  class="period-toolbar"
  data-theme={theme.definition().identity.id}
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
    min-height: 5.25rem;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.85rem 1rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }

  .period-toolbar[data-theme="atlas"] {
    border-width: 2px;
    background-image:
      linear-gradient(
        color-mix(in srgb, var(--accent) 7%, transparent) 1px,
        transparent 1px
      ),
      linear-gradient(
        90deg,
        color-mix(in srgb, var(--accent) 7%, transparent) 1px,
        transparent 1px
      );
    background-size: 12px 12px;
  }

  .period-toolbar[data-theme="field-journal"] {
    border-style: dashed;
  }

  .period-toolbar[data-theme="phenology"] {
    border-radius: 1.2rem;
  }

  .period-toolbar[data-theme="sound-map"] {
    border-inline-width: 3px;
  }

  .period-toolbar[data-theme="archive"] {
    border-width: 3px;
    border-style: double;
    border-radius: 0;
  }

  .period-toolbar[data-theme="grapher"] {
    border-radius: 2px;
    border-bottom-width: 3px;
  }

  .period-copy {
    display: grid;
    min-width: 0;
    gap: 0.2rem;
  }

  .period-copy span {
    color: var(--accent);
    font-size: 0.66rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .period-copy strong {
    font-size: 0.9rem;
  }

  .period-slot {
    display: flex;
    min-width: 0;
    margin-left: auto;
    justify-content: flex-end;
  }

  @media (max-width: 760px) {
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
