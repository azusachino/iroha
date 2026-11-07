<script lang="ts">
  import ApprovedActivityDetail from "$lib/public/components/ApprovedActivityDetail.svelte";
  import ActivityDetail from "$lib/public/components/ActivityDetail.svelte";
  import type {
    Activity,
    ActivityDetail as ActivityDetailData,
    RouteFeatureCollection,
  } from "$lib/public/types";

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
