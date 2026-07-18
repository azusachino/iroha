<script lang="ts">
  import { DESIGN_LANGUAGES } from "$lib/design-language";
  import { useTheme } from "$lib/themes/context.svelte";

  const theme = useTheme();

  function onChange(event: Event) {
    const value = (event.currentTarget as HTMLSelectElement).value;
    const next = DESIGN_LANGUAGES.find((item) => item.id === value);
    if (!next) return;
    theme.select(next.id);
  }
</script>

<label class="language-picker">
  <span>Design language</span>
  <select
    aria-label="Design language"
    value={theme.language()}
    onchange={onChange}
  >
    {#each DESIGN_LANGUAGES as item}
      <option value={item.id}>{item.label}</option>
    {/each}
  </select>
</label>
