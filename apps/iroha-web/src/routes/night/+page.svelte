<script lang="ts">
  import { goto } from "$app/navigation";
  import type { SleepSession } from "$lib/api";
  import SleepScopeSummary from "@iroha/shared/theme-ui/components/SleepScopeSummary.svelte";
  import PeriodSelector from "@iroha/shared/components/PeriodSelector.svelte";
  import PeriodToolbar from "@iroha/shared/components/PeriodToolbar.svelte";
  import RetryNotice from "@iroha/shared/theme-ui/components/RetryNotice.svelte";
  import { useTheme } from "@iroha/shared/theme-ui/context.svelte";
  import GrapherSleep from "@iroha/shared/theme-ui/grapher/Sleep.svelte";
  import { createNightState } from "./night-state.svelte";

  const theme = useTheme();
  const t = createNightState();
  let root = $state<HTMLElement>();
  let summaryPanel = $state<HTMLElement>();
  let paginationPanel = $state<HTMLElement>();
</script>

<svelte:head>
  <title>iroha · Night</title>
</svelte:head>

<section class="sleep-shell" bind:this={root}>
  <GrapherSleep
    theme={theme.language()}
    sessions={t.sessions}
    selected={t.selected}
    sessionsLoading={t.sessionsResource.loading}
    sessionsReady={t.sessionsResource.ready}
    sessionsError={t.sessionsResource.error}
    recordsScope={t.recordsScope}
    onRetrySessions={() => t.loadSessions()}
    averageAsleep={t.averageAsleep}
    averageEfficiency={t.averageEfficiency}
    sleepSummary={t.sleepSummary}
    rollupBuckets={t.rollupBuckets}
    rollupGranularity={t.rollupGranularity}
    rollupScope={t.sleepScope}
    onOpenDetail={(session: SleepSession) => void goto(`/night/${session.id}`)}
  >
    {#snippet children()}
      <PeriodToolbar
        title="Sleep history scope"
        ariaLabel="Sleep period"
        onRefresh={() => void t.hardRefresh()}
        refreshing={t.sessionsResource.loading || t.aggregatesResource.loading}
      >
        <PeriodSelector
          years={t.periodYears}
          months={t.periodMonths}
          year={t.selectedYear}
          month={t.selectedMonth}
          bounds={t.bounds}
          monthDisabled={!t.selectedYear}
          surface="inline"
          onYear={t.changeYear}
          onMonth={t.changeMonth}
        />
      </PeriodToolbar>
      {#if t.boundsResource.error}
        <RetryNotice
          message={`Could not load the sleep date range: ${t.boundsResource.error}. Your selected period is unchanged.`}
          retryLabel="Retry sleep date range"
          onRetry={t.loadBounds}
          focusTarget={root?.querySelector<HTMLElement>(
            '[aria-label="Filter by year"]',
          ) ?? undefined}
        />
      {:else if t.boundsResource.loading}<p role="status">
          Loading the sleep date range…
        </p>{/if}
      <section
        aria-label="Sleep totals read state"
        tabindex="-1"
        bind:this={summaryPanel}
      >
        {#each [{ key: "year", label: "yearly", resource: t.yearResource }, { key: "month", label: "monthly", resource: t.monthResource }, { key: "lifetime", label: "lifetime", resource: t.lifetimeResource }] as read}
          {#if read.resource.error}
            <RetryNotice
              message={`Could not load ${read.label} sleep totals: ${read.resource.error}`}
              retryLabel={`Retry ${read.label} sleep totals`}
              onRetry={() =>
                t.loadAggregates(read.key as "year" | "month" | "lifetime")}
              focusTarget={summaryPanel}
            />
          {/if}
        {/each}
        {#if t.aggregatesResource.loading}<p role="status">
            {t.aggregatesResource.ready
              ? "Updating sleep totals…"
              : "Loading sleep totals…"}
          </p>{/if}
        {#if t.sleepSummary}<p>Observed totals: {t.sleepScope}</p>{/if}
        <SleepScopeSummary
          summary={t.sleepSummary}
          scope={t.sleepScope}
          theme={theme.language()}
        />
        {#if t.availableSummary}
          <p>
            Available totals for another scope: {t.availableSummary.scope}
          </p>
          <SleepScopeSummary
            summary={t.availableSummary.summary}
            scope={t.availableSummary.scope}
            theme={theme.language()}
          />
        {/if}
      </section>
    {/snippet}
  </GrapherSleep>
  <section
    aria-label="Night pagination"
    tabindex="-1"
    bind:this={paginationPanel}
  >
    {#if t.loadMoreError}
      <RetryNotice
        message={`Could not load the next sleep page: ${t.loadMoreError}. Observed sessions and cursor are retained.`}
        retryLabel="Retry more nights"
        onRetry={() => t.loadSessions(true)}
        focusTarget={paginationPanel}
      />
    {/if}
    {#if t.hasMore}
      <div
        bind:this={t.loadMoreSentinel}
        class="theme-load-more"
        aria-live="polite"
      >
        {#if t.loadingMore}<span>Loading more nights…</span>{:else}<button
            type="button"
            onclick={() => t.loadSessions(true)}
          >
            Load more nights
          </button>{/if}
      </div>
    {/if}
  </section>
</section>

<style>
  .sleep-shell {
    display: grid;
    gap: 1rem;
  }
  .theme-load-more {
    display: grid;
    min-height: 2rem;
    place-items: center;
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .theme-load-more button {
    padding: 0.55rem 0.8rem;
    border: 1px solid var(--border);
    border-radius: 8px;
    background: var(--surface-2);
    color: var(--text);
    font: inherit;
    cursor: pointer;
  }
  .theme-load-more button:hover {
    border-color: var(--accent);
  }
</style>
