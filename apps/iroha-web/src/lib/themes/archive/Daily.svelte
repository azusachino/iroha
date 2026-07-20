<script lang="ts">
  import ThemeDaily from "$lib/themes/ThemeDaily.svelte";
  import type { ComponentProps } from "svelte";
  type Props = ComponentProps<typeof ThemeDaily>;
  let { chrono, gran, onGran }: Omit<Props, "variant"> = $props();
</script>

<ThemeDaily variant="archive" {chrono} {gran} {onGran} />
