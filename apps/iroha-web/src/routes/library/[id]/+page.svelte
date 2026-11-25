<script lang="ts">
  import { page } from "$app/state";
  import { getMedia, type MediaDetail } from "$lib/api";
  import { mediaWorkTotal, progressPercent } from "$lib/format";
  import { useTheme } from "@iroha/shared/theme-ui/context.svelte";
  import GrapherMediaDetail from "@iroha/shared/theme-ui/grapher/MediaDetail.svelte";
  import { createAsyncResource } from "$lib/asyncResource.svelte";
  import RetryNotice from "@iroha/shared/theme-ui/components/RetryNotice.svelte";
  import { untrack } from "svelte";

  const resource = createAsyncResource<{ id: string; value: MediaDetail }>();
  const id = $derived(page.params.id ?? "");
  const detail = $derived(
    resource.data?.id === id ? resource.data.value : null,
  );
  const loading = $derived(resource.loading);
  const error = $derived(resource.error);
  let detailPanel = $state<HTMLElement>();
  const theme = useTheme();

  // Re-fetch whenever the route param changes. SvelteKit reuses this component
  // across /library/[id] navigations, so onMount would fire only once and
  // clicking a related title would change the URL without reloading the page.
  $effect(() => {
    const selected = id;
    untrack(() => void load(selected));
  });

  async function load(id: string) {
    await resource.run(async () => {
      if (!id) throw new Error("Missing media id");
      return { id, value: await getMedia(id) };
    });
  }

  // The work's own episode/chapter count, independent of this user's
  // progress row -- known even for an ongoing series where progress.total
  // never is.
  function workTotal(): number | undefined {
    return mediaWorkTotal(
      detail?.item.media_type,
      detail?.item.episode_count,
      detail?.item.chapter_count,
    );
  }

  function progressValue(): number {
    const status = detail?.progress?.status ?? detail?.item.status;
    const position = detail?.progress?.position ?? detail?.item.position;
    const total = detail?.progress?.total ?? detail?.item.total;
    const percent =
      detail?.progress?.progress_percent ?? detail?.item.progress_percent;
    return progressPercent(status, position, total, percent, workTotal());
  }

  // Whether a total is known or can be inferred (a completed item's
  // position stands in for its total, or the work's own episode/chapter
  // count is known) -- gates the percent readout and bar, which need an
  // actual total to mean anything, on the same basis
  // progressValue() already uses internally.
  function hasKnownTotal(): boolean {
    if (!detail?.progress) return false;
    if (detail.progress.total != null) return true;
    if (workTotal() != null) return true;
    return (
      detail.progress.status === "completed" && detail.progress.position != null
    );
  }
</script>

<svelte:head>
  <title>iroha · Library · {detail?.item.title ?? "Library"}</title>
</svelte:head>

<section
  class="detail-shell"
  aria-label="Library item read state"
  tabindex="-1"
  bind:this={detailPanel}
>
  <p><a class="back-link" href="/library">← Back to Library</a></p>
  {#if error}<RetryNotice
      message={`Could not load library item: ${error}`}
      retryLabel="Retry library item"
      onRetry={() => load(id)}
      focusTarget={detailPanel}
    />{/if}
  {#if loading && detail}<p role="status">Updating library item…</p>{/if}
  {#if detail}
    <GrapherMediaDetail
      {detail}
      progress={progressValue()}
      hasKnownTotal={hasKnownTotal()}
      theme={theme.language()}
    />
  {:else if loading}
    <p class="muted">Loading item…</p>
  {:else if error}
    <p class="error">Item unavailable. The Library remains accessible.</p>
  {/if}
</section>

<style>
  .detail-shell {
    display: grid;
    gap: 1.25rem;
  }
  .back-link {
    color: var(--text-muted);
    font-size: 0.85rem;
  }
  .error {
    color: var(--danger);
  }
</style>
