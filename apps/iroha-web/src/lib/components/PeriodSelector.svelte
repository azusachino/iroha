<script lang="ts">
  import SharedPeriodSelector from "@iroha/shared/PeriodSelector.svelte";
  import type { PeriodOption } from "@iroha/shared/PeriodSelector.svelte";
  import { useTheme } from "$lib/themes/context.svelte";

  export type { PeriodOption };

  let {
    year,
    month,
    years,
    months,
    monthDisabled = false,
    showAllYears = true,
    surface = "panel",
    onYear,
    onMonth,
  }: {
    year: string;
    month: string;
    years: (PeriodOption | string)[];
    months: PeriodOption[];
    monthDisabled?: boolean;
    showAllYears?: boolean;
    surface?: "panel" | "inline";
    onYear: (value: string) => void;
    onMonth: (value: string) => void;
  } = $props();

  const theme = useTheme();
</script>

<SharedPeriodSelector
  {year}
  {month}
  {years}
  {months}
  {monthDisabled}
  {showAllYears}
  {surface}
  appearance={theme.definition().primitives.periodControl.appearance}
  {onYear}
  {onMonth}
/>
