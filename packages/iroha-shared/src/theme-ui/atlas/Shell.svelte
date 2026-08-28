<script lang="ts">
  import type { ShellThemeProps } from "../../view-contracts/shell-view";

  let { children, theme, brand, nav, actions }: ShellThemeProps = $props();
</script>

<div class="atlas-site" data-theme={theme}>
  <header class="appbar atlas-header">
    <div class="atlas-header-brand">
      <span class="atlas-compass atlas-compass-header" aria-hidden="true">
        <svg viewBox="0 0 32 32" role="img" aria-hidden="true">
          <circle cx="16" cy="16" r="14" />
          <path d="M16 4 L19 16 L16 28 L13 16 Z" />
          <text x="16" y="9" text-anchor="middle">N</text>
        </svg>
      </span>
      {@render brand()}
    </div>
    {@render nav()}
    <div class="appbar-actions atlas-header-legend">
      {@render actions()}
    </div>
  </header>
  <div class="atlas-content">{@render children()}</div>
  <footer class="atlas-footer">
    <span class="atlas-compass" aria-hidden="true">
      <svg viewBox="0 0 32 32" role="img" aria-hidden="true">
        <circle cx="16" cy="16" r="14" />
        <path d="M16 4 L19 16 L16 28 L13 16 Z" />
        <text x="16" y="9" text-anchor="middle">N</text>
      </svg>
    </span>
    <span>iroha atlas · places, routes, distance</span>
    <span>every plate stays traceable to imported evidence</span>
  </footer>
</div>

<style>
  .atlas-site {
    --shell-width: 1160px;
    position: relative;
    min-height: 100vh;
    background:
      repeating-linear-gradient(
        0deg,
        color-mix(in srgb, var(--accent) 6%, transparent) 0 1px,
        transparent 1px 5rem
      ),
      repeating-linear-gradient(
        90deg,
        color-mix(in srgb, var(--accent) 6%, transparent) 0 1px,
        transparent 1px 5rem
      ),
      var(--bg);
    color: var(--text);
  }

  .atlas-content {
    width: min(var(--shell-width), calc(100% - 3rem));
    margin: 0 auto;
    padding: 3rem 0 4rem;
  }

  .atlas-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding: 1rem max(1rem, calc((100% - var(--shell-width)) / 2));
    background: color-mix(in srgb, var(--bg) 88%, transparent);
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.65rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }

  .atlas-compass svg {
    display: block;
    width: 1.4rem;
    height: 1.4rem;
  }

  .atlas-compass circle {
    fill: none;
    stroke: var(--text-muted);
    stroke-width: 1;
  }

  .atlas-compass path {
    fill: var(--accent);
  }

  .atlas-compass text {
    fill: var(--text-muted);
    font-size: 8px;
    font-family: var(--font-mono);
  }

  .atlas-compass-header {
    margin-right: 0.4rem;
  }

  /* Everything below restyles the shared appbar/nav/actions markup into a
     surveyor's map legend; the links/buttons/select themselves are
     untouched, so focus order, tap targets, and accessible names stay
     whatever the host defines. */
  .atlas-header {
    background:
      repeating-linear-gradient(
        90deg,
        color-mix(in srgb, var(--accent) 8%, transparent) 0 1px,
        transparent 1px 2.5rem
      ),
      color-mix(in srgb, var(--bg) 92%, transparent);
    border-bottom: 1px dashed
      color-mix(in srgb, var(--accent) 35%, var(--border));
    box-shadow: none;
  }

  .atlas-header-brand {
    display: inline-flex;
    align-items: center;
    min-width: max-content;
  }

  .atlas-header :global(.brand) {
    font-family: var(--font-mono);
    letter-spacing: 0.04em;
  }

  .atlas-header :global(.main-nav) {
    background: color-mix(in srgb, var(--bg) 78%, transparent);
    border: 1px dashed color-mix(in srgb, var(--accent) 35%, var(--border));
  }

  .atlas-header :global(.main-nav > a),
  .atlas-header :global(.navigation-menu > summary) {
    font-family: var(--font-mono);
    letter-spacing: 0.02em;
  }

  .atlas-header :global(.main-nav > a:hover),
  .atlas-header :global(.main-nav > a.active),
  .atlas-header :global(.navigation-menu.active > summary) {
    background: transparent;
    color: var(--accent);
    box-shadow: inset 0 0 0 1px var(--accent);
  }

  .atlas-header-legend {
    padding: 0.2rem 0.5rem;
    border: 1px dashed color-mix(in srgb, var(--accent) 35%, var(--border));
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--bg) 78%, transparent);
  }

  @media (max-width: 640px) {
    .atlas-content {
      width: min(100% - 2rem, var(--shell-width));
      padding-top: 2rem;
    }

    .atlas-footer {
      flex-direction: column;
      align-items: flex-start;
      gap: 0.4rem;
      padding: 1rem;
    }

    .atlas-compass-header {
      display: none;
    }
  }
</style>
