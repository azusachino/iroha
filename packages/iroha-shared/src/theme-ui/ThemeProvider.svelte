<script lang="ts">
  import { onDestroy } from "svelte";
  import type { Snippet } from "svelte";
  import type { DesignLanguage } from "../theme/themes";
  import { getThemeDefinition } from "./registry";
  import { provideTheme } from "./context.svelte";

  let {
    language,
    onSelect,
    children,
  }: {
    language: DesignLanguage;
    onSelect: (language: DesignLanguage) => void;
    children?: Snippet;
  } = $props();

  const definition = $derived(getThemeDefinition(language));

  // The registry is also the source of the semantic token palette. Keep the
  // document attribute in sync here so every host gets the same theme CSS;
  // hosts only provide persistence and selection policy.
  let capturedLanguage: string | undefined;
  let ownsLanguageAttribute = false;

  $effect(() => {
    if (typeof document === "undefined") return;
    const root = document.documentElement;
    if (!ownsLanguageAttribute) {
      capturedLanguage = root.dataset.language;
      ownsLanguageAttribute = true;
    }
    root.dataset.language = language;
  });

  onDestroy(() => {
    if (!ownsLanguageAttribute) return;
    const root = document.documentElement;
    if (capturedLanguage) root.dataset.language = capturedLanguage;
    else delete root.dataset.language;
  });

  provideTheme({
    language: () => language,
    definition: () => definition,
    select: (next) => onSelect(next),
  });
</script>

{@render children?.()}
