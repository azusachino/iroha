<script lang="ts">
  import ThemeShare from "$lib/themes/ThemeShare.svelte";
  import type { ComponentProps } from "svelte";
  type Props = ComponentProps<typeof ThemeShare>;
  let { ...props }: Omit<Props, "variant"> = $props();
</script>

<ThemeShare variant="atlas" {...props} />
