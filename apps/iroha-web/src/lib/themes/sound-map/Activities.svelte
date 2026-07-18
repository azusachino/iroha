<script lang="ts">
  import ThemeActivities from "$lib/themes/ThemeActivities.svelte";
  import type { ComponentProps } from "svelte";
  type Props = ComponentProps<typeof ThemeActivities>;
  let { ...props }: Omit<Props, "variant"> = $props();
</script>

<ThemeActivities variant="sound-map" {...props} />
