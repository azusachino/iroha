<script lang="ts">
  import PublicShowcase from "$lib/components/PublicShowcase.svelte";
  import type { PageProps } from "./$types";

  let { data }: PageProps = $props();
</script>

<PublicShowcase initialData={data} />
