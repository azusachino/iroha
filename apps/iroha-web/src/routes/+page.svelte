<script lang="ts">
  import { auth } from "$lib/auth.svelte";
  import TodayCockpit from "./TodayCockpit.svelte";
  import PublicShowcase from "$lib/components/PublicShowcase.svelte";
  import type { PageProps } from "./$types";

  let { data }: PageProps = $props();
</script>

{#if auth.status === "ready"}
  <TodayCockpit />
{:else}
  <PublicShowcase initialData={data} />
{/if}
