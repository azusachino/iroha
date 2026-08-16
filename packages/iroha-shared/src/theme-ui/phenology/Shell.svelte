<script lang="ts">
  import type { ShellThemeProps } from "../../view-contracts/shell-view";

  let { children, theme }: ShellThemeProps = $props();
</script>

<div class="bloom-site" data-theme={theme}>
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
  }
</style>
