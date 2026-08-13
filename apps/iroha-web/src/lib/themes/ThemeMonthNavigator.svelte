<script lang="ts">
  import MonthNavigator from "@iroha/shared/MonthNavigator.svelte";
  import { useTheme } from "$lib/themes/context.svelte";

  let {
    month,
    onMonth,
    disabled = false,
  }: {
    month: string;
    onMonth: (value: string) => void;
    disabled?: boolean;
  } = $props();

  const theme = useTheme();
</script>

<MonthNavigator
  {month}
  {onMonth}
  {disabled}
  appearance={theme.definition().primitives.periodControl.appearance}
/>
