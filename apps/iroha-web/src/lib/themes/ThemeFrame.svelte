<script lang="ts">
  import type { Snippet } from "svelte";
  import { fade } from "svelte/transition";
  import { browser } from "$app/environment";
  import { APP_VERSION } from "$lib/config";
  import { useTheme } from "$lib/themes/context.svelte";

  let { children }: { children: Snippet } = $props();
  const theme = useTheme();
  const Shell = $derived(
    theme.definition().components?.shell as unknown as
      import("svelte").Component<Record<string, unknown>> | undefined,
  );

  // Mirrors --motion-language-switch (themes.css); Svelte's transition
  // duration is a JS number, not the CSS custom property directly.
  const languageSwitch = browser
    ? window.matchMedia("(prefers-reduced-motion: reduce)").matches
      ? { duration: 0 }
      : { duration: 300 }
    : { duration: 0 };
</script>

{#if Shell}
  {#key theme.language()}
    <div class="theme-transition" transition:fade={languageSwitch}>
      <Shell theme={theme.language()}>
        <main id="main-content" class="theme-content" tabindex="-1">
          {@render children()}
        </main>
      </Shell>
    </div>
  {/key}
{:else}
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

  /* Layout-neutral: renders no box of its own, only exists as a transition
     anchor for the language cross-fade so it doesn't disturb whatever flex
     layout the shell it wraps expects to participate in. */
  .theme-transition {
    display: contents;
  }
</style>
