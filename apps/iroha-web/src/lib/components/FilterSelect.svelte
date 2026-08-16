<script lang="ts">
  import SharedSelectControl, {
    type SelectControlOption,
  } from "@iroha/shared/components/SelectControl.svelte";
  import { useTheme } from "$lib/themes/context.svelte";

  export type { SelectControlOption };

  let {
    label,
    value,
    options,
    ariaLabel,
    disabled = false,
    markerColor,
    onChange,
  }: {
    label: string;
    value: string;
    options: SelectControlOption[];
    ariaLabel?: string;
    disabled?: boolean;
    markerColor?: string;
    onChange: (value: string) => void;
  } = $props();

  const theme = useTheme();
</script>

<SharedSelectControl
  {label}
  {value}
  {options}
  {ariaLabel}
  {disabled}
  {markerColor}
  appearance={theme.definition().primitives.periodControl.appearance}
  {onChange}
/>
