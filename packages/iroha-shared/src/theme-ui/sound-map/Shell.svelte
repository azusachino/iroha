<script lang="ts">
  import type { ShellThemeProps } from "../../view-contracts/shell-view";
  let { children, theme, brand, nav, actions }: ShellThemeProps = $props();
</script>

<div class="mix-site" data-theme={theme}>
  <header class="appbar mix-console">
    {@render brand()}
    <span class="mix-console-tag" aria-hidden="true">CH</span>
    {@render nav()}
    <div class="appbar-actions mix-console-master">
      <span class="mix-console-tag" aria-hidden="true">MST</span>
      {@render actions()}
    </div>
  </header>
  <div class="mix-content">{@render children()}</div>
  <footer class="mix-footer">
    <span class="mix-meter" aria-hidden="true">
      <i></i><i></i><i></i><i></i><i></i><i></i><i></i><i></i>
    </span>
    <span>iroha sound-map · rhythm, cadence, intensity</span>
    <span>every level traces back to an imported source</span>
  </footer>
</div>

<style>
  .mix-site {
    --shell-width: 1160px;
    position: relative;
    min-height: 100vh;
    background:
      repeating-linear-gradient(
        0deg,
        color-mix(in srgb, var(--accent) 5%, transparent) 0 1px,
        transparent 1px 2.5rem
      ),
      radial-gradient(
        circle at 88% 2%,
        color-mix(in srgb, var(--accent) 12%, transparent),
        transparent 26rem
      ),
      radial-gradient(
        circle at 4% 96%,
        color-mix(in srgb, var(--accent-2) 10%, transparent),
        transparent 24rem
      ),
      var(--bg);
    color: var(--text);
    font-family: var(--font-mono);
  }

  .mix-content {
    width: min(var(--shell-width), calc(100% - 3rem));
    margin: 0 auto;
    padding: 3rem 0 4rem;
  }

  .mix-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding: 1rem max(1rem, calc((100% - var(--shell-width)) / 2));
    background: color-mix(in srgb, var(--bg) 88%, transparent);
    color: var(--text-muted);
    font-size: 0.65rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .mix-meter {
    display: inline-flex;
    align-items: flex-end;
    gap: 0.15rem;
    height: 0.9rem;
  }

  .mix-meter i {
    display: block;
    width: 0.16rem;
    background: var(--accent);
    opacity: 0.8;
  }

  .mix-meter i:nth-child(1) {
    height: 30%;
  }
  .mix-meter i:nth-child(2) {
    height: 55%;
  }
  .mix-meter i:nth-child(3) {
    height: 80%;
  }
  .mix-meter i:nth-child(4) {
    height: 100%;
    background: var(--accent-2);
  }
  .mix-meter i:nth-child(5) {
    height: 65%;
  }
  .mix-meter i:nth-child(6) {
    height: 90%;
    background: var(--accent-2);
  }
  .mix-meter i:nth-child(7) {
    height: 45%;
  }
  .mix-meter i:nth-child(8) {
    height: 22%;
  }

  /* Everything below restyles the shared appbar/nav/actions markup into a
     mixing-console rack; the elements themselves (links, buttons, the
     select) are unchanged, so focus order, tap targets, and accessible
     names stay whatever the host defines. */
  .mix-console {
    --console-line: color-mix(in srgb, var(--accent) 30%, var(--border));
    background: linear-gradient(
      180deg,
      color-mix(in srgb, var(--surface) 94%, transparent),
      color-mix(in srgb, var(--bg) 97%, transparent)
    );
    border-bottom: 1px solid var(--console-line);
    box-shadow: none;
  }

  .mix-console :global(.brand) {
    font-family: var(--font-mono);
    text-transform: uppercase;
    font-size: 0.85rem;
  }

  .mix-console-tag {
    flex: 0 0 auto;
    padding: 0.15rem 0.3rem;
    color: var(--text-muted);
    font-size: 0.58rem;
    font-weight: 700;
    letter-spacing: 0.12em;
  }

  .mix-console :global(.main-nav) {
    background: color-mix(in srgb, var(--bg) 65%, transparent);
    border: 1px solid var(--console-line);
    border-radius: 3px;
    padding: 0;
    gap: 0;
  }

  .mix-console :global(.main-nav > a),
  .mix-console :global(.navigation-menu > summary) {
    border-radius: 0;
    border-right: 1px solid var(--console-line);
  }

  .mix-console :global(.main-nav > a:last-child) {
    border-right: none;
  }

  .mix-console :global(.main-nav > a.active),
  .mix-console :global(.navigation-menu.active > summary) {
    background: color-mix(in srgb, var(--accent) 18%, transparent);
    box-shadow: inset 0 -2px 0 var(--accent);
  }

  .mix-console-master {
    padding: 0.15rem 0.4rem;
    border: 1px solid var(--console-line);
    border-radius: 3px;
    background: color-mix(in srgb, var(--bg) 65%, transparent);
  }

  @media (max-width: 640px) {
    .mix-content {
      width: min(100% - 2rem, var(--shell-width));
      padding-top: 2rem;
    }

    .mix-footer {
      flex-direction: column;
      align-items: flex-start;
      gap: 0.4rem;
      padding: 1rem;
    }

    .mix-console-tag {
      display: none;
    }
  }
</style>
