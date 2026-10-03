<script lang="ts">
  import ApprovedActivityDetail from "$lib/components/ApprovedActivityDetail.svelte";
  import ActivityDetail from "$lib/components/ActivityDetail.svelte";
  import type {
    Activity,
    ActivityDetail as ActivityDetailData,
    RouteFeatureCollection,
  } from "$lib/types";

  let {
    activity,
    detail,
    routes,
    backHref,
  }: {
    activity: Activity;
    detail: ActivityDetailData | undefined;
    routes: RouteFeatureCollection;
    backHref: string;
  } = $props();
</script>

{#if detail}
  <ApprovedActivityDetail {detail} {backHref} />
{:else}
  <ActivityDetail {activity} {routes} {backHref} />
{/if}
