<script lang="ts">
  import type { Snippet } from "svelte";

  let { children }: { children: Snippet } = $props();
</script>

<div class="grapher-site">
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
    width: min(1180px, calc(100% - 3rem));
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

  @media (max-width: 520px) {
    .grapher-content {
      width: min(100% - 2rem, 1180px);
      padding-top: 2rem;
    }

    .grapher-footer {
      flex-direction: column;
      padding: 1rem;
    }
  }
</style>
