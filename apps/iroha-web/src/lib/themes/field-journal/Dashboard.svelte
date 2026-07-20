<script lang="ts">
  import ThemeDashboard from "$lib/themes/ThemeDashboard.svelte";
  import type { ComponentProps } from "svelte";
  type Props = ComponentProps<typeof ThemeDashboard>;
  let { ...props }: Omit<Props, "variant"> = $props();
</script>

<ThemeDashboard variant="field-journal" {...props} />
