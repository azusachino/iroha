<script lang="ts">
  import type { ShellThemeProps } from "../../shell-view";

  let { children, theme }: ShellThemeProps = $props();
</script>

<div class="field-journal-site" data-theme={theme}>
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
