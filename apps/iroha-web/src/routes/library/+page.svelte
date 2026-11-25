<script lang="ts">
  import GrapherMedia from "@iroha/shared/theme-ui/grapher/Media.svelte";
  import { createLibraryState } from "./library-state.svelte";

  const l = createLibraryState();
</script>

<svelte:head>
  <title>iroha · Library</title>
</svelte:head>

<section class="media-shell">
  <GrapherMedia
    theme="grapher"
    items={l.items}
    aggregates={l.aggregatesForView}
    aggregatesLoading={l.aggregatesResource.loading}
    aggregatesReady={l.aggregatesResource.ready}
    aggregatesError={l.aggregatesResource.error}
    aggregatesScope={l.aggregatesScope}
    recordsLoading={l.libraryResource.loading}
    recordsReady={l.libraryResource.ready}
    recordsError={l.libraryResource.error}
    recordsScope={l.recordsScope}
    onRetryAggregates={l.loadAggregates}
    onRetryRecords={l.loadRecords}
    family={l.family}
    status={l.status}
    completedYear={l.completedYear}
    yearOptions={l.yearOptions}
    typeFamilies={l.typeFamilies}
    completions={l.completions}
    scores={l.scores}
    currentCompletedCount={l.aggregatesForView?.totals
      .current_completed_count ?? null}
    activeCount={l.activeCount}
    activeScope={l.activeScope}
    onFamily={l.selectFamily}
    onStatus={l.selectStatus}
    onYear={l.selectYear}
    onLoadMore={l.loadMore}
    hasMore={l.hasMore}
    loadingMore={l.loadingMore}
    loadMoreError={l.loadMoreError}
  />
</section>

<style>
  .media-shell {
    display: grid;
    gap: 1.75rem;
  }
</style>
