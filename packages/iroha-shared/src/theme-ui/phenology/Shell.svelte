<script lang="ts">
  import type { ShellThemeProps } from "../../view-contracts/shell-view";

  let { children, theme, brand, nav, actions }: ShellThemeProps = $props();
</script>

<div class="bloom-site" data-theme={theme}>
  <header class="appbar bloom-header">
    {@render brand()}
    {@render nav()}
    <div class="appbar-actions bloom-header-actions">
      {@render actions()}
    </div>
  </header>
  <div class="bloom-content">{@render children()}</div>
  <footer class="bloom-footer">
    <span class="bloom-phases" aria-hidden="true">○ ◔ ◑ ◕ ●</span>
    <span>iroha phenology · sleep, seasons, recovery</span>
    <span>every reading returns to its season</span>
  </footer>
</div>

<style>
  .bloom-site {
    --shell-width: 1120px;
    position: relative;
    min-height: 100vh;
    background:
      repeating-radial-gradient(
        circle at 100% 0%,
        color-mix(in srgb, var(--accent) 5%, transparent) 0 1px,
        transparent 1px 3rem
      ),
      radial-gradient(
        circle at 12% 6%,
        color-mix(in srgb, var(--accent) 14%, transparent),
        transparent 30rem
      ),
      radial-gradient(
        circle at 90% 94%,
        color-mix(in srgb, var(--accent-2) 12%, transparent),
        transparent 28rem
      ),
      var(--bg);
    color: var(--text);
    font-family: var(--font-serif);
  }

  .bloom-content {
    width: min(var(--shell-width), calc(100% - 3rem));
    margin: 0 auto;
    padding: 3rem 0 4rem;
  }

  .bloom-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding: 1rem max(1rem, calc((100% - var(--shell-width)) / 2));
    background: color-mix(in srgb, var(--bg) 88%, transparent);
    color: var(--text-muted);
    font-size: 0.68rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }

  .bloom-phases {
    color: var(--accent);
    letter-spacing: 0.3em;
    font-size: 0.85rem;
  }

  /* Everything below restyles the shared appbar/nav/actions markup into a
     seasonal-wheel bar -- a phase dot per item instead of a flat fill; the
     links/buttons/select themselves are untouched, so focus order, tap
     targets, and accessible names stay whatever the host defines. */
  .bloom-header {
    background: color-mix(in srgb, var(--bg) 92%, transparent);
    border-bottom: 1px solid
      color-mix(in srgb, var(--accent) 25%, var(--border));
    box-shadow: none;
  }

  .bloom-header :global(.brand) {
    font-family: var(--font-serif);
  }

  .bloom-header :global(.main-nav) {
    border-radius: 999px;
    background: color-mix(in srgb, var(--bg) 70%, transparent);
    border: 1px solid color-mix(in srgb, var(--accent) 20%, var(--border));
  }

  .bloom-header :global(.main-nav > a),
  .bloom-header :global(.navigation-menu > summary) {
    position: relative;
    border-radius: 999px;
    padding-left: 1.3rem;
  }

  .bloom-header :global(.main-nav > a)::before,
  .bloom-header :global(.navigation-menu > summary)::before {
    content: "";
    position: absolute;
    left: 0.55rem;
    top: 50%;
    width: 0.4rem;
    height: 0.4rem;
    transform: translateY(-50%);
    border-radius: 999px;
    border: 1px solid color-mix(in srgb, var(--text-muted) 60%, transparent);
    background: transparent;
  }

  .bloom-header :global(.main-nav > a.active)::before,
  .bloom-header :global(.navigation-menu.active > summary)::before {
    background: var(--accent);
    border-color: var(--accent);
    box-shadow: 0 0 0.4rem color-mix(in srgb, var(--accent) 60%, transparent);
  }

  .bloom-header-actions {
    border-radius: 999px;
    background: color-mix(in srgb, var(--bg) 70%, transparent);
    border: 1px solid color-mix(in srgb, var(--accent) 20%, var(--border));
  }

  @media (max-width: 640px) {
    .bloom-content {
      width: min(100% - 2rem, var(--shell-width));
      padding-top: 2rem;
    }

    .bloom-footer {
      flex-direction: column;
      align-items: flex-start;
      gap: 0.4rem;
      padding: 1rem;
    }

    .bloom-header :global(.main-nav > a),
    .bloom-header :global(.navigation-menu > summary) {
      padding-left: 0.1rem;
    }

    .bloom-header :global(.main-nav > a)::before,
    .bloom-header :global(.navigation-menu > summary)::before {
      display: none;
    }
  }
</style>
