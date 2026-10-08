<script lang="ts">
  import type { ShellThemeProps } from "../../view-contracts/shell-view";

  let { children, theme, brand, nav, actions }: ShellThemeProps = $props();
</script>

<div class="grapher-site" data-theme={theme}>
  <header class="appbar grapher-header">
    {@render brand()}
    <p class="rail-label">Private workspace</p>
    {@render nav()}
    <div class="appbar-actions">
      {@render actions()}
    </div>
  </header>
  <div class="grapher-content">{@render children()}</div>
  <footer class="grapher-footer">
    <span>iroha / personal data archive</span>
    <span>Every value remains traceable to imported evidence.</span>
  </footer>
</div>

<style>
  .grapher-site {
    display: grid;
    grid-template-columns: 15rem minmax(0, 1fr);
    grid-template-rows: minmax(100vh, 1fr) auto;
    min-height: 100vh;
    background: var(--bg);
    color: var(--text);
  }

  .grapher-header {
    position: sticky;
    top: 0;
    grid-column: 1;
    grid-row: 1 / span 2;
    display: flex;
    flex-direction: column;
    align-items: stretch;
    justify-content: flex-start;
    gap: 1.25rem;
    width: 15rem;
    height: 100vh;
    max-height: 100vh;
    overflow-y: auto;
    padding: 1.5rem 1rem 1rem;
    border: 0;
    border-right: 1px solid var(--border);
    background: var(--surface);
    box-shadow: none;
    backdrop-filter: none;
  }

  .rail-label {
    margin: 0.75rem 0 -0.75rem;
    padding: 0 0.65rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.62rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .grapher-header :global(.main-nav) {
    display: grid;
    align-content: start;
    gap: 0.25rem;
    margin: 0;
    padding: 0;
    border: 0;
    border-radius: 0;
    background: transparent;
    box-shadow: none;
    backdrop-filter: none;
  }

  .grapher-header :global(.main-nav > a),
  .grapher-header :global(.navigation-menu > summary) {
    display: flex;
    align-items: center;
    justify-content: flex-start;
    gap: 0.7rem;
    width: 100%;
    min-height: 2.75rem;
    padding: 0.65rem 0.75rem;
    border: 0;
    border-radius: calc(var(--radius) - 3px);
    color: var(--text-muted);
    font-size: 0.82rem;
    font-weight: 650;
    text-align: left;
    text-decoration: none;
  }

  .grapher-header :global(.main-nav > a)::after,
  .grapher-header :global(.navigation-menu > summary)::after {
    display: none;
  }

  .grapher-header :global(.main-nav > a),
  .grapher-header :global(.navigation-menu > summary) {
    transition:
      background-color var(--motion-micro),
      box-shadow var(--motion-micro),
      color var(--motion-micro),
      transform var(--motion-micro);
  }

  .grapher-header :global(.main-nav > a:hover),
  .grapher-header :global(.navigation-menu > summary:hover) {
    background: var(--surface-2);
    color: var(--text);
  }

  .grapher-header :global(.main-nav > a:active) {
    transform: scale(0.98);
  }

  .grapher-header :global(.main-nav > a.active),
  .grapher-header :global(.navigation-menu.active > summary) {
    background: color-mix(in srgb, var(--accent) 14%, var(--surface));
    color: var(--text);
  }

  .grapher-header :global(.main-nav > a.active) {
    box-shadow:
      inset 3px 0 0 var(--accent),
      0 0 12px color-mix(in srgb, var(--accent) 15%, transparent);
  }

  .grapher-header :global(.main-nav > a.active svg) {
    color: var(--accent);
  }

  .grapher-header :global(.navigation-menu) {
    width: 100%;
  }

  .grapher-header :global(.navigation-menu > summary) {
    list-style: none;
    cursor: pointer;
  }

  .grapher-header :global(.navigation-menu .navigation-popover) {
    position: static;
    display: grid;
    gap: 0.15rem;
    min-width: 0;
    margin: 0.2rem 0 0 0.65rem;
    padding: 0.25rem 0 0.25rem 0.7rem;
    border: 0;
    border-left: 1px solid var(--border);
    border-radius: 0;
    background: transparent;
    box-shadow: none;
    backdrop-filter: none;
  }

  .grapher-header :global(.navigation-menu:not([open]) .navigation-popover) {
    display: none;
  }

  .grapher-header :global(.navigation-menu .navigation-popover a) {
    min-height: 2.5rem;
    padding: 0.5rem 0.65rem;
    border-radius: calc(var(--radius) - 4px);
    font-size: 0.78rem;
  }

  .grapher-header :global(.appbar-actions) {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: flex-start;
    gap: 0.5rem;
    width: 100%;
    margin: auto 0 0;
    padding-top: 1rem;
    border-top: 1px solid var(--border);
  }

  .grapher-header :global(.command-trigger) {
    justify-content: space-between;
    width: 100%;
    min-height: 2.75rem;
    border-radius: var(--radius);
    box-shadow: var(--tile-shadow);
    transition:
      border-color var(--motion-micro),
      box-shadow var(--motion-micro),
      transform var(--motion-micro);
  }

  .grapher-header :global(.command-trigger:hover),
  .grapher-header :global(.appbar-actions .user-menu > summary:hover) {
    box-shadow:
      var(--tile-shadow),
      0 8px 20px -6px rgb(0 0 0 / 0.3);
    transform: translateY(-1px);
  }

  .grapher-header :global(.appbar-actions .user-menu > summary) {
    border-radius: var(--radius);
    box-shadow: var(--tile-shadow);
    transition:
      border-color var(--motion-micro),
      box-shadow var(--motion-micro),
      transform var(--motion-micro);
  }

  .grapher-header :global(.appbar-actions .user-menu) {
    flex: 1 1 0;
    min-width: 0;
  }

  .grapher-header :global(.appbar-actions .user-menu > summary) {
    width: 100%;
  }

  .grapher-header :global(.appbar-actions .user-menu .user-name) {
    flex: 1;
    min-width: 0;
    max-width: none;
  }

  .grapher-content {
    grid-column: 2;
    grid-row: 1;
    --shell-width: 1440px;
    width: min(var(--shell-width), calc(100% - 4rem));
    min-width: 0;
    margin: 0 auto;
    padding: 2.5rem 0 4rem;
  }

  .grapher-footer {
    grid-column: 2;
    grid-row: 2;
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    padding: 1rem 2rem;
    border-top: 1px solid var(--border);
    color: var(--text-muted);
    font-size: 0.7rem;
  }

  /* The account control sits at the bottom of the rail, so its menu opens
     upward across the full row. Opening down ran past the viewport, and a
     menu wider than the rail made the rail scroll sideways. */
  @media (min-width: 1025px) {
    .grapher-header :global(.appbar-actions) {
      position: relative;
    }

    .grapher-header :global(.appbar-actions .user-menu) {
      position: static;
    }

    .grapher-header :global(.appbar-actions .user-menu .user-popover) {
      top: auto;
      right: 0;
      bottom: calc(100% + 0.4rem);
      left: 0;
      min-width: 0;
    }
  }

  @media (max-width: 1024px) {
    .grapher-site {
      display: block;
      min-height: 100vh;
    }

    .grapher-header {
      position: sticky;
      z-index: 20;
      display: flex;
      flex-direction: row;
      align-items: center;
      justify-content: space-between;
      gap: 1rem;
      width: 100%;
      height: auto;
      max-height: none;
      overflow: visible;
      padding: 0.75rem 1.25rem;
      border-right: 0;
      border-bottom: 1px solid var(--border);
      background: var(--bg);
    }

    .rail-label {
      display: none;
    }

    .grapher-header :global(.main-nav) {
      display: flex;
      flex-direction: row;
      align-items: center;
      justify-content: flex-start;
      gap: 0.15rem;
      margin: 0 auto;
      padding: 0.2rem;
      border: 0;
      background: transparent;
    }

    .grapher-header :global(.main-nav > a),
    .grapher-header :global(.navigation-menu > summary) {
      width: auto;
      min-height: 2rem;
      padding: 0.42rem 0.62rem;
      border-radius: 999px;
      font-size: 0.76rem;
    }

    .grapher-header :global(.navigation-menu .navigation-popover) {
      position: absolute;
      min-width: 13rem;
      margin: 0;
      padding: 0.45rem;
      border: 1px solid var(--border);
      border-radius: var(--radius);
      background: var(--surface-2);
      box-shadow: var(--tile-shadow);
    }

    .grapher-header :global(.appbar-actions) {
      display: flex;
      flex-direction: row;
      flex-wrap: nowrap;
      align-items: center;
      justify-content: flex-end;
      gap: 0.5rem;
      width: auto;
      margin: 0;
      padding: 0;
      border: 0;
    }

    .grapher-header :global(.command-trigger) {
      width: auto;
      min-height: 2.25rem;
    }

    .grapher-header :global(.appbar-actions .user-menu) {
      flex: 0 0 auto;
    }

    .grapher-header :global(.appbar-actions .user-menu > summary) {
      width: auto;
    }

    .grapher-header :global(.appbar-actions .user-menu .user-name) {
      flex: 0 1 auto;
      max-width: 9rem;
    }

    .grapher-content {
      width: min(var(--shell-width), calc(100% - 3rem));
      padding: 2.25rem 0 4rem;
    }

    .grapher-footer {
      padding: 1rem 1.5rem;
    }
  }

  @media (max-width: 768px) {
    .grapher-header :global(.navigation-menu .navigation-popover) {
      position: fixed;
      top: var(--navigation-popover-top, -9999px);
      left: var(--navigation-popover-left, -9999px);
      width: min(13rem, calc(100vw - 1rem));
      min-width: 0;
      max-height: min(60vh, 20rem);
      overflow-y: auto;
      margin: 0;
      padding: 0.45rem;
      border: 1px solid var(--border);
      border-radius: var(--radius);
      background: var(--surface-2);
      box-shadow: var(--tile-shadow);
    }
  }

  @media (max-width: 640px) {
    .grapher-header {
      flex-wrap: wrap;
      gap: 0.65rem;
      padding: 0.7rem 0.8rem;
    }

    .grapher-header :global(.main-nav) {
      display: grid;
      grid-template-columns: repeat(5, minmax(0, 1fr));
      gap: 0.15rem;
      order: 2;
      width: 100%;
      margin: 0;
      padding: 0.2rem;
      border: 1px solid var(--border);
      border-radius: var(--radius);
      background: var(--surface);
    }

    .grapher-header :global(.main-nav > a),
    .grapher-header :global(.navigation-menu > summary) {
      justify-content: center;
      min-height: 2.5rem;
      padding: 0.42rem 0.1rem;
      font-size: 0.68rem;
    }

    .grapher-header :global(.main-nav > a svg) {
      display: none;
    }

    .grapher-header :global(.appbar-actions) {
      order: 3;
      width: 100%;
      justify-content: flex-end;
    }

    .grapher-content {
      width: min(100% - 2rem, var(--shell-width));
      padding: 1.75rem 0 3rem;
    }

    .grapher-footer {
      flex-direction: column;
      padding: 1rem;
    }
  }
</style>
