<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    brand,
    nav,
    actions,
  }: { brand: Snippet; nav: Snippet; actions: Snippet } = $props();
</script>

<!--
  The plain top app-bar shared by every registered language that has not
  earned a bespoke shell/nav treatment yet. Its visual rules (`.appbar`,
  `.brand`, `.main-nav`, `.appbar-actions`) live in the web host's app.css,
  not here — this component only owns arrangement, not styling, so the
  hard-won compact focus-order and standalone-target-size fixes stay in one
  place instead of being copy/pasted per theme.
-->
<header class="appbar">
  {@render brand()}
  {@render nav()}
  <div class="appbar-actions">
    {@render actions()}
  </div>
</header>
