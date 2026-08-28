<script lang="ts">
  import type { ShellThemeProps } from "../../view-contracts/shell-view";

  let { children, theme, brand, nav, actions }: ShellThemeProps = $props();
</script>

<div class="grapher-site" data-theme={theme}>
  <header class="appbar grapher-header">
    {@render brand()}
    {@render nav()}
    <div class="appbar-actions">
      {@render actions()}
    </div>
  </header>
  <div class="grapher-content">{@render children()}</div>
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

  .grapher-content {
    --shell-width: 1180px;
    width: min(var(--shell-width), calc(100% - 3rem));
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

  /* Everything below restyles the shared appbar/nav/actions markup into a
     flat, axis-tick control-room bar; the links/buttons/select themselves
     are untouched, so focus order, tap targets, and accessible names stay
     whatever the host defines. Deliberately the plainest of the six shells
     -- Grapher's identity is restraint, not decoration. */
  .grapher-header {
    background: var(--bg);
    border-bottom: 1px solid var(--border);
    box-shadow: none;
    backdrop-filter: none;
  }

  .grapher-header :global(.main-nav) {
    background: transparent;
    border: none;
    box-shadow: none;
    gap: 0.1rem;
  }

  .grapher-header :global(.main-nav > a),
  .grapher-header :global(.navigation-menu > summary) {
    position: relative;
    border-radius: 0;
  }

  .grapher-header :global(.main-nav > a)::after,
  .grapher-header :global(.navigation-menu > summary)::after {
    content: "";
    position: absolute;
    left: 0.6rem;
    right: 0.6rem;
    bottom: 0;
    height: 2px;
    background: transparent;
    transition: background var(--motion-micro, 160ms ease-out);
  }

  .grapher-header :global(.main-nav > a:hover)::after,
  .grapher-header :global(.main-nav > a.active)::after,
  .grapher-header :global(.navigation-menu.active > summary)::after {
    background: var(--accent);
  }

  .grapher-header :global(.main-nav > a.active),
  .grapher-header :global(.navigation-menu.active > summary) {
    background: transparent;
  }

  @media (max-width: 640px) {
    .grapher-content {
      width: min(100% - 2rem, var(--shell-width));
      padding-top: 2rem;
    }

    .grapher-footer {
      flex-direction: column;
      padding: 1rem;
    }
  }
</style>
