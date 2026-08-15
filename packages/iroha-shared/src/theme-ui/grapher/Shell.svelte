<script lang="ts">
  import type { ShellThemeProps } from "../../shell-view";

  let { children, theme }: ShellThemeProps = $props();
</script>

<div class="grapher-site" data-theme={theme}>
  <main class="grapher-content">{@render children()}</main>
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
