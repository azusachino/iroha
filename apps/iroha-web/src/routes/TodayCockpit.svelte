<script lang="ts">
  import { goto } from "$app/navigation";
  import DayPicker from "$lib/components/DayPicker.svelte";
  import GrapherToday from "@iroha/shared/theme-ui/grapher/Today.svelte";
  import EmptyState from "@iroha/shared/theme-ui/components/EmptyState.svelte";
  import TodaySkeleton from "$lib/components/TodaySkeleton.svelte";
  import RetryNotice from "@iroha/shared/theme-ui/components/RetryNotice.svelte";
  import { createTodayState } from "./today-state.svelte";
  import { RefreshCw } from "@lucide/svelte";

  const t = createTodayState();
  let briefingPanel = $state<HTMLElement>();
  let calendarPanel = $state<HTMLElement>();
</script>

<svelte:head>
  <title>iroha · Today</title>
</svelte:head>

<svelte:window onkeydown={t.onKey} />

<section class="cockpit">
  {#if !t.briefing || !t.dayHasData}
    <h1 class="visually-hidden">Today, {t.dayLabel}</h1>
  {/if}
  <div
    class="scrubber tile glow"
    role="region"
    aria-label="Day calendar"
    tabindex="-1"
    bind:this={calendarPanel}
  >
    {#if t.calendarResource.error}<RetryNotice
        message={`Day calendar unavailable: ${t.calendarResource.error}. Date selection remains available.`}
        retryLabel="Retry day calendar"
        onRetry={t.retryCalendar}
        focusTarget={calendarPanel}
      />{/if}
    <button class="nav" aria-label="Previous day" onclick={() => t.shift(-1)}
      >‹</button
    >
    <button
      class="scrub-center"
      aria-haspopup="dialog"
      aria-expanded={t.pickerOpen}
      onclick={() => (t.pickerOpen = !t.pickerOpen)}
    >
      <span class="day-main">{t.dayLabel}</span>
    </button>
    <button
      class="nav"
      aria-label="Next day"
      disabled={!t.canMoveNext}
      onclick={() => t.shift(1)}>›</button
    >
    <button
      type="button"
      class="scrub-refresh"
      aria-label="Refresh today"
      onclick={() => void t.hardRefresh()}
      disabled={t.loading}
    >
      <RefreshCw size={13} class={t.loading ? "spin" : ""} />
    </button>
    {#if t.pickerOpen}
      <button
        class="pk-backdrop"
        aria-label="Close picker"
        onclick={() => (t.pickerOpen = false)}
      ></button>
      <div class="pk-pop tile" role="dialog" aria-label="Pick a day">
        <DayPicker
          value={t.day}
          days={t.daysSet}
          max={t.today}
          onselect={(d) => {
            t.day = d;
            t.pickerOpen = false;
          }}
        />
        <button
          class="today-link"
          type="button"
          onclick={() => {
            t.day = t.today;
            t.pickerOpen = false;
          }}
        >
          Return to today
        </button>
      </div>
    {/if}
  </div>
  <section
    aria-label="Daily briefing read state"
    tabindex="-1"
    bind:this={briefingPanel}
  >
    {#if t.error || t.unavailableSections.length}<RetryNotice
        message={t.error
          ? `Daily briefing unavailable: ${t.error}`
          : `Unavailable briefing sections: ${t.unavailableSections.join(", ")}. Absence of records is not established for these sections.`}
        retryLabel="Retry daily briefing"
        onRetry={t.retryBriefing}
        focusTarget={briefingPanel}
      />{/if}
  </section>

  {#if !t.briefing && t.loading}
    <TodaySkeleton label={`Loading ${t.dayLabel}…`} />
  {:else if !t.briefing && t.error}
    <p class="error status" role="alert">Could not load data: {t.error}</p>
  {:else if t.briefing}
    <div class="briefing-surface" aria-busy={t.loading}>
      {#if t.loading}
        <p class="briefing-update" role="status" aria-live="polite">
          Updating {t.dayLabel}…
        </p>
      {/if}
      {#if t.error}
        <p class="error update-error" role="alert">
          Could not update {t.dayLabel}; showing {t.dataDayLabel}.
        </p>
      {/if}
      <div class="briefing-content">
        {#if !t.dayHasData && t.unavailableSections.length}
          <p role="status">
            Some sections for {t.dataDayLabel} are unavailable. Available sections
            contain no records.
          </p>
        {:else if !t.dayHasData}
          <EmptyState
            eyebrow="Quiet day"
            title={`No records for ${t.dataDayLabel}.`}
            description="The canonical cockpit has no imported movement, recovery, health, media, or task records for this date."
            actionHref={!t.canJumpToLatestDay && t.dataDay !== t.today
              ? "/"
              : undefined}
            actionLabel={t.canJumpToLatestDay
              ? "Jump to latest recorded day"
              : "Return to today"}
            actionOnclick={t.canJumpToLatestDay ? t.jumpToLatestDay : undefined}
          />
        {:else}
          <GrapherToday
            theme="grapher"
            dayLabel={t.dataDayLabel}
            day={t.dataDay}
            unavailableSections={t.unavailableSections}
            dRow={t.dRow}
            mainNight={t.mainNight}
            acts={t.acts}
            mediaEvents={t.mediaEvents}
            mediaUpdates={t.mediaUpdates}
            onOpenActivity={(id: string) => void goto(`/motion/${id}`)}
            onOpenMedia={(id: string) => void goto(`/library/${id}`)}
          />
        {/if}
      </div>
    </div>
  {/if}
</section>

<style>
  .cockpit {
    display: grid;
    gap: 1.25rem;
  }
  .status {
    padding: 2rem 0;
  }
  .error {
    color: var(--danger);
  }
  .briefing-surface {
    position: relative;
    min-width: 0;
  }
  .briefing-content {
    min-width: 0;
  }
  .briefing-update,
  .update-error {
    margin: 0 0 0.65rem;
    font-size: 0.75rem;
  }
  .briefing-update {
    /* Keep the live announcement out of layout so date scrubbing cannot
       move the whole data surface while the next snapshot is loading. */
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    overflow: hidden;
    clip: rect(0 0 0 0);
    white-space: nowrap;
    border: 0;
    margin: -1px;
  }
  .update-error {
    color: var(--danger);
  }

  /* Date scrubber — the spine control, gets the neon chrome glow. */
  .scrubber {
    position: relative;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.75rem;
    min-width: 0;
    padding: 0.85rem 1rem;
  }
  .scrubber > :global(.retry-notice) {
    order: 1;
    flex: 0 0 100%;
    min-width: 0;
    box-sizing: border-box;
  }
  .glow {
    border: 1px solid color-mix(in srgb, var(--accent) 34%, var(--border));
    box-shadow: var(--accent-glow), var(--tile-shadow);
  }
  .scrub-center {
    flex: 1;
    min-width: max-content;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.1rem;
    appearance: none;
    border: 0;
    background: transparent;
    cursor: pointer;
    border-radius: var(--radius);
    padding: 0.25rem;
  }
  .scrub-center:hover {
    background: color-mix(in srgb, var(--accent) 8%, transparent);
  }
  .day-main {
    color: var(--accent);
    font-size: 1.1rem;
    white-space: nowrap;
    font-weight: 700;
    text-shadow: 0 0 16px color-mix(in srgb, var(--accent) 28%, transparent);
  }
  .pk-backdrop {
    position: fixed;
    inset: 0;
    z-index: 20;
    border: 0;
    background: transparent;
    cursor: default;
  }
  .pk-pop {
    position: absolute;
    z-index: 21;
    top: calc(100% + 0.5rem);
    left: 50%;
    transform: translateX(-50%);
    box-shadow: var(--accent-glow), var(--tile-shadow);
    border: 1px solid color-mix(in srgb, var(--accent) 30%, var(--border));
  }
  .nav {
    flex: 0 0 2.2rem;
    appearance: none;
    border: 1px solid var(--border);
    background: var(--surface);
    color: var(--text);
    width: 2.2rem;
    height: 2.2rem;
    border-radius: 50%;
    font-size: 1.2rem;
    line-height: 1;
    cursor: pointer;
  }
  .nav:hover {
    border-color: var(--accent);
    color: var(--accent);
  }
  .nav:disabled {
    cursor: default;
    opacity: 0.35;
  }
  .scrub-refresh {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 2.2rem;
    height: 2.2rem;
    border-radius: 50%;
    border: 1px solid var(--border);
    background: var(--surface);
    color: var(--text-muted);
    cursor: pointer;
    transition:
      color 0.15s ease,
      border-color 0.15s ease;
  }
  .scrub-refresh:hover:not(:disabled) {
    color: var(--accent);
    border-color: var(--accent);
  }
  .scrub-refresh:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
  :global(.scrub-refresh .spin) {
    animation: scrub-spin 0.8s linear infinite;
  }
  @keyframes scrub-spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }
  .today-link {
    display: block;
    width: calc(100% - 1.5rem);
    margin: 0 auto 0.7rem;
    padding: 0.45rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface-2);
    color: var(--text-muted);
    font: inherit;
    font-size: 0.75rem;
    cursor: pointer;
  }
  .today-link:hover {
    border-color: var(--accent);
    color: var(--text);
  }

  @media (max-width: 640px) {
    .scrubber {
      flex-wrap: wrap;
      gap: 0.45rem;
    }

    .scrub-center {
      min-width: 0;
    }
  }
</style>
