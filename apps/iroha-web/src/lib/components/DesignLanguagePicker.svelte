<script lang="ts">
  import { DESIGN_LANGUAGES } from "$lib/design-language";
  import { useTheme } from "$lib/themes/context.svelte";

  const theme = useTheme();

  function onChange(event: Event) {
    const value = (event.currentTarget as HTMLSelectElement).value;
    const next = DESIGN_LANGUAGES.find((item) => item.identity.id === value);
    if (!next) return;
    theme.select(next.identity.id);
  }

  function compactLabel(label: string): string {
    return label.replace(/^Iroha\s+/, "");
  }
</script>

<label class="language-picker">
  <span>Design language</span>
  <select
    aria-label="Design language"
    title={theme.definition().identity.label}
    value={theme.language()}
    onchange={onChange}
  >
    {#each DESIGN_LANGUAGES as item}
      <option value={item.identity.id}>
        {compactLabel(item.identity.label)}{item.implementation === "preview"
          ? " · preview"
          : ""}
      </option>
    {/each}
  </select>
</label>
