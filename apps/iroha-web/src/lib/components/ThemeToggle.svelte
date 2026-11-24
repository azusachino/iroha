<script lang="ts">
  import { Sun, Moon, Monitor } from "@lucide/svelte";
  import { cycleTheme, themeState } from "$lib/theme";
</script>

<button
  class="theme-toggle"
  type="button"
  aria-label={`Theme: ${themeState.preference} (currently ${themeState.effective}). Click to cycle.`}
  title={`Theme: ${themeState.preference} (${themeState.effective})`}
  onclick={() => cycleTheme()}
>
  {#if themeState.preference === "auto"}
    <Monitor size={17} />
  {:else if themeState.effective === "dark"}
    <Sun size={17} />
  {:else}
    <Moon size={17} />
  {/if}
</button>
