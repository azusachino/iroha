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
    gap: var(--space-5);
    width: 15rem;
    height: 100vh;
    max-height: 100vh;
    overflow-y: auto;
    padding: var(--space-6) var(--space-4) var(--space-4);
    border: 0;
    border-right: 1px solid var(--border);
    background: var(--surface);
    box-shadow: none;
  }

  .rail-label {
    margin: var(--space-3) 0 calc(var(--space-3) * -1);
    padding: 0 0.65rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: var(--type-caption);
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .grapher-header :global(.main-nav) {
    display: grid;
    align-content: start;
    gap: var(--space-1);
    margin: 0;
    padding: 0;
    border: 0;
    border-radius: 0;
    background: transparent;
    box-shadow: none;
  }

  .grapher-header :global(.main-nav > a),
  .grapher-header :global(.navigation-menu > .navigation-trigger) {
    display: flex;
    align-items: center;
    justify-content: flex-start;
    gap: 0.7rem;
    width: 100%;
    min-height: var(--control-height-lg);
    padding: 0.65rem var(--space-3);
    border: 0;
    border-radius: var(--radius-md);
    color: var(--text-muted);
    font-size: var(--type-label);
    font-weight: 650;
    text-align: left;
    text-decoration: none;
  }

  .grapher-header :global(.main-nav > a)::after,
  .grapher-header :global(.navigation-menu > .navigation-trigger)::after {
    display: none;
  }

  .grapher-header :global(.main-nav > a),
  .grapher-header :global(.navigation-menu > .navigation-trigger) {
    transition:
      background-color var(--motion-micro),
      box-shadow var(--motion-micro),
      color var(--motion-micro),
      transform var(--motion-micro);
  }

  .grapher-header :global(.main-nav > a:hover),
  .grapher-header :global(.navigation-menu > .navigation-trigger:hover) {
    background: var(--surface-2);
    color: var(--text);
  }

  .grapher-header :global(.main-nav > a:active) {
    transform: scale(0.98);
  }

  .grapher-header :global(.main-nav > a.active),
  .grapher-header :global(.navigation-menu.active > .navigation-trigger) {
    background: color-mix(in srgb, var(--accent) 14%, var(--surface));
    color: var(--text);
  }

  .grapher-header :global(.main-nav > a.active) {
    box-shadow:
      inset 3px 0 0 var(--accent),
      0 0 12px color-mix(in srgb, var(--accent) 15%, transparent);
  }

  .grapher-header :global(.main-nav > a.active svg),
  .grapher-header :global(.navigation-menu.active > .navigation-trigger svg) {
    color: var(--accent);
  }

  .grapher-header :global(.navigation-menu) {
    width: 100%;
  }

  .grapher-header :global(.navigation-menu > .navigation-trigger) {
    cursor: pointer;
  }

  .grapher-header :global(.navigation-menu.inline .navigation-popover) {
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
  }

  .grapher-header :global(.navigation-menu.inline .navigation-popover[hidden]) {
    display: none;
  }

  .grapher-header :global(.navigation-menu .navigation-popover a) {
    min-height: var(--control-height);
    padding: var(--space-2) 0.65rem;
    border-radius: var(--radius-md);
    font-size: var(--type-label);
  }

  .grapher-header :global(.appbar-actions) {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: flex-start;
    gap: var(--space-2);
    width: 100%;
    margin: auto 0 0;
    padding-top: var(--space-4);
    border-top: 1px solid var(--border);
  }

  .grapher-header :global(.command-trigger) {
    justify-content: space-between;
    width: 100%;
    min-height: var(--control-height-lg);
    border-radius: var(--radius);
    box-shadow: var(--tile-shadow), var(--tile-rim);
    transition:
      border-color var(--motion-micro),
      box-shadow var(--motion-micro),
      transform var(--motion-micro);
  }

  .grapher-header :global(.command-trigger:hover),
  .grapher-header :global(.appbar-actions .user-menu > .user-menu-trigger:hover) {
    box-shadow: var(--tile-shadow-hover), var(--tile-rim);
    transform: translateY(-1px);
  }

  .grapher-header :global(.appbar-actions .user-menu > .user-menu-trigger) {
    border-radius: var(--radius);
    box-shadow: var(--tile-shadow), var(--tile-rim);
    transition:
      border-color var(--motion-micro),
      box-shadow var(--motion-micro),
      transform var(--motion-micro);
  }

  .grapher-header :global(.appbar-actions .user-menu) {
    flex: 1 1 0;
    min-width: 0;
  }

  .grapher-header :global(.appbar-actions .user-menu > .user-menu-trigger) {
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
    width: min(var(--shell-width), calc(100% - var(--gutter-page) * 2));
    min-width: 0;
    margin: 0 auto;
    padding: var(--space-10) 0 4rem;
  }

  .grapher-footer {
    grid-column: 2;
    grid-row: 2;
    display: flex;
    justify-content: space-between;
    gap: var(--space-4);
    padding: var(--space-4) var(--gutter-page);
    border-top: 1px solid var(--border);
    color: var(--text-muted);
    font-size: var(--type-caption);
  }

  /* The account control sits at the bottom of the rail, so its menu opens
     upward from the trigger using CSS Anchor Positioning (with JS fallback). */
  @media (min-width: 1025px) {
    @supports (position-area: block-end) {
      .grapher-header :global(.appbar-actions .user-menu .user-popover) {
        position-area: block-start span-inline-end;
        position-try-fallbacks: flip-block, flip-inline, flip-block flip-inline;
        inset: auto;
        margin: 0 0 0.4rem;
        min-width: anchor-size(width);
      }
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
    .grapher-header :global(.navigation-menu > .navigation-trigger) {
      width: auto;
      min-height: var(--control-height);
      padding: 0.42rem 0.62rem;
      border-radius: var(--radius-pill);
      font-size: var(--type-caption);
    }

    .grapher-header :global(.appbar-actions) {
      display: flex;
      flex-direction: row;
      flex-wrap: nowrap;
      align-items: center;
      justify-content: flex-end;
      gap: var(--space-2);
      width: auto;
      margin: 0;
      padding: 0;
      border: 0;
    }

    .grapher-header :global(.command-trigger) {
      width: auto;
      min-height: var(--control-height);
    }

    .grapher-header :global(.appbar-actions .user-menu) {
      flex: 0 0 auto;
    }

    .grapher-header :global(.appbar-actions .user-menu > .user-menu-trigger) {
      width: auto;
    }

    .grapher-header :global(.appbar-actions .user-menu .user-name) {
      flex: 0 1 auto;
      max-width: 9rem;
    }

    .grapher-content {
      padding: 2.25rem 0 4rem;
    }

    .grapher-footer {
      padding: var(--space-4) var(--gutter-page);
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
    .grapher-header :global(.navigation-menu > .navigation-trigger) {
      justify-content: center;
      min-height: var(--control-height);
      padding: 0.42rem 0.1rem;
      font-size: var(--type-caption);
    }

    .grapher-header :global(.main-nav > a svg),
    .grapher-header :global(.navigation-menu > .navigation-trigger svg) {
      display: none;
    }

    .grapher-header :global(.appbar-actions) {
      order: 3;
      width: 100%;
      justify-content: flex-end;
    }

    .grapher-content {
      padding: 1.75rem 0 3rem;
    }

    .grapher-footer {
      flex-direction: column;
      padding: var(--space-4) var(--gutter-page);
    }
  }
</style>
