<script lang="ts">
  import ThemeMediaDetail from "$lib/themes/ThemeMediaDetail.svelte";
  import type { ComponentProps } from "svelte";
  type Props = ComponentProps<typeof ThemeMediaDetail>;
  let { ...props }: Omit<Props, "variant"> = $props();
</script>

<ThemeMediaDetail variant="phenology" {...props} />
