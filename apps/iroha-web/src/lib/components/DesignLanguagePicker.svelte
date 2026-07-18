<script lang="ts">
  import { onMount } from "svelte";
  import {
    DESIGN_LANGUAGES,
    getDesignLanguage,
    setDesignLanguage,
    type DesignLanguage,
  } from "$lib/design-language";

  let language = $state<DesignLanguage>("field-journal");

  onMount(() => {
    language = getDesignLanguage();
  });

  function onChange(event: Event) {
    const value = (event.currentTarget as HTMLSelectElement).value;
    const next = DESIGN_LANGUAGES.find((item) => item.id === value);
    if (!next) return;
    language = next.id;
    setDesignLanguage(language);
  }
</script>

<label class="language-picker">
  <span>Design language</span>
  <select aria-label="Design language" value={language} onchange={onChange}>
    {#each DESIGN_LANGUAGES as item}
      <option value={item.id}>{item.label}</option>
    {/each}
  </select>
</label>
