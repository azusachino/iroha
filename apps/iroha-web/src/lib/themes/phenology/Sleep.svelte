<script lang="ts">
  import ThemeSleep from "$lib/themes/ThemeSleep.svelte";
  import type { ComponentProps } from "svelte";
  type Props = ComponentProps<typeof ThemeSleep>;
  let { ...props }: Omit<Props, "variant"> = $props();
</script>

<ThemeSleep variant="phenology" {...props} />
