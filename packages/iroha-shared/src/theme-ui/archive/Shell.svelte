<script lang="ts">
  import type { ShellThemeProps } from "../../shell-view";
  let { children, theme }: ShellThemeProps = $props();
</script>

<div class="vault-site" data-theme={theme}>
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
  }
</style>
