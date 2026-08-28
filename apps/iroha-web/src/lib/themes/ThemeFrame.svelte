<script lang="ts">
  import type { Snippet } from "svelte";
  import { fade } from "svelte/transition";
  import { browser } from "$app/environment";
  import { APP_VERSION } from "$lib/config";
  import { useTheme } from "$lib/themes/context.svelte";

  let {
    children,
    brand,
    nav,
    actions,
  }: { children: Snippet; brand: Snippet; nav: Snippet; actions: Snippet } =
    $props();
  const theme = useTheme();
  const Shell = $derived(
    theme.definition().components?.shell as unknown as
      import("svelte").Component<Record<string, unknown>> | undefined,
  );

  // Mirrors --motion-language-switch (themes.css); Svelte's transition
  // duration is a JS number, not the CSS custom property directly. One
  // plain cross-fade for every language -- distinct pan/bloom/stamp motion
  // per theme read as gimmicky in practice; a full-composition swap should
  // stay calm regardless of which language is switching in.
  const languageSwitch = browser
    ? window.matchMedia("(prefers-reduced-motion: reduce)").matches
      ? { duration: 0 }
      : { duration: 300 }
    : { duration: 0 };
</script>

{#if Shell}
  {#key theme.language()}
    <div class="theme-transition" transition:fade={languageSwitch}>
      <Shell theme={theme.language()} {brand} {nav} {actions}>
        <main id="main-content" class="theme-content" tabindex="-1">
          {@render children()}
        </main>
      </Shell>
    </div>
  {/key}
{:else}
  <header class="appbar">
    {@render brand()}
    {@render nav()}
    <div class="appbar-actions">
      {@render actions()}
    </div>
  </header>
  <main id="main-content" class="content" tabindex="-1">
    {@render children()}
  </main>
  <footer class="footer">Private activity viewer · v{APP_VERSION}</footer>
{/if}

<style>
  .theme-content,
  .content {
    min-width: 0;
    scroll-margin-top: 8rem;
  }

  /* `display: contents` here would generate no box at all, which makes any
     opacity/transform-based transition (fade, fly, scale) a silent no-op --
     there is nothing for the browser to fade or move. A plain block still
     behaves as a normal flex item of `.app` (display:flex; flex-direction:
     column), since neither this wrapper nor the Shell root it contains
     relies on flex-grow from `.app`. */
  .theme-transition {
    display: block;
    min-width: 0;
  }
</style>
