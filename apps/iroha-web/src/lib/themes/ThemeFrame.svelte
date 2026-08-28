<script lang="ts">
  import type { Snippet } from "svelte";
  import { fade, fly, scale } from "svelte/transition";
  import { backOut, cubicOut } from "svelte/easing";
  import { browser } from "$app/environment";
  import { APP_VERSION } from "$lib/config";
  import { useTheme } from "$lib/themes/context.svelte";
  import type { DesignLanguage } from "@iroha/shared/theme/themes";

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
  // duration is a JS number, not the CSS custom property directly.
  const reducedMotion = browser
    ? window.matchMedia("(prefers-reduced-motion: reduce)").matches
    : false;
  const languageSwitch = { duration: reducedMotion ? 0 : 300 };

  // One motion signature per language instead of a single shared cross-fade
  // -- Grapher keeps the plain fade, matching its identity as the plainest
  // of the six. `theme.language()` is read fresh on every call, which is
  // safe here because {#key theme.language()} below already guarantees a
  // fresh element (and therefore a fresh transition call) on every switch.
  function themeTransition(node: Element, params: { duration: number }) {
    const language: DesignLanguage = theme.language();
    const { duration } = params;
    switch (language) {
      case "atlas":
        return fly(node, { x: 16, duration, easing: cubicOut });
      case "field-journal":
        return scale(node, { start: 0.98, duration, easing: cubicOut });
      case "phenology":
        return scale(node, {
          start: 0.92,
          duration: duration * 1.3,
          easing: backOut,
        });
      case "sound-map":
        return fly(node, {
          y: -8,
          duration: duration * 0.7,
          easing: cubicOut,
        });
      case "archive":
        return scale(node, {
          start: 1.05,
          duration: duration * 0.7,
          easing: cubicOut,
        });
      default:
        return fade(node, { duration });
    }
  }
</script>

{#if Shell}
  {#key theme.language()}
    <div class="theme-transition" transition:themeTransition={languageSwitch}>
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
