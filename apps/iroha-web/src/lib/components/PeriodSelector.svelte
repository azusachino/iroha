<script lang="ts">
  import SharedPeriodSelector from "@iroha/shared/PeriodSelector.svelte";
  import type { PeriodOption } from "@iroha/shared/PeriodSelector.svelte";
  import { IROHA_TIMEZONE } from "$lib/config";
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
    timezone = IROHA_TIMEZONE,
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
    timezone?: string;
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
  {timezone}
  appearance={theme.definition().primitives.periodControl.appearance}
  {onYear}
  {onMonth}
/>
