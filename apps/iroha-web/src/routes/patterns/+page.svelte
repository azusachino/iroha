<script lang="ts">
  import PeriodSelector from "@iroha/shared/components/PeriodSelector.svelte";
  import PeriodToolbar from "@iroha/shared/components/PeriodToolbar.svelte";
  import LoadingBoundary from "$lib/components/LoadingBoundary.svelte";
  import GrapherDaily from "@iroha/shared/theme-ui/grapher/Daily.svelte";
  import RetryNotice from "@iroha/shared/theme-ui/components/RetryNotice.svelte";
  import { createPatternsState } from "./patterns-state.svelte";

  type Gran = "day" | "month" | "year";
  const t = createPatternsState();
  const themeProps = $derived({
    theme: "grapher" as const,
    seriesAvailable: t.seriesAvailable,
    chrono: t.chrono,
    gran: t.gran,
    onGran: (value: Gran) => void t.changeGranularity(value),
    onDrillIndex: t.drillIntoIndex,
    onDrillPeriod: t.drillIntoPeriod,
    ringData: t.ringData,
    latestRingDay: t.latestRingDay,
    ringGoalSourceDate: t.ringGoalSourceDate,
  });
  let seriesTarget = $state<HTMLElement>();
  let latestTarget = $state<HTMLElement>();
  let boundsTarget = $state<HTMLElement>();
</script>

<svelte:head>
  <title>iroha · Patterns</title>
</svelte:head>

<section class="daily">
  <GrapherDaily {...themeProps} section="controls">
    {#snippet children()}
      <section
        aria-label="Pattern scope selection"
        tabindex="-1"
        bind:this={boundsTarget}
      >
        <PeriodToolbar
          title="Daily pattern scope"
          ariaLabel="Daily period"
          onRefresh={() => void t.hardRefresh()}
          refreshing={t.seriesResource.loading || t.latestDayResource.loading}
        >
          <PeriodSelector
            years={t.periodYears}
            months={t.periodMonths}
            year={t.gran === "year" ? t.selectedYear : t.activeYear}
            month={t.gran === "day" ? t.activeMonth : t.selectedMonth}
            bounds={t.dailyBounds}
            showAllYears={t.gran === "year"}
            surface="inline"
            onYear={t.selectYear}
            onMonth={t.selectMonth}
          />
        </PeriodToolbar>
        <p>Selected pattern scope: {t.periodLabel}</p>
        {#if t.boundsResource.error}
          <RetryNotice
            message={`Pattern range unavailable. ${t.boundsResource.error}`}
            retryLabel="Retry pattern range"
            onRetry={t.loadBounds}
            focusTarget={boundsTarget}
          />
          <p>Calendar choices are not an observed data inventory.</p>
        {/if}
      </section>
    {/snippet}
  </GrapherDaily>
  <section
    aria-label="Latest pattern reading"
    tabindex="-1"
    bind:this={latestTarget}
  >
    {#if t.latestDayResource.error}
      <RetryNotice
        message={`Latest reading unavailable. ${t.latestDayResource.error}`}
        retryLabel="Retry latest reading"
        onRetry={t.loadLatestDay}
        focusTarget={latestTarget}
      />
    {/if}
    <LoadingBoundary
      resource={t.latestDayResource}
      preserveLayout
      label="Loading latest reading…"
    >
      {#if t.latestDayResource.ready}
        <GrapherDaily {...themeProps} section="rings" />
      {/if}
    </LoadingBoundary>
  </section>
  <section aria-label="Pattern periods" tabindex="-1" bind:this={seriesTarget}>
    {#if t.seriesResource.error}
      <RetryNotice
        message={`Pattern periods unavailable. ${t.seriesResource.error}`}
        retryLabel={t.seriesRetryLabel}
        onRetry={t.retrySeries}
        focusTarget={seriesTarget}
      />
    {/if}
    <LoadingBoundary
      resource={t.seriesResource}
      preserveLayout
      label="Loading time-series data…"
    >
      {#if t.seriesAvailable}
        <p>Observed pattern scope: {t.observedScope}</p>
        <GrapherDaily {...themeProps} section="series" />
      {/if}
    </LoadingBoundary>
  </section>
</section>

<style>
  .daily {
    display: grid;
    gap: 1.25rem;
  }
</style>
