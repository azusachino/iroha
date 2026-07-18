<script lang="ts">
  import type { Snippet } from "svelte";
  import { page } from "$app/state";

  let { children }: { children: Snippet } = $props();
  const links = [
    ["/", "Daily view"],
    ["/daily", "Patterns"],
    ["/activities", "Activity data"],
    ["/sleep", "Sleep data"],
    ["/media", "Media data"],
    ["/share", "Public view"],
  ];
</script>

<div class="grapher-site">
  <header class="grapher-bar">
    <a class="grapher-brand" href="/">
      <span class="grapher-brand-mark" aria-hidden="true">I</span>
      <span>
        <strong>iroha</strong>
        <small>personal data explorer</small>
      </span>
    </a>
    <nav aria-label="Data explorer navigation">
      {#each links as [href, label]}
        <a
          class:active={page.url.pathname === href ||
            (href !== "/" && page.url.pathname.startsWith(href))}
          {href}>{label}</a
        >
      {/each}
    </nav>
    <div class="grapher-status">PRIVATE DATASET · LIVE</div>
  </header>
  <main class="grapher-content">{@render children()}</main>
  <footer class="grapher-footer">
    <span>iroha grapher</span>
    <span>Every value remains traceable to imported evidence.</span>
  </footer>
</div>

<style>
  .grapher-site {
    min-height: 100vh;
    background: var(--bg);
    color: var(--text);
  }

  .grapher-bar {
    position: sticky;
    top: 0;
    z-index: 20;
    display: grid;
    grid-template-columns: minmax(12rem, 1fr) auto minmax(12rem, 1fr);
    align-items: center;
    gap: 1.5rem;
    min-height: 4.5rem;
    padding: 0.75rem 2rem;
    border-bottom: 1px solid var(--border);
    background: color-mix(in srgb, var(--bg) 94%, transparent);
    backdrop-filter: blur(12px);
  }

  .grapher-brand {
    display: inline-flex;
    align-items: center;
    gap: 0.7rem;
    color: var(--text);
    text-decoration: none;
  }

  .grapher-brand-mark {
    display: grid;
    width: 2.25rem;
    height: 2.25rem;
    place-items: center;
    border: 1px solid var(--accent);
    color: var(--accent);
    font-family: Georgia, serif;
    font-size: 1.35rem;
  }

  .grapher-brand strong,
  .grapher-brand small {
    display: block;
  }

  .grapher-brand strong {
    font-size: 1.05rem;
    letter-spacing: -0.04em;
  }

  .grapher-brand small {
    margin-top: 0.1rem;
    color: var(--text-muted);
    font-size: 0.62rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .grapher-bar nav {
    display: flex;
    align-items: center;
    gap: 0.15rem;
  }

  .grapher-bar nav a {
    padding: 0.45rem 0.6rem;
    color: var(--text-muted);
    font-size: 0.72rem;
    text-decoration: none;
    white-space: nowrap;
  }

  .grapher-bar nav a:hover,
  .grapher-bar nav a.active {
    background: var(--surface-2);
    color: var(--text);
  }

  .grapher-bar nav a.active {
    box-shadow: inset 0 -2px var(--accent);
  }

  .grapher-status {
    justify-self: end;
    color: var(--text-muted);
    font-size: 0.6rem;
    letter-spacing: 0.1em;
  }

  .grapher-content {
    width: min(1180px, calc(100% - 3rem));
    margin: 0 auto;
    padding: 3rem 0 5rem;
  }

  .grapher-footer {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    padding: 1rem 2rem;
    border-top: 1px solid var(--border);
    color: var(--text-muted);
    font-size: 0.7rem;
  }

  @media (max-width: 900px) {
    .grapher-bar {
      grid-template-columns: 1fr auto;
    }

    .grapher-bar nav {
      grid-column: 1 / -1;
      order: 3;
      overflow-x: auto;
    }

    .grapher-status {
      grid-column: 2;
      grid-row: 1;
    }
  }

  @media (max-width: 520px) {
    .grapher-bar {
      padding: 0.75rem 1rem;
    }

    .grapher-content {
      width: min(100% - 2rem, 1180px);
      padding-top: 2rem;
    }

    .grapher-footer {
      flex-direction: column;
      padding: 1rem;
    }
  }
</style>
