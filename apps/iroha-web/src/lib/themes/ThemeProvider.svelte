<script lang="ts">
  import { onMount } from "svelte";
  import { getDesignLanguage, setDesignLanguage } from "$lib/design-language";
  import { getThemeDefinition } from "$lib/themes/registry";
  import type { DesignLanguage } from "@iroha/shared/themes";
  import { provideTheme } from "$lib/themes/context.svelte";

  let { children } = $props();
  let language = $state<DesignLanguage>(getDesignLanguage());
  const definition = $derived(getThemeDefinition(language));

  function select(next: DesignLanguage) {
    language = next;
    setDesignLanguage(next);
  }

  provideTheme({
    language: () => language,
    definition: () => definition,
    select,
  });

  onMount(() => {
    language = getDesignLanguage();
  });
</script>

{@render children?.()}
