<script lang="ts">
  import { goto } from "$app/navigation";
  import type { SleepSession } from "$lib/api";
  import StatTile from "@iroha/shared/components/StatTile.svelte";
  import SleepScopeSummary from "@iroha/shared/theme-ui/components/SleepScopeSummary.svelte";
  import SleepArchitectureChart from "@iroha/shared/theme-ui/components/SleepArchitectureChart.svelte";
  import PeriodSelector from "$lib/components/PeriodSelector.svelte";
  import PeriodToolbar from "$lib/components/PeriodToolbar.svelte";
  import { formatDateOnly, formatDuration } from "$lib/format";
  import RouteIntro from "$lib/components/RouteIntro.svelte";
  import LoadingBoundary from "$lib/components/LoadingBoundary.svelte";
  import { useTheme } from "$lib/themes/context.svelte";
  import ThemeRouteRenderer from "@iroha/shared/theme-ui/ThemeRouteRenderer.svelte";
  import { hasThemeRoute } from "$lib/themes/registry";
  import { createNightState } from "./night-state.svelte";

  const theme = useTheme();
  const t = createNightState();
</script>

<svelte:head>
  <title>Night · iroha</title>
</svelte:head>

<section class="sleep-shell">
  {#if hasThemeRoute(theme.definition(), "sleep")}
    <LoadingBoundary
      resource={[t.sessionsResource, t.aggregatesResource]}
      preserveLayout
      label="Loading sleep data…"
    >
      {#if t.sessionsResource.error}
        <p class="error" role="alert">
          Sleep could not be loaded: {t.sessionsResource.error}
        </p>
      {/if}
      <ThemeRouteRenderer
        route="sleep"
        props={{
          sessions: t.sessions,
          selected: t.selected,
          averageAsleep: t.averageAsleep,
          averageEfficiency: t.averageEfficiency,
          sleepSummary: t.sleepSummary,
          rollupBuckets: t.rollupBuckets,
          rollupGranularity: t.rollupGranularity,
          rollupScope: t.sleepScope,
          onOpenDetail: (session: SleepSession) =>
            void goto(`/night/${session.id}`),
        }}
      >
        {#snippet children()}
          <PeriodToolbar title="Sleep history scope" ariaLabel="Sleep period">
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
          <SleepScopeSummary
            summary={t.sleepSummary}
            scope={t.sleepScope}
            theme={theme.language()}
          />
        {/snippet}
      </ThemeRouteRenderer>
    </LoadingBoundary>
    {#if t.hasMore}
      <div
        bind:this={t.loadMoreSentinel}
        class="theme-load-more"
        aria-live="polite"
      >
        {#if t.loadingMore}<span>Loading more nights…</span>{:else}<button
            type="button"
            onclick={() => t.loadSessions(true)}>Load more nights</button
          >{/if}
      </div>
    {/if}
  {:else}
    <RouteIntro
      eyebrow="Night / recovery history"
      title="How are you recovering?"
      description="A long view of your nights, with the latest one ready to inspect and the longer rhythm close at hand."
      actionHref="/"
      actionLabel="Back to Today"
    />

    {#if t.sessionsResource.loading && t.sessions.length === 0}
      <section class="status tile"><p>Loading your sleep history…</p></section>
    {:else if t.sessionsResource.error && !t.selected}
      <section class="status tile">
        <p class="error">
          Sleep could not be loaded: {t.sessionsResource.error}
        </p>
      </section>
    {:else if !t.selected}
      <section class="status tile">
        <p class="muted">No sleep sessions imported yet.</p>
      </section>
    {:else}
      <PeriodToolbar title="Sleep history scope" ariaLabel="Sleep period">
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
      <SleepScopeSummary
        summary={t.sleepSummary}
        scope={t.sleepScope}
        theme={theme.language()}
      />
      <section class="hero tile">
        <div class="hero-orb"></div>
        <div class="hero-topline">
          <div>
            <p class="eyebrow">{t.heroEyebrow}</p>
            <h2>{formatDuration(t.selected.asleep_s)} <span>asleep</span></h2>
          </div>
          <div class="hero-status">
            <span class="status-pill"
              >{t.selected.is_main_sleep
                ? "Primary overnight sleep"
                : "Short session"}</span
            >
            <button
              class="info-button"
              type="button"
              aria-label="Explain sleep classification"
              title="Primary overnight sleep means at least 3 hours asleep. Shorter sessions are treated as naps or fragments."
              >i</button
            >
          </div>
        </div>
        <p class="hero-copy">
          {#if t.selected.is_main_sleep}
            Your main overnight window was {formatDuration(
              t.selected.time_in_bed_s,
            )} in bed, with {Math.round(t.selected.efficiency * 100)}%
            efficiency.
          {:else}
            This session is under three hours asleep, so iroha treats it as a
            nap or short fragment.
          {/if}
        </p>
        <div class="hero-metrics">
          <div>
            <span>In bed</span><strong
              >{formatDuration(t.selected.time_in_bed_s)}</strong
            >
          </div>
          <div>
            <span>Efficiency</span><strong
              >{Math.round(t.selected.efficiency * 100)}%</strong
            >
          </div>
          <div>
            <span>Deep + REM</span><strong
              >{formatDuration(t.selected.deep_s + t.selected.rem_s)}</strong
            >
          </div>
          <div>
            <span>Source</span><strong
              >{t.selected.source || "Apple Health"}</strong
            >
          </div>
        </div>
      </section>

      <section class="insight-strip" aria-label="Recent sleep context">
        <StatTile
          label="Main sleep"
          value={(
            t.sleepSummary?.main_sleep_count ?? t.loadedMainSleep.length
          ).toLocaleString()}
          sub="Canonical sessions in scope"
        />
        <StatTile
          label="Naps"
          value={(t.sleepSummary?.nap_count ?? 0).toLocaleString()}
          sub="Separate from main sleep"
        />
        <StatTile
          label="Wake dates"
          value={(
            t.sleepSummary?.observed_wake_dates ?? t.loadedMainSleep.length
          ).toLocaleString()}
          sub="Distinct canonical dates"
        />
      </section>

      <section class="trend-panel tile">
        <header class="section-heading">
          <div>
            <p class="eyebrow">The long view</p>
            <h2>Sleep over time</h2>
          </div>
        </header>
        {#if t.aggregatesResource.loading && t.yearBuckets.length === 0}
          <p class="muted panel-loading">Building your history…</p>
        {:else if t.aggregatesResource.error}
          <p class="error panel-loading">
            History could not be loaded: {t.aggregatesResource.error}
          </p>
        {:else}
          <div class="year-trend">
            {#each t.visibleYears as bucket (bucket.period)}
              <div class="year-card">
                <div class="year-heading">
                  <strong>{t.formatPeriod(bucket.period, "year")}</strong><span
                    >{bucket.session_count} sessions</span
                  >
                </div>
                <div class="year-bar">
                  <span
                    style={`width: ${(bucket.session_count / t.yearMaxSessions) * 100}%`}
                  ></span>
                </div>
                <div class="year-detail">
                  <span
                    >{formatDuration(bucket.average_asleep_s)} avg asleep</span
                  ><b>{Math.round(bucket.average_efficiency * 100)}%</b>
                </div>
              </div>
            {/each}
          </div>
          {#if t.focusedBucket}
            <div class="focus-callout">
              <span
                >{t.selectedMonth !== ""
                  ? t.formatPeriod(t.focusedBucket.period, "month")
                  : t.selectedYear}</span
              ><strong
                >{t.focusedBucket.session_count} sessions · {formatDuration(
                  t.focusedBucket.average_asleep_s,
                )} average asleep</strong
              ><em
                >{t.focusedBucket.main_sleep_count} primary overnight sessions</em
              >
            </div>
          {/if}
          <div class="trend-legend">
            <span><i class="dot dot-session"></i>Recorded sessions</span><span
              >Average efficiency shown per year</span
            >
          </div>
        {/if}
      </section>

      <div class="analysis-grid">
        <section class="stage-panel tile">
          <header class="section-heading">
            <div>
              <p class="eyebrow">Last night</p>
              <h2>Sleep architecture</h2>
            </div>
          </header>
          <div class="architecture">
            <SleepArchitectureChart
              stages={t.architectureStages}
              selectedStage={t.selectedStage}
              onStageSelect={(stage) => (t.selectedStage = stage)}
              onStageHover={(stage) => (t.hoveredStage = stage)}
            />
            <div class="stage-list">
              {#each t.architectureStages as stage (stage.name)}
                <button
                  class:selected={t.selectedStage === stage.name}
                  class="stage-button"
                  onclick={() => (t.selectedStage = stage.name)}
                  ><i class="dot" style={`background: ${stage.color}`}></i><span
                    >{stage.name}</span
                  ><b>{formatDuration(stage.value)}</b></button
                >
              {/each}
              <div class="stage-focus">
                <span>Selected stage</span><strong>{t.activeStage}</strong><b
                  >{formatDuration(t.activeStageSeconds)}</b
                >
              </div>
            </div>
          </div>
          <p class="panel-footnote">
            Stage estimates are reconstructed from Apple Health records and are
            best read as patterns, not clinical measurements.
          </p>
        </section>

        <section class="month-panel tile">
          <header class="section-heading">
            <div>
              <p class="eyebrow">The rhythm</p>
              <h2>Monthly pattern</h2>
            </div>
            <span class="section-note">Average asleep</span>
          </header>
          <div class="month-list">
            {#each t.visibleMonths.slice(0, 12) as bucket (bucket.period)}
              <div class="month-row">
                <span>{t.formatPeriod(bucket.period, "month")}</span>
                <div class="month-bar">
                  <i
                    style={`width: ${(bucket.average_asleep_s / t.monthMaxAsleep) * 100}%`}
                  ></i>
                </div>
                <b>{formatDuration(bucket.average_asleep_s)}</b>
              </div>
            {/each}
          </div>
        </section>
      </div>

      <section class="night-panel tile">
        <header class="section-heading">
          <div>
            <p class="eyebrow">Drill down</p>
            <h2>{t.nightsHeading}</h2>
            <p class="muted">Open a night to see its stage timeline.</p>
          </div>
          <span class="section-note"
            >{t.sessions.length}{t.hasMore ? "+" : ""} loaded · {t.sleepSummary
              ?.session_count ?? t.sessions.length} total sessions</span
          >
        </header>
        <div class="night-layout">
          <div bind:this={t.nightListContainer} class="night-list">
            {#each t.sessions as session (session.id)}
              <button
                class:selected={t.selected && t.selected.id === session.id}
                class="night-row"
                type="button"
                onclick={() => void goto(`/night/${session.id}`)}
                onmouseenter={() => t.selectSession(session)}
                onfocus={() => t.selectSession(session)}
                aria-label={`${formatDateOnly(session.wake_date)}, ${session.is_main_sleep ? "primary overnight sleep" : "short session"}, ${formatDuration(session.asleep_s)} asleep, ${Math.round(session.efficiency * 100)} percent efficiency`}
              >
                <span class="night-date"
                  >{formatDateOnly(session.wake_date)}</span
                ><span>{formatDuration(session.asleep_s)}</span><b
                  >{Math.round(session.efficiency * 100)}%</b
                ><em class:primary={session.is_main_sleep}
                  >{session.is_main_sleep ? "Primary" : "Short"}</em
                >
              </button>
            {/each}
            <div
              bind:this={t.loadMoreSentinel}
              class="load-more-sentinel"
              aria-live="polite"
            >
              {#if t.loadingMore}<span>Loading more nights…</span
                >{:else if t.hasMore}<span>Scroll for more nights</span>{/if}
            </div>
          </div>
          <div class="timeline-card">
            <div class="timeline-meta">
              <span>{formatDateOnly(t.selected.wake_date)}</span><span
                >{t.selected.is_main_sleep
                  ? "Primary overnight sleep"
                  : "Short session"}</span
              >
            </div>
            <p class="muted panel-loading">
              Open this session to inspect its stage timeline.
            </p>
            <a href={`/night/${t.selected.id}`}>Open session details</a>
          </div>
        </div>
        {#if t.hasMore}
          <button
            class="load-more-button"
            type="button"
            disabled={t.loadingMore}
            onclick={() => t.loadSessions(true)}
          >
            {t.loadingMore ? "Loading…" : "Load more nights"}
          </button>
        {/if}
      </section>
    {/if}
  {/if}
</section>

<style>
  .sleep-shell {
    display: grid;
    gap: 1rem;
  }
  .section-heading {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1rem;
  }
  .section-heading h2 {
    margin: 0;
  }
  .section-heading .muted {
    margin: 0.35rem 0 0;
  }
  .eyebrow {
    margin: 0 0 0.35rem;
    color: var(--accent);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.11em;
    text-transform: uppercase;
  }
  .section-note {
    color: var(--text-muted);
    font-size: 0.78rem;
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
  .hero {
    position: relative;
    overflow: hidden;
    min-height: 16rem;
    padding: 1.5rem;
    background:
      radial-gradient(
        circle at 92% 8%,
        color-mix(in srgb, var(--accent) 28%, transparent),
        transparent 34%
      ),
      radial-gradient(
        circle at 70% 100%,
        color-mix(in srgb, var(--accent-2) 18%, transparent),
        transparent 42%
      ),
      var(--surface);
  }
  .hero-orb {
    position: absolute;
    width: 15rem;
    height: 15rem;
    right: -5rem;
    top: -7rem;
    border: 1px solid color-mix(in srgb, var(--accent) 25%, transparent);
    border-radius: 50%;
    box-shadow:
      0 0 0 1.5rem color-mix(in srgb, var(--accent) 4%, transparent),
      0 0 0 3rem color-mix(in srgb, var(--accent) 2.5%, transparent);
  }
  .hero-topline,
  .hero-metrics {
    position: relative;
    z-index: 1;
  }
  .hero-topline {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .hero h2 {
    margin: 0;
    font-size: clamp(2.8rem, 8vw, 5.4rem);
    line-height: 0.95;
    letter-spacing: -0.07em;
  }
  .hero h2 span {
    margin-left: 0.35rem;
    color: var(--text-muted);
    font-size: 0.28em;
    font-weight: 600;
    letter-spacing: -0.01em;
  }
  .hero-status {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    padding-top: 0.25rem;
  }
  .status-pill {
    padding: 0.45rem 0.65rem;
    border: 1px solid color-mix(in srgb, var(--accent) 38%, transparent);
    border-radius: 99px;
    background: color-mix(in srgb, var(--accent) 12%, transparent);
    color: var(--accent);
    font-size: 0.75rem;
    font-weight: 700;
  }
  .info-button {
    width: 1.4rem;
    height: 1.4rem;
    border: 1px solid var(--border);
    border-radius: 50%;
    background: transparent;
    color: var(--text-muted);
    cursor: help;
  }
  .hero-copy {
    position: relative;
    max-width: 35rem;
    margin: 1.25rem 0 1.5rem;
    color: var(--text);
    font-size: 0.92rem;
    line-height: 1.55;
  }
  .hero-metrics {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 1rem;
    padding-top: 1rem;
    border-top: 1px solid color-mix(in srgb, var(--border) 45%, transparent);
  }
  .hero-metrics div {
    display: grid;
    gap: 0.25rem;
  }
  .hero-metrics span {
    color: var(--text-muted);
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }
  .hero-metrics strong {
    overflow: hidden;
    color: var(--text);
    font-size: 0.88rem;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .insight-strip {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 0.75rem;
  }
  .trend-panel,
  .stage-panel,
  .month-panel,
  .night-panel {
    padding: 1.25rem;
  }
  .trend-panel {
    background:
      linear-gradient(
        135deg,
        color-mix(in srgb, var(--accent) 7%, transparent),
        transparent 42%
      ),
      var(--tile-surface);
  }
  .panel-loading {
    min-height: 7rem;
    display: grid;
    place-items: center;
  }
  .year-trend {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(8rem, 1fr));
    gap: 0.75rem;
    margin-top: 1.5rem;
  }
  .year-card {
    padding: 0.8rem;
    border: 1px solid var(--border);
    border-radius: 10px;
    background: color-mix(in srgb, var(--surface-2) 25%, transparent);
  }
  .year-heading,
  .year-detail {
    display: flex;
    justify-content: space-between;
    gap: 0.5rem;
    font-size: 0.78rem;
  }
  .year-heading span,
  .year-detail span {
    color: var(--text-muted);
  }
  .year-bar {
    height: 0.48rem;
    margin: 1rem 0 0.6rem;
    overflow: hidden;
    border-radius: 99px;
    background: var(--surface-2);
  }
  .year-bar span {
    display: block;
    height: 100%;
    border-radius: inherit;
    background: linear-gradient(90deg, var(--accent), var(--accent-2));
  }
  .year-detail {
    font-size: 0.7rem;
  }
  .year-detail b {
    color: var(--accent);
  }
  .trend-legend {
    display: flex;
    gap: 1rem;
    margin-top: 1rem;
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  .focus-callout {
    display: flex;
    align-items: baseline;
    gap: 0.75rem;
    margin-top: 1rem;
    padding: 0.75rem 0.9rem;
    border-left: 2px solid var(--accent);
    background: color-mix(in srgb, var(--accent) 8%, transparent);
    font-size: 0.78rem;
  }
  .focus-callout span {
    color: var(--accent);
    font-weight: 750;
  }
  .focus-callout strong {
    color: var(--text);
  }
  .focus-callout em {
    color: var(--text-muted);
    font-style: normal;
  }
  .analysis-grid {
    display: grid;
    grid-template-columns: minmax(0, 0.9fr) minmax(0, 1.1fr);
    gap: 1rem;
  }
  .architecture {
    display: flex;
    align-items: center;
    gap: 1.5rem;
    margin: 1.5rem 0 1rem;
  }
  .stage-list {
    display: grid;
    flex: 1;
    gap: 0.55rem;
  }
  .stage-button {
    display: grid;
    grid-template-columns: 0.65rem 1fr auto;
    gap: 0.45rem;
    align-items: center;
    padding: 0.25rem 0.35rem;
    border: 1px solid transparent;
    border-radius: 6px;
    background: transparent;
    color: var(--text-muted);
    text-align: left;
    font: inherit;
    font-size: 0.78rem;
    cursor: pointer;
  }
  .stage-button:hover,
  .stage-button:focus-visible,
  .stage-button.selected {
    border-color: var(--border);
    background: color-mix(in srgb, var(--accent) 8%, transparent);
    color: var(--text);
    outline: none;
  }
  .stage-list b {
    color: var(--text);
    font-weight: 650;
  }
  .stage-focus {
    display: grid;
    grid-template-columns: 1fr auto auto;
    gap: 0.5rem;
    align-items: baseline;
    margin-top: 0.25rem;
    padding-top: 0.65rem;
    border-top: 1px solid var(--border);
    font-size: 0.7rem;
  }
  .stage-focus span {
    color: var(--text-muted);
  }
  .stage-focus strong {
    color: var(--accent);
  }
  .stage-focus b {
    color: var(--text);
  }
  .dot {
    display: inline-block;
    width: 0.55rem;
    height: 0.55rem;
    border-radius: 50%;
  }
  .dot-session {
    background: var(--accent);
  }
  .panel-footnote {
    margin: 0;
    color: var(--text-muted);
    font-size: 0.72rem;
    line-height: 1.45;
  }
  .month-list {
    display: grid;
    gap: 0.75rem;
    margin-top: 1.5rem;
  }
  .month-row {
    display: grid;
    grid-template-columns: 5.5rem 1fr auto;
    gap: 0.65rem;
    align-items: center;
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .month-row b {
    color: var(--text);
    font-weight: 650;
  }
  .month-bar {
    height: 0.42rem;
    overflow: hidden;
    border-radius: 99px;
    background: var(--surface-2);
  }
  .month-bar i {
    display: block;
    height: 100%;
    border-radius: inherit;
    background: linear-gradient(90deg, var(--accent), var(--accent-2));
  }
  .night-layout {
    display: grid;
    grid-template-columns: minmax(15rem, 0.75fr) minmax(0, 1.25fr);
    gap: 1.5rem;
    margin-top: 1.25rem;
  }
  .night-list {
    max-height: 19rem;
    overflow: auto;
  }
  .night-row {
    display: grid;
    grid-template-columns: 1fr auto auto auto;
    width: 100%;
    gap: 0.75rem;
    padding: 0.7rem 0.45rem;
    border: 0;
    border-top: 0;
    border-radius: 7px;
    background: transparent;
    color: var(--text-muted);
    text-align: left;
    cursor: pointer;
    font-size: 0.78rem;
  }
  .night-row:hover,
  .night-row:focus-visible,
  .night-row.selected {
    background: color-mix(in srgb, var(--accent) 8%, transparent);
    color: var(--text);
    outline: none;
  }
  .night-row.selected .night-date {
    color: var(--accent);
  }
  .night-row b {
    color: var(--text);
    font-weight: 650;
  }
  .night-row em {
    color: var(--text-muted);
    font-size: 0.68rem;
    font-style: normal;
  }
  .night-row em.primary {
    color: var(--accent);
  }
  .timeline-card {
    align-self: center;
    padding: 1.25rem;
    border: 1px solid var(--border);
    border-radius: 10px;
    background: color-mix(in srgb, var(--surface-2) 25%, transparent);
  }
  .timeline-meta {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .load-more-sentinel {
    min-height: 2rem;
    display: grid;
    place-items: center;
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .load-more-button {
    display: block;
    margin: 0.5rem auto 0;
    padding: 0.55rem 0.8rem;
    border: 1px solid var(--border);
    border-radius: 8px;
    background: var(--surface-2);
    color: var(--text);
    font: inherit;
    font-size: 0.78rem;
    cursor: pointer;
  }
  .load-more-button:hover {
    border-color: var(--accent);
  }
  .load-more-button:disabled {
    cursor: wait;
    opacity: 0.6;
  }
  @media (max-width: 768px) {
    .hero-topline,
    .section-heading {
      flex-direction: column;
    }
    .hero-status {
      padding-top: 0;
    }
    .hero-metrics,
    .insight-strip,
    .analysis-grid,
    .night-layout {
      grid-template-columns: 1fr 1fr;
    }
    .analysis-grid,
    .night-layout {
      grid-column: 1 / -1;
    }
    .hero-metrics {
      gap: 0.7rem;
    }
    .focus-callout {
      align-items: flex-start;
      flex-direction: column;
      gap: 0.25rem;
    }
  }
  @media (max-width: 640px) {
    .hero-metrics,
    .insight-strip,
    .analysis-grid,
    .night-layout {
      grid-template-columns: 1fr;
    }
    .architecture {
      align-items: flex-start;
      gap: 0.75rem;
    }
    .night-row em {
      display: none;
    }
  }
</style>
