<script lang="ts">
  import ThemeActivityDetail from "$lib/themes/ThemeActivityDetail.svelte";
  import type { ComponentProps } from "svelte";
  type Props = ComponentProps<typeof ThemeActivityDetail>;
  let { ...props }: Omit<Props, "variant"> = $props();
</script>

<ThemeActivityDetail variant="atlas" {...props} />
