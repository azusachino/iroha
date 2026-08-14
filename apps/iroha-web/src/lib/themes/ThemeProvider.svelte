<script lang="ts">
  import { onMount } from "svelte";
  import { getDesignLanguage, setDesignLanguage } from "$lib/design-language";
  import SharedThemeProvider from "@iroha/shared/theme-ui/ThemeProvider.svelte";
  import type { DesignLanguage } from "@iroha/shared/themes";

  let { children } = $props();
  let language = $state<DesignLanguage>(getDesignLanguage());
  function select(next: DesignLanguage) {
    language = next;
    setDesignLanguage(next);
  }

  onMount(() => {
    language = getDesignLanguage();
  });
</script>

<SharedThemeProvider {language} onSelect={select}>
  {@render children?.()}
</SharedThemeProvider>
