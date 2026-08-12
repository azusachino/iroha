<script lang="ts">
  import type { Snippet } from "svelte";
  import { useTheme } from "$lib/themes/context.svelte";

  let {
    route,
    children,
  }: {
    route: "expenses" | "reports";
    children?: Snippet;
  } = $props();

  const theme = useTheme();
  const language = $derived(theme.definition().id);
</script>

<div class={`cockpit-frame cockpit-frame-${language} cockpit-frame-${route}`}>
  <div class="cockpit-frame-mark" aria-hidden="true">
    {route === "reports" ? "◫" : "₿"}
  </div>
  {@render children?.()}
</div>

<style>
  .cockpit-frame {
    position: relative;
    min-height: 12rem;
  }

  .cockpit-frame-mark {
    position: absolute;
    top: 0.35rem;
    right: 0;
    color: color-mix(in srgb, var(--accent) 42%, transparent);
    font-size: 2.4rem;
    line-height: 1;
    pointer-events: none;
  }

  .cockpit-frame-atlas .cockpit-frame-mark {
    border: 1px solid currentColor;
    border-radius: 50%;
    padding: 0.25rem;
    font-family: sans-serif;
    font-size: 1.5rem;
  }

  .cockpit-frame-grapher {
    font-family: "IBM Plex Mono", "SFMono-Regular", monospace;
  }

  .cockpit-frame-grapher .cockpit-frame-mark {
    font-size: 1.5rem;
  }

  .cockpit-frame-field-journal .cockpit-frame-mark,
  .cockpit-frame-phenology .cockpit-frame-mark,
  .cockpit-frame-archive .cockpit-frame-mark {
    font-family: Georgia, serif;
  }

  .cockpit-frame-sound-map .cockpit-frame-mark {
    color: var(--accent);
    text-shadow: 0 0 1rem color-mix(in srgb, var(--accent) 70%, transparent);
  }

  .cockpit-frame-archive .cockpit-frame-mark {
    font-size: 2rem;
  }
</style>
