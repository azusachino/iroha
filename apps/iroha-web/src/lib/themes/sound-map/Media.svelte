<script lang="ts">
  import ThemeMedia from "$lib/themes/ThemeMedia.svelte";
  import type { ComponentProps } from "svelte";
  type Props = ComponentProps<typeof ThemeMedia>;
  let { ...props }: Omit<Props, "variant"> = $props();
</script>

<ThemeMedia variant="sound-map" {...props} />
