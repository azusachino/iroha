<script lang="ts">
  import type { Snippet } from "svelte";

  let { children }: { children: Snippet } = $props();
</script>

<div class="field-journal-site">
  <div class="field-journal-content">{@render children()}</div>
</div>

<style>
  .field-journal-site {
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
    width: min(1120px, calc(100% - 3rem));
    margin: 0 auto;
    padding: 3rem 0 5rem;
  }

  @media (max-width: 520px) {
    .field-journal-content {
      width: min(100% - 2rem, 1120px);
      padding-top: 2rem;
    }
  }
</style>
