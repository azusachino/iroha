<script lang="ts">
  import { onMount } from "svelte";
  import { Sun, Moon, Monitor } from "@lucide/svelte";
  import {
    cycleTheme,
    getEffectiveTheme,
    getThemePreference,
    type EffectiveTheme,
    type ThemePreference,
  } from "$lib/theme";

  let preference = $state<ThemePreference>("auto");
  let effectiveTheme = $state<EffectiveTheme>("dark");

  function syncState() {
    preference = getThemePreference();
    effectiveTheme = getEffectiveTheme();
  }

  onMount(() => {
    syncState();
    const observer = new MutationObserver(() => {
      syncState();
    });
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["data-theme", "data-theme-preference"],
    });
    return () => observer.disconnect();
  });

  function onClick() {
    cycleTheme();
    syncState();
  }
</script>

<button
  class="theme-toggle"
  type="button"
  aria-label={`Theme: ${preference} (currently ${effectiveTheme}). Click to cycle.`}
  title={`Theme: ${preference} (${effectiveTheme})`}
  onclick={onClick}
>
  {#if preference === "auto"}
    <Monitor size={17} />
  {:else if effectiveTheme === "dark"}
    <Sun size={17} />
  {:else}
    <Moon size={17} />
  {/if}
</button>
