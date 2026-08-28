<script lang="ts">
  import type { ShellThemeProps } from "../../view-contracts/shell-view";
  let { children, theme, brand, nav, actions }: ShellThemeProps = $props();
</script>

<div class="vault-site" data-theme={theme}>
  <header class="appbar vault-header">
    <div class="vault-header-brand">
      <span class="vault-stamp vault-stamp-header" aria-hidden="true">№</span>
      {@render brand()}
    </div>
    {@render nav()}
    <div class="appbar-actions vault-header-actions">
      {@render actions()}
    </div>
  </header>
  <div class="vault-content">{@render children()}</div>
  <footer class="vault-footer">
    <span class="vault-stamp" aria-hidden="true">№</span>
    <span>iroha archive · media, history, provenance</span>
    <span>every record keeps its accession trail</span>
  </footer>
</div>

<style>
  .vault-site {
    --shell-width: 1180px;
    position: relative;
    min-height: 100vh;
    background:
      repeating-linear-gradient(
        0deg,
        color-mix(in srgb, var(--accent-2) 6%, transparent) 0 1px,
        transparent 1px 2.75rem
      ),
      radial-gradient(
        circle at 92% 4%,
        color-mix(in srgb, var(--accent) 11%, transparent),
        transparent 26rem
      ),
      radial-gradient(
        circle at 4% 94%,
        color-mix(in srgb, var(--accent-2) 9%, transparent),
        transparent 24rem
      ),
      var(--bg);
    color: var(--text);
    font-family: var(--font-serif);
  }

  .vault-content {
    position: relative;
    width: min(var(--shell-width), calc(100% - 3rem));
    margin: 0 auto;
    padding: 3rem 0 4rem 2.25rem;
    border-left: 1px solid
      color-mix(in srgb, var(--accent-2) 35%, var(--border));
  }

  .vault-footer {
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
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .vault-stamp {
    display: grid;
    place-items: center;
    width: 1.6rem;
    height: 1.6rem;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    color: var(--accent);
    font-family: var(--font-serif);
    font-size: 0.95rem;
  }

  .vault-stamp-header {
    width: 1.3rem;
    height: 1.3rem;
    font-size: 0.7rem;
  }

  /* Everything below restyles the shared appbar/nav/actions markup into a
     card-catalog drawer label strip; the links/buttons/select themselves
     are untouched, so focus order, tap targets, and accessible names stay
     whatever the host defines. */
  .vault-header {
    background: color-mix(in srgb, var(--bg) 90%, transparent);
    border-bottom: 1px solid
      color-mix(in srgb, var(--accent-2) 40%, var(--border));
    box-shadow: none;
  }

  .vault-header-brand {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    min-width: max-content;
  }

  .vault-header :global(.brand) {
    font-family: var(--font-serif);
  }

  .vault-header :global(.main-nav) {
    border-radius: 0;
    background: color-mix(in srgb, var(--bg) 80%, transparent);
    border: 1px solid color-mix(in srgb, var(--accent-2) 35%, var(--border));
  }

  .vault-header :global(.main-nav > a),
  .vault-header :global(.navigation-menu > summary) {
    border-radius: 0;
    border-right: 1px solid
      color-mix(in srgb, var(--accent-2) 35%, var(--border));
    font-family: var(--font-mono);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .vault-header :global(.main-nav > a:last-child) {
    border-right: none;
  }

  .vault-header :global(.main-nav > a.active),
  .vault-header :global(.navigation-menu.active > summary) {
    background: color-mix(in srgb, var(--accent-2) 22%, transparent);
    color: var(--text);
  }

  .vault-header-actions {
    border-radius: 0;
    border: 1px solid color-mix(in srgb, var(--accent-2) 35%, var(--border));
    background: color-mix(in srgb, var(--bg) 80%, transparent);
  }

  @media (max-width: 640px) {
    .vault-content {
      width: min(100% - 2rem, var(--shell-width));
      padding: 2rem 0 3rem 1.25rem;
    }

    .vault-footer {
      flex-direction: column;
      align-items: flex-start;
      gap: 0.4rem;
      padding: 1rem;
    }

    .vault-stamp-header {
      display: none;
    }
  }
</style>
