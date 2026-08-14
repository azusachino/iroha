<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    label,
    title,
    summary,
    tone = "default",
    children,
  }: {
    label: string;
    title: string;
    summary?: string;
    tone?: "default" | "feature" | "quiet";
    children: Snippet;
  } = $props();
</script>

<article
  class:feature={tone === "feature"}
  class:quiet={tone === "quiet"}
  class="report-metric-card"
  aria-label={title}
>
  <header>
    <div>
      <p>{label}</p>
      <h3>{title}</h3>
    </div>
    {#if summary}<span>{summary}</span>{/if}
  </header>
  <div class="metric-body">
    {@render children()}
  </div>
</article>

<style>
  .report-metric-card {
    display: grid;
    min-width: 0;
    gap: 0.85rem;
    border: 1px solid var(--border);
    padding: 1rem;
    background: var(--surface);
  }

  .report-metric-card.feature {
    border-color: color-mix(in srgb, var(--accent) 60%, var(--border));
    box-shadow: var(--tile-shadow);
  }

  .report-metric-card.quiet {
    background: color-mix(in srgb, var(--surface) 88%, var(--surface-2));
  }

  header {
    display: flex;
    align-items: start;
    justify-content: space-between;
    gap: 1rem;
  }

  p,
  h3 {
    margin: 0;
  }

  header p {
    color: var(--accent);
    font-size: 0.66rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  h3 {
    margin-top: 0.2rem;
    font-size: 1.05rem;
    letter-spacing: -0.03em;
  }

  header > span {
    color: var(--text-muted);
    font-size: 0.72rem;
    text-align: right;
  }

  .metric-body {
    min-width: 0;
  }

  @media (max-width: 640px) {
    header {
      display: grid;
    }

    header > span {
      text-align: left;
    }
  }
</style>
