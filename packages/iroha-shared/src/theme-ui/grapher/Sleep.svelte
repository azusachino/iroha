<script lang="ts">
  import type { SleepThemeProps } from "../../view-contracts/sleep-view";
  import EmptyNote from "../../components/EmptyNote.svelte";
  import FocalBand, {
    type FocalPrimaryMetric,
    type FocalSecondaryItem,
  } from "../../components/FocalBand.svelte";
  import MetricTable from "../../components/MetricTable.svelte";
  import PanelFrame from "../../components/PanelFrame.svelte";
  import RouteHeader from "../../components/RouteHeader.svelte";
  import BarChart from "../components/BarChart.svelte";
  import RetryNotice from "../components/RetryNotice.svelte";
  import SleepAggregateChart from "../components/SleepAggregateChart.svelte";
  import { formatDateOnly, formatDuration } from "../../format/format";

  let {
    sessions,
    selected,
    sessionsLoading = false,
    sessionsReady = false,
    sessionsError = null,
    recordsScope = "",
    onRetrySessions,
    averageAsleep,
    averageEfficiency,
    onOpenDetail,
    sleepSummary = null,
    rollupBuckets = [],
    rollupGranularity = "year",
    rollupScope = "",
    theme,
    children,
  }: SleepThemeProps = $props();

  const NIGHT_CHART_MIN_POINTS = 3;
  const chartSessions = $derived(sessions.toReversed());
  const summaryHeadingId = $props.id();
  let recordsPanel = $state<HTMLElement>();

  const hasRecordedNights = $derived(
    (sleepSummary?.session_count ?? 0) > 0 ||
      averageAsleep != null ||
      sessions.length > 0,
  );

  const primaryMetric = $derived<FocalPrimaryMetric>({
    label: "Average main sleep",
    value: averageAsleep == null ? "—" : formatDuration(averageAsleep),
    detail:
      sleepSummary && sleepSummary.main_sleep_count > 0
        ? `${sleepSummary.main_sleep_count.toLocaleString()} main ${sleepSummary.main_sleep_count === 1 ? "sleep" : "sleeps"}`
        : undefined,
  });

  const stageSplits = $derived.by(() => {
    if (!sleepSummary) return [];
    const stages = [
      { label: "Deep", seconds: sleepSummary.deep_s },
      { label: "REM", seconds: sleepSummary.rem_s },
      { label: "Core", seconds: sleepSummary.core_s },
    ].filter((stage) => stage.seconds > 0);
    return stages.map(
      (stage) => `${stage.label} ${formatDuration(stage.seconds)}`,
    );
  });

  const secondaryItems = $derived.by<FocalSecondaryItem[]>(() => {
    const items: FocalSecondaryItem[] = [];

    if (averageEfficiency != null) {
      items.push({
        label: "Average efficiency",
        value: `${Math.round(averageEfficiency * 100)}%`,
      });
    }

    if (sleepSummary && sleepSummary.session_count > 0) {
      items.push({
        label: "Recorded sessions",
        value: sleepSummary.session_count.toLocaleString(),
        detail:
          sleepSummary.nap_count > 0
            ? `${sleepSummary.main_sleep_count.toLocaleString()} main · ${sleepSummary.nap_count.toLocaleString()} ${sleepSummary.nap_count === 1 ? "nap" : "naps"}`
            : undefined,
      });
    }

    if (stageSplits.length > 0) {
      items.push({
        label: "Stage splits",
        value: stageSplits[0],
        detail: stageSplits.slice(1).join(" · ") || undefined,
      });
    }

    return items;
  });

  const emptyNotes = $derived.by<string[]>(() => {
    const notes: string[] = [];

    if (averageAsleep == null) {
      notes.push("Main sleep duration unavailable");
    }
    if (averageEfficiency == null) {
      notes.push("Efficiency unavailable");
    }
    if (sleepSummary && sleepSummary.session_count === 0) {
      notes.push("No sleep sessions recorded");
    } else if (sleepSummary && stageSplits.length === 0) {
      notes.push("No sleep stage splits recorded");
    }

    return notes;
  });

  const shortSeriesSummary = $derived.by(() => {
    if (chartSessions.length === 1) {
      const only = chartSessions[0];
      return `Single night recorded: ${formatDateOnly(only.wake_date)} · ${formatDuration(only.asleep_s)} asleep (${Math.round(only.efficiency * 100)}% efficiency).`;
    }
    if (chartSessions.length === 2) {
      const [first, second] = chartSessions;
      return `Two nights recorded: ${formatDateOnly(first.wake_date)} (${formatDuration(first.asleep_s)}) and ${formatDateOnly(second.wake_date)} (${formatDuration(second.asleep_s)}).`;
    }
    return "";
  });
</script>

<section class="grapher-sleep" aria-label="Night">
  <RouteHeader
    title="Night"
    context="Recovery and sleep sessions"
  />

  {#if hasRecordedNights}
    <FocalBand
      heading="Sleep summary"
      controls={children}
      primary={primaryMetric}
      secondary={secondaryItems}
      empty={emptyNotes.length ? emptyNotes.join(" · ") : undefined}
    />
  {:else}
    <section class="focal-band" aria-labelledby={summaryHeadingId}>
      <div class="focal-header">
        <h2 id={summaryHeadingId} class="focal-heading">Sleep summary</h2>
        {#if children}
          <div class="focal-controls">{@render children()}</div>
        {/if}
      </div>
      <EmptyNote
        text={emptyNotes.length
          ? emptyNotes.join(" · ")
          : "No sleep sessions recorded"}
      />
    </section>
  {/if}

  {#if rollupBuckets.length}
    <SleepAggregateChart
      buckets={rollupBuckets}
      granularity={rollupGranularity}
      scope={rollupScope}
      {theme}
    />
  {:else if sessions.length}
    <PanelFrame title="Asleep time by night">
      {#if chartSessions.length >= NIGHT_CHART_MIN_POINTS}
        <BarChart
          categories={chartSessions.map((session) =>
            formatDateOnly(session.wake_date),
          )}
          primary={{
            name: "Asleep",
            values: chartSessions.map((session) => session.asleep_s),
            colors: chartSessions.map((session) =>
              session.is_main_sleep ? "var(--accent)" : "var(--accent-2)",
            ),
            formatter: (value) => formatDuration(value),
          }}
          onBarClick={(index) => {
            const session = chartSessions[index];
            if (session) onOpenDetail(session);
          }}
        />
      {:else}
        <EmptyNote text={shortSeriesSummary} />
      {/if}
    </PanelFrame>
  {/if}

  <section
    class="session-table-region"
    aria-label="Sleep session read state"
    tabindex="-1"
    bind:this={recordsPanel}
  >
    <PanelFrame title="Night by night">
      {#snippet meta()}
        {#if recordsScope}<span>{recordsScope}</span>{/if}
      {/snippet}

      {#if sessionsError && onRetrySessions}
        <RetryNotice
          message={`Could not load sleep sessions: ${sessionsError}`}
          retryLabel="Retry sleep sessions"
          onRetry={onRetrySessions}
          focusTarget={recordsPanel}
        />
      {/if}
      {#if sessionsLoading}
        <p class="status-note" role="status">
          {sessionsReady
            ? "Updating sleep sessions…"
            : "Loading sleep sessions…"}
        </p>
      {/if}

      {#if sessionsReady && !sessionsLoading && !sessionsError && !sessions.length}
        <EmptyNote text="No sleep sessions in this observed scope." />
      {:else if sessions.length}
        <MetricTable
          ariaLabel="Sleep session records"
          scrollLabel="Night exact sleep records"
          caption="Sleep session records"
          minWidth="38rem"
          nowrap
          numericAlign
          dense
        >
          <thead>
            <tr>
              <th>Date</th>
              <th>Asleep</th>
              <th>In bed</th>
              <th>Efficiency</th>
              <th>Type</th>
            </tr>
          </thead>
          <tbody>
            {#each sessions as session (session.id)}
              <tr
                class="session-row"
                role="link"
                tabindex="0"
                onclick={() => onOpenDetail(session)}
                onkeydown={(event) => {
                  if (event.key === "Enter" || event.key === " ") {
                    event.preventDefault();
                    onOpenDetail(session);
                  }
                }}
              >
                <td>{formatDateOnly(session.wake_date)}</td>
                <td>{formatDuration(session.asleep_s)}</td>
                <td>{formatDuration(session.time_in_bed_s)}</td>
                <td>{Math.round(session.efficiency * 100)}%</td>
                <td>{session.is_main_sleep ? "Main sleep" : "Nap"}</td>
              </tr>
            {/each}
          </tbody>
        </MetricTable>
      {/if}
    </PanelFrame>
  </section>

  {#if selected}
    <aside class="selected-note">
      <p class="kicker">Selected session</p>
      <strong>
        {formatDateOnly(selected.wake_date)} · {formatDuration(
          selected.asleep_s,
        )} asleep
      </strong>
      <span>Open the standard detail view for stage-level inspection.</span>
    </aside>
  {/if}
</section>

<style>
  .grapher-sleep {
    --accent: var(--mark-violet);
    display: grid;
    gap: var(--gap-section);
    min-width: 0;
  }

  .grapher-sleep > * {
    min-width: 0;
  }

  .focal-band {
    display: grid;
    gap: var(--space-3);
    margin-bottom: var(--gap-section);
  }

  .focal-header {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
  }

  .focal-heading {
    margin: 0;
    color: var(--text-muted);
    font-size: var(--type-title-small);
    font-weight: 700;
    line-height: 1.2;
  }

  .focal-controls {
    margin-left: auto;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
  }

  .session-table-region {
    min-width: 0;
  }

  .status-note {
    margin: 0;
    color: var(--text-muted);
    font-size: var(--type-label);
  }

  .grapher-sleep
    :global(
      .focal-controls
        section[aria-label="Sleep totals read state"]
        > p:not([role="status"])
    ),
  .grapher-sleep
    :global(.focal-controls .sleep-scope-summary .scope-label) {
    display: none;
  }

  .session-row {
    cursor: pointer;
    transition: background-color var(--motion-micro);
  }

  .session-row:hover {
    background: color-mix(in srgb, var(--accent) 10%, var(--surface));
  }

  .session-row:focus-visible {
    outline: 2px solid var(--color-focus);
    outline-offset: -2px;
  }

  .selected-note {
    display: grid;
    gap: var(--space-1);
    padding: var(--pad-card);
    border-left: 3px solid var(--accent);
    border-radius: var(--radius);
    background: var(--surface-2);
  }

  .selected-note strong {
    font-size: var(--type-body);
  }

  .selected-note span {
    color: var(--text-muted);
    font-size: var(--type-caption);
  }
</style>
