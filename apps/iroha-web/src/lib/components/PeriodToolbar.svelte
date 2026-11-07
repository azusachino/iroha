<script lang="ts">
  import type { Snippet } from "svelte";
  import SharedPeriodToolbar from "@iroha/shared/components/PeriodToolbar.svelte";
  import { useTheme } from "$lib/themes/context.svelte";

  let {
    title,
    eyebrow = "Period",
    ariaLabel = "Period controls",
    onRefresh,
    refreshing = false,
    children,
  }: {
    title?: string;
    eyebrow?: string;
    ariaLabel?: string;
    onRefresh?: () => void;
    refreshing?: boolean;
    children: Snippet;
  } = $props();

  const theme = useTheme();
</script>

<SharedPeriodToolbar
  {title}
  {eyebrow}
  {ariaLabel}
  {onRefresh}
  {refreshing}
  appearance={theme.definition().primitives.periodControl.appearance}
>
  {#snippet children()}
    {@render children()}
  {/snippet}
</SharedPeriodToolbar>
