<script lang="ts">
  import { invalidateAll } from "$app/navigation";
  import RetryNotice from "@iroha/shared/theme-ui/components/RetryNotice.svelte";
</script>

<svelte:head><title>Public activity data unavailable</title></svelte:head>

<h1>Public activity data unavailable</h1>
<RetryNotice
  message="Public activity data is temporarily unavailable. No activity totals can be shown."
  onRetry={() => {
    void invalidateAll();
  }}
/>
