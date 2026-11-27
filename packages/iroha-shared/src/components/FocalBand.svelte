<script lang="ts">
  import type { Snippet } from "svelte";
  import EmptyNote from "./EmptyNote.svelte";

  export interface FocalPrimaryMetric {
    label: string;
    value: string;
    unit?: string;
    detail?: string;
  }

  export interface FocalSecondaryItem {
    label: string;
    value: string;
    detail?: string;
    href?: string;
    linkLabel?: string;
  }

  let {
    heading,
    controls,
    primary,
    secondary,
    empty,
  }: {
    heading?: string;
    controls?: Snippet;
    primary: Snippet | FocalPrimaryMetric;
    secondary: FocalSecondaryItem[] | Snippet;
    empty?: string | Snippet;
  } = $props();

  const headingId = $props.id();
  const sectionLabel = $derived(
    heading
      ? undefined
      : typeof primary === "function"
        ? undefined
        : primary.label,
  );
</script>

<section
  class="focal-band"
  aria-labelledby={heading ? headingId : undefined}
  aria-label={sectionLabel}
>
  {#if heading || controls}
    <div class="focal-header">
      {#if heading}
        <h2 id={headingId} class="focal-heading">{heading}</h2>
      {/if}
      {#if controls}
        <div class="focal-controls">{@render controls()}</div>
      {/if}
    </div>
  {/if}
  <div class="focal-grid">
    <div class="focal-primary">
      {#if typeof primary === "function"}
        {@render primary()}
      {:else}
        <div class="focal-primary-label">{primary.label}</div>
        <div class="focal-primary-metric">
          <span class="focal-primary-value">{primary.value}</span>
          {#if primary.unit}
            <span class="focal-primary-unit">{primary.unit}</span>
          {/if}
        </div>
        {#if primary.detail}
          <p class="focal-primary-detail">{primary.detail}</p>
        {/if}
      {/if}
    </div>

    <div class="focal-secondary">
      {#if typeof secondary === "function"}
        {@render secondary()}
      {:else}
        <dl class="focal-secondary-list">
          {#each secondary as item}
            <div class="focal-secondary-item">
              <dt>{item.label}</dt>
              <dd>
                <span>{item.value}</span>
                {#if item.detail}
                  <span class="focal-secondary-detail">{item.detail}</span>
                {/if}
                {#if item.href}
                  <a
                    class="focal-secondary-link"
                    href={item.href}
                    aria-label={item.linkLabel ?? item.label}
                  >
                    {item.linkLabel ?? item.label}
                  </a>
                {/if}
              </dd>
            </div>
          {/each}
        </dl>
      {/if}
    </div>
  </div>

  {#if empty}
    <div class="focal-empty">
      {#if typeof empty === "string"}
        <EmptyNote text={empty} />
      {:else}
        {@render empty()}
      {/if}
    </div>
  {/if}
</section>

<style>
  .focal-band {
    display: grid;
    gap: var(--space-3);
    margin-bottom: var(--gap-section);
  }

  .focal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
  }

  .focal-heading {
    margin: 0;
    color: var(--text-muted);
    font-size: var(--type-title-small);
    font-weight: 700;
    line-height: 1.2;
  }

  .focal-controls {
    margin-left: auto;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
  }

  @media (max-width: 640px) {
    .focal-header {
      flex-direction: column;
      align-items: stretch;
    }

    .focal-controls {
      margin-left: 0;
    }
  }

  .focal-grid {
    display: grid;
    grid-template-columns: minmax(0, 2fr) minmax(0, 1fr);
    gap: var(--gap-grid);
    align-items: stretch;
  }

  @media (max-width: 1024px) {
    .focal-grid {
      grid-template-columns: minmax(0, 1fr);
    }
  }

  .focal-primary,
  .focal-secondary {
    min-width: 0;
    padding: var(--pad-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    background: var(--tile-surface, var(--surface));
    box-shadow: var(--tile-shadow), var(--tile-rim);
  }

  .focal-primary {
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    gap: var(--space-3);
  }

  .focal-primary-label {
    color: var(--text-muted);
    font-size: var(--type-label);
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .focal-primary-metric {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: var(--space-2);
  }

  .focal-primary-value {
    color: var(--text);
    font-size: var(--type-hero);
    font-weight: 800;
    font-variant-numeric: tabular-nums;
    line-height: 1.05;
    letter-spacing: -0.02em;
  }

  .focal-primary-unit {
    color: var(--text-muted);
    font-size: var(--type-title-small);
    font-weight: 600;
  }

  .focal-primary-detail {
    margin: 0;
    color: var(--text-muted);
    font-size: var(--type-label);
    line-height: 1.4;
  }

  .focal-secondary {
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  .focal-secondary-list {
    display: grid;
    gap: var(--space-3);
    margin: 0;
  }

  .focal-secondary-item {
    display: grid;
    gap: var(--space-1);
  }

  .focal-secondary-item dt {
    color: var(--text-muted);
    font-size: var(--type-caption);
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }

  .focal-secondary-item dd {
    margin: 0;
    color: var(--text);
    font-size: var(--type-title-small);
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: var(--space-2);
  }

  .focal-secondary-detail {
    color: var(--text-muted);
    font-size: var(--type-caption);
    font-weight: 500;
  }

  .focal-secondary-link {
    display: inline-flex;
    align-items: center;
    min-height: var(--control-target-min);
    color: var(--accent);
    font-size: var(--type-caption);
    font-weight: 600;
    text-decoration: none;
  }

  .focal-secondary-link:hover {
    text-decoration: underline;
  }
</style>
