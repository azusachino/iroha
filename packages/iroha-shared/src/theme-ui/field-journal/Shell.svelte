<script lang="ts">
  import type { ShellThemeProps } from "../../view-contracts/shell-view";

  let { children, theme, brand, nav, actions }: ShellThemeProps = $props();
</script>

<div class="field-journal-site" data-theme={theme}>
  <header class="appbar field-journal-header">
    {@render brand()}
    {@render nav()}
    <div class="appbar-actions field-journal-header-actions">
      {@render actions()}
    </div>
  </header>
  <div class="field-journal-content">{@render children()}</div>
  <footer class="field-journal-footer">
    <span>iroha field journal · dates, entries, continuity</span>
    <span>Every entry remains traceable to imported evidence.</span>
  </footer>
</div>

<style>
  .field-journal-site {
    --shell-width: 1120px;
    min-height: 100vh;
    background:
      radial-gradient(
        circle at 10% 0%,
        color-mix(in srgb, var(--accent) 12%, transparent),
        transparent 28rem
      ),
      var(--bg);
    color: var(--text);
  }

  .field-journal-content {
    width: min(var(--shell-width), calc(100% - 3rem));
    margin: 0 auto;
    padding: 3rem 0 5rem;
  }

  .field-journal-footer {
    display: flex;
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

  /* Everything below restyles the shared appbar/nav/actions markup into a
     journal masthead; the links/buttons/select themselves are untouched, so
     focus order, tap targets, and accessible names stay whatever the host
     defines. */
  .field-journal-header {
    background: color-mix(in srgb, var(--bg) 94%, transparent);
    border-bottom: 3px double
      color-mix(in srgb, var(--accent) 45%, var(--border));
    box-shadow: none;
    backdrop-filter: none;
  }

  .field-journal-header :global(.brand) {
    font-family: var(--font-serif);
    font-style: italic;
    font-weight: 700;
  }

  .field-journal-header :global(.main-nav) {
    background: transparent;
    border: none;
    box-shadow: none;
    gap: 0;
  }

  .field-journal-header :global(.main-nav > a),
  .field-journal-header :global(.navigation-menu > summary) {
    font-family: var(--font-serif);
    border-radius: 0;
    border-left: 1px dotted
      color-mix(in srgb, var(--accent-2) 55%, var(--border));
  }

  .field-journal-header :global(.main-nav > a:first-child) {
    border-left: none;
  }

  .field-journal-header :global(.main-nav > a.active),
  .field-journal-header :global(.navigation-menu.active > summary) {
    background: transparent;
    color: var(--accent-2);
    text-decoration: underline;
    text-decoration-style: wavy;
    text-decoration-color: var(--accent-2);
    text-underline-offset: 3px;
  }

  .field-journal-header-actions {
    border-radius: var(--radius);
    border: 1px solid color-mix(in srgb, var(--accent-2) 30%, var(--border));
    background: color-mix(in srgb, var(--bg) 90%, transparent);
  }

  @media (max-width: 640px) {
    .field-journal-content {
      width: min(100% - 2rem, var(--shell-width));
      padding-top: 2rem;
    }

    .field-journal-footer {
      flex-direction: column;
      gap: 0.4rem;
      padding: 1rem;
    }
  }
</style>
