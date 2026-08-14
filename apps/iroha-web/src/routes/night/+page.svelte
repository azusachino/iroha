<script lang="ts">
  import { goto, replaceState } from "$app/navigation";
  import { page } from "$app/state";
  import { onMount } from "svelte";
  import {
    getSleepSegments,
    listSleep,
    listSleepAggregates,
    type SleepAggregateBucket,
    type SleepSegment,
    type SleepSession,
  } from "$lib/api";
  import StatTile from "@iroha/shared/StatTile.svelte";
  import SleepScopeSummary from "@iroha/shared/theme-ui/components/SleepScopeSummary.svelte";
  import SleepArchitectureChart from "@iroha/shared/theme-ui/components/SleepArchitectureChart.svelte";
  import SleepTimelineChart from "@iroha/shared/theme-ui/components/SleepTimelineChart.svelte";
  import PeriodSelector from "$lib/components/PeriodSelector.svelte";
  import PeriodToolbar from "$lib/components/PeriodToolbar.svelte";
  import {
    formatDate,
    formatDateOnly,
    formatDuration,
    formatMonth,
  } from "$lib/format";
  import { currentYear } from "@iroha/shared/month";
  import RouteIntro from "$lib/components/RouteIntro.svelte";
  import { useTheme } from "$lib/themes/context.svelte";
  import ThemeRouteRenderer from "@iroha/shared/theme-ui/ThemeRouteRenderer.svelte";
  import { hasThemeRoute } from "$lib/themes/registry";

  const PAGE_SIZE = 31;
  const initialMonthParam = page.url.searchParams.get("month") ?? "";
  const initialMonth = /^\d{4}-\d{2}$/.test(initialMonthParam)
    ? initialMonthParam
    : "";
  const initialYearParam = page.url.searchParams.get("year") ?? "";
  const initialYear = /^\d{4}$/.test(initialYearParam)
    ? initialYearParam
    : initialMonth.slice(0, 4) || currentYear();
  let sessions = $state<SleepSession[]>([]);
  let selected = $state<SleepSession | null>(null);
  let segments = $state<SleepSegment[]>([]);
  let sessionsLoading = $state(true);
  let loadingMore = $state(false);
  let cursor = $state<string | null>(null);
  let hasMore = $state(false);
  let loadMoreSentinel = $state<HTMLDivElement>();
  let nightListContainer = $state<HTMLDivElement>();
  let segmentCache = $state<Record<string, SleepSegment[]>>({});
  let selectionRequest = 0;
  let segmentsLoading = $state(false);
  let error = $state<string | null>(null);
  let yearBuckets = $state<SleepAggregateBucket[]>([]);
  let monthBuckets = $state<SleepAggregateBucket[]>([]);
  let aggregatesLoading = $state(true);
  let aggregatesError = $state<string | null>(null);
  let selectedYear = $state(initialYear);
  let selectedMonth = $state(initialMonth);
  let selectedStage = $state("Core");
  let hoveredStage = $state<string | null>(null);
  const theme = useTheme();

  const loadedMainSleep = $derived(
    sessions.filter((session) => session.is_main_sleep),
  );
  const loadedAverageAsleep = $derived(
    loadedMainSleep.length
      ? loadedMainSleep.reduce(
          (total, session) => total + session.asleep_s,
          0,
        ) / loadedMainSleep.length
      : 0,
  );
  const loadedAverageEfficiency = $derived(
    loadedMainSleep.length
      ? loadedMainSleep.reduce(
          (total, session) => total + session.efficiency,
          0,
        ) / loadedMainSleep.length
      : 0,
  );
  const monthlyBuckets = $derived(monthBuckets.slice().reverse());
  const yearlyBuckets = $derived(yearBuckets.slice().reverse());
  const availableYears = $derived(
    yearBuckets.map((bucket) => bucket.period.slice(0, 4)).reverse(),
  );
  const availableMonths = $derived(
    monthBuckets
      .filter(
        (bucket) =>
          selectedYear === "" || bucket.period.slice(0, 4) === selectedYear,
      )
      .map((bucket) => bucket.period.slice(0, 7))
      .reverse(),
  );
  const periodYears = $derived(
    availableYears.map((year) => ({ value: year, label: year })),
  );
  const periodMonths = $derived(
    availableMonths.map((month) => ({
      value: month,
      label: formatPeriod(`${month}-01T00:00:00Z`, "month"),
    })),
  );
  const visibleYears = $derived(
    selectedYear === ""
      ? yearlyBuckets
      : yearlyBuckets.filter(
          (bucket) => bucket.period.slice(0, 4) === selectedYear,
        ),
  );
  const visibleMonths = $derived(
    monthlyBuckets.filter((bucket) => {
      const period = bucket.period.slice(0, 7);
      return (
        (selectedYear === "" || period.slice(0, 4) === selectedYear) &&
        (selectedMonth === "" || period === selectedMonth)
      );
    }),
  );
  const focusedBucket = $derived(
    selectedMonth !== ""
      ? monthBuckets.find(
          (bucket) => bucket.period.slice(0, 7) === selectedMonth,
        )
      : selectedYear !== ""
        ? yearBuckets.find(
            (bucket) => bucket.period.slice(0, 4) === selectedYear,
          )
        : null,
  );
  const isPeriodFiltered = $derived(selectedYear !== "");
  function combineBuckets(
    buckets: SleepAggregateBucket[],
    period: string,
  ): SleepAggregateBucket | null {
    if (buckets.length === 0) return null;
    const mainSleepCount = buckets.reduce(
      (total, bucket) => total + bucket.main_sleep_count,
      0,
    );
    const weightedAverage = (
      field: "average_asleep_s" | "average_efficiency",
    ) =>
      mainSleepCount === 0
        ? 0
        : buckets.reduce(
            (total, bucket) => total + bucket[field] * bucket.main_sleep_count,
            0,
          ) / mainSleepCount;
    return {
      period,
      session_count: buckets.reduce(
        (total, bucket) => total + bucket.session_count,
        0,
      ),
      main_sleep_count: mainSleepCount,
      nap_count: buckets.reduce((total, bucket) => total + bucket.nap_count, 0),
      observed_wake_dates: buckets.reduce(
        (total, bucket) => total + (bucket.observed_wake_dates ?? 0),
        0,
      ),
      average_asleep_s: weightedAverage("average_asleep_s"),
      average_time_in_bed_s:
        mainSleepCount === 0
          ? 0
          : buckets.reduce(
              (total, bucket) =>
                total + bucket.average_time_in_bed_s * bucket.main_sleep_count,
              0,
            ) / mainSleepCount,
      average_efficiency: weightedAverage("average_efficiency"),
      core_s: buckets.reduce((total, bucket) => total + bucket.core_s, 0),
      deep_s: buckets.reduce((total, bucket) => total + bucket.deep_s, 0),
      rem_s: buckets.reduce((total, bucket) => total + bucket.rem_s, 0),
      awake_s: buckets.reduce((total, bucket) => total + bucket.awake_s, 0),
      unspecified_s: buckets.reduce(
        (total, bucket) => total + bucket.unspecified_s,
        0,
      ),
    };
  }
  const sleepSummary = $derived.by<SleepAggregateBucket | null>(() => {
    if (selectedMonth !== "") {
      return (
        monthBuckets.find(
          (bucket) => bucket.period.slice(0, 7) === selectedMonth,
        ) ?? null
      );
    }
    if (selectedYear !== "") {
      return (
        yearBuckets.find(
          (bucket) => bucket.period.slice(0, 4) === selectedYear,
        ) ?? null
      );
    }
    return combineBuckets(yearBuckets, "lifetime");
  });
  const sleepScope = $derived(selectedMonth || selectedYear || "Lifetime");
  const rollupGranularity = $derived<"month" | "year">(
    selectedYear === "" ? "year" : "month",
  );
  const rollupBuckets = $derived.by(() => {
    if (selectedMonth !== "") return [];
    if (selectedYear !== "") {
      return monthBuckets.filter(
        (bucket) => bucket.period.slice(0, 4) === selectedYear,
      );
    }
    return yearBuckets;
  });
  const averageAsleep = $derived(
    sleepSummary?.average_asleep_s ?? loadedAverageAsleep,
  );
  const averageEfficiency = $derived(
    sleepSummary?.average_efficiency ?? loadedAverageEfficiency,
  );
  const heroEyebrow = $derived(
    isPeriodFiltered
      ? `Selected night · ${selectedMonth !== "" ? formatPeriod(`${selectedMonth}-01T00:00:00Z`, "month") : selectedYear}`
      : `Last night · ${selected ? formatDateOnly(selected.wake_date) : ""}`,
  );
  const nightsHeading = $derived(
    selectedMonth !== ""
      ? "Sessions in selected month"
      : "Recent session detail",
  );
  const monthMaxAsleep = $derived(
    Math.max(1, ...monthBuckets.map((bucket) => bucket.average_asleep_s)),
  );
  const yearMaxSessions = $derived(
    Math.max(1, ...yearBuckets.map((bucket) => bucket.session_count)),
  );
  const architectureStages = $derived([
    { name: "Core", value: selected?.core_s ?? 0, color: "var(--accent)" },
    { name: "Deep", value: selected?.deep_s ?? 0, color: "var(--accent-2)" },
    { name: "REM", value: selected?.rem_s ?? 0, color: "var(--ring-move)" },
    {
      name: "Awake",
      value: selected?.awake_s ?? 0,
      color: "var(--ring-exercise)",
    },
    ...(selected?.unspecified_s
      ? [
          {
            name: "Unspecified",
            value: selected.unspecified_s,
            color: "var(--text-muted)",
          },
        ]
      : []),
  ]);
  const activeStage = $derived(hoveredStage ?? selectedStage);
  const activeStageSeconds = $derived(
    activeStage === "Core"
      ? (selected?.core_s ?? 0)
      : activeStage === "Deep"
        ? (selected?.deep_s ?? 0)
        : activeStage === "REM"
          ? (selected?.rem_s ?? 0)
          : activeStage === "Awake"
            ? (selected?.awake_s ?? 0)
            : (selected?.unspecified_s ?? 0),
  );

  function errorMessage(value: unknown): string {
    return value instanceof Error ? value.message : String(value);
  }

  function syncPeriodUrl() {
    const url = new URL(window.location.href);
    if (selectedYear) url.searchParams.set("year", selectedYear);
    else url.searchParams.delete("year");
    if (selectedMonth) url.searchParams.set("month", selectedMonth);
    else url.searchParams.delete("month");
    if (url.href !== window.location.href) replaceState(url, page.state);
  }

  async function selectSession(session: SleepSession) {
    const requestId = ++selectionRequest;
    selected = session;
    segmentsLoading = true;
    if (segmentCache[session.id]) {
      segments = segmentCache[session.id];
      segmentsLoading = false;
      return;
    }
    try {
      const loaded = await getSleepSegments(session.id);
      if (requestId !== selectionRequest) return;
      segmentCache[session.id] = loaded;
      segments = loaded;
    } catch (value) {
      error = errorMessage(value);
      segments = [];
    } finally {
      segmentsLoading = false;
    }
  }

  function selectedRange(): { from?: string; to?: string } {
    if (selectedMonth !== "") {
      const [year, month] = selectedMonth.split("-").map(Number);
      const lastDay = new Date(Date.UTC(year, month, 0)).getUTCDate();
      return {
        from: `${selectedMonth}-01`,
        to: `${selectedMonth}-${String(lastDay).padStart(2, "0")}`,
      };
    }
    if (selectedYear !== "")
      return { from: `${selectedYear}-01-01`, to: `${selectedYear}-12-31` };
    return {};
  }

  let sessionsRequest = 0;

  async function loadSessions(append = false) {
    if (append && (!hasMore || !cursor || loadingMore)) return;
    const requestId = ++sessionsRequest;
    if (append) loadingMore = true;
    else {
      sessionsLoading = true;
      loadingMore = false;
      cursor = null;
      hasMore = false;
    }
    error = null;
    try {
      const page = await listSleep({
        limit: PAGE_SIZE,
        cursor: append ? (cursor ?? undefined) : undefined,
        ...selectedRange(),
      });
      if (requestId !== sessionsRequest) return;
      sessions = append ? [...sessions, ...page.items] : page.items;
      cursor = page.next_cursor;
      hasMore = page.has_more;
      if (!append) {
        if (page.items[0]) await selectSession(page.items[0]);
        else {
          selected = null;
          segments = [];
        }
      }
    } catch (value) {
      if (requestId !== sessionsRequest) return;
      error = errorMessage(value);
    } finally {
      if (requestId === sessionsRequest) {
        sessionsLoading = false;
        loadingMore = false;
      }
    }
  }

  function changeYear(value: string) {
    selectedYear = value;
    selectedMonth = "";
    syncPeriodUrl();
    void loadSessions(false);
  }

  function changeMonth(value: string) {
    selectedMonth = value;
    if (value) selectedYear = value.slice(0, 4);
    syncPeriodUrl();
    void loadSessions(false);
  }

  $effect(() => {
    if (!loadMoreSentinel || !hasMore) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting))
          void loadSessions(true);
      },
      { root: nightListContainer ?? null, rootMargin: "120px" },
    );
    observer.observe(loadMoreSentinel);
    return () => observer.disconnect();
  });

  async function loadAggregates() {
    aggregatesLoading = true;
    aggregatesError = null;
    try {
      const [years, months] = await Promise.all([
        listSleepAggregates("year"),
        listSleepAggregates("month"),
      ]);
      yearBuckets = years.buckets;
      monthBuckets = months.buckets;
      const validYears = new Set(
        yearBuckets.map((bucket) => bucket.period.slice(0, 4)),
      );
      const validMonths = new Set(
        monthBuckets.map((bucket) => bucket.period.slice(0, 7)),
      );
      if (selectedMonth) selectedYear = selectedMonth.slice(0, 4);
      if (selectedYear && !validYears.has(selectedYear)) selectedYear = "";
      if (selectedMonth && !validMonths.has(selectedMonth)) selectedMonth = "";
      syncPeriodUrl();
    } catch (value) {
      aggregatesError = errorMessage(value);
    } finally {
      aggregatesLoading = false;
    }
  }

  function formatPeriod(period: string, granularity: "month" | "year"): string {
    if (granularity === "year") return period.slice(0, 4);
    return formatMonth(period.slice(0, 7));
  }

  onMount(async () => {
    await loadAggregates();
    void loadSessions(false);
  });
</script>

<svelte:head>
  <title>Night · iroha</title>
</svelte:head>

<section class="sleep-shell">
  {#if hasThemeRoute(theme.definition(), "sleep")}
    {#if sessionsLoading && sessions.length === 0}
      <section class="status tile"><p>Loading sleep data…</p></section>
    {:else if error && !selected}
      <section class="status tile">
        <p class="error">Sleep could not be loaded: {error}</p>
      </section>
    {:else if sessions.length === 0}
      <section class="status tile">
        <p class="muted">No sleep sessions imported yet.</p>
      </section>
    {:else}
      <ThemeRouteRenderer
        route="sleep"
        props={{
          sessions,
          selected,
          averageAsleep,
          averageEfficiency,
          sleepSummary,
          rollupBuckets,
          rollupGranularity,
          rollupScope: sleepScope,
          onOpenDetail: (session: SleepSession) =>
            void goto(`/night/${session.id}`),
        }}
      >
        {#snippet children()}
          <PeriodToolbar title="Sleep history scope" ariaLabel="Sleep period">
            <PeriodSelector
              years={periodYears}
              months={periodMonths}
              year={selectedYear}
              month={selectedMonth}
              monthDisabled={!selectedYear}
              surface="inline"
              onYear={changeYear}
              onMonth={changeMonth}
            />
          </PeriodToolbar>
          <SleepScopeSummary
            summary={sleepSummary}
            scope={sleepScope}
            theme={theme.language()}
          />
        {/snippet}
      </ThemeRouteRenderer>
      {#if hasMore}
        <div
          bind:this={loadMoreSentinel}
          class="theme-load-more"
          aria-live="polite"
        >
          {#if loadingMore}<span>Loading more nights…</span>{:else}<button
              type="button"
              onclick={() => loadSessions(true)}>Load more nights</button
            >{/if}
        </div>
      {/if}
    {/if}
  {:else}
    <RouteIntro
      eyebrow="Night / recovery history"
      title="How are you recovering?"
      description="A long view of your nights, with the latest one ready to inspect and the longer rhythm close at hand."
      actionHref="/"
      actionLabel="Back to Today"
    />

    {#if sessionsLoading && sessions.length === 0}
      <section class="status tile"><p>Loading your sleep history…</p></section>
    {:else if error && !selected}
      <section class="status tile">
        <p class="error">Sleep could not be loaded: {error}</p>
      </section>
    {:else if !selected}
      <section class="status tile">
        <p class="muted">No sleep sessions imported yet.</p>
      </section>
    {:else}
      <PeriodToolbar title="Sleep history scope" ariaLabel="Sleep period">
        <PeriodSelector
          years={periodYears}
          months={periodMonths}
          year={selectedYear}
          month={selectedMonth}
          monthDisabled={!selectedYear}
          surface="inline"
          onYear={changeYear}
          onMonth={changeMonth}
        />
      </PeriodToolbar>
      <SleepScopeSummary
        summary={sleepSummary}
        scope={sleepScope}
        theme={theme.language()}
      />
      <section class="hero tile">
        <div class="hero-orb"></div>
        <div class="hero-topline">
          <div>
            <p class="eyebrow">{heroEyebrow}</p>
            <h2>{formatDuration(selected.asleep_s)} <span>asleep</span></h2>
          </div>
          <div class="hero-status">
            <span class="status-pill"
              >{selected.is_main_sleep
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
          {#if selected.is_main_sleep}
            Your main overnight window was {formatDuration(
              selected.time_in_bed_s,
            )} in bed, with {Math.round(selected.efficiency * 100)}% efficiency.
          {:else}
            This session is under three hours asleep, so iroha treats it as a
            nap or short fragment.
          {/if}
        </p>
        <div class="hero-metrics">
          <div>
            <span>In bed</span><strong
              >{formatDuration(selected.time_in_bed_s)}</strong
            >
          </div>
          <div>
            <span>Efficiency</span><strong
              >{Math.round(selected.efficiency * 100)}%</strong
            >
          </div>
          <div>
            <span>Deep + REM</span><strong
              >{formatDuration(selected.deep_s + selected.rem_s)}</strong
            >
          </div>
          <div>
            <span>Source</span><strong
              >{selected.source || "Apple Health"}</strong
            >
          </div>
        </div>
      </section>

      <section class="insight-strip" aria-label="Recent sleep context">
        <StatTile
          label="Main sleep"
          value={(
            sleepSummary?.main_sleep_count ?? loadedMainSleep.length
          ).toLocaleString()}
          sub="Canonical sessions in scope"
        />
        <StatTile
          label="Naps"
          value={(sleepSummary?.nap_count ?? 0).toLocaleString()}
          sub="Separate from main sleep"
        />
        <StatTile
          label="Wake dates"
          value={(
            sleepSummary?.observed_wake_dates ?? loadedMainSleep.length
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
        {#if aggregatesLoading}
          <p class="muted panel-loading">Building your history…</p>
        {:else if aggregatesError}
          <p class="error panel-loading">
            History could not be loaded: {aggregatesError}
          </p>
        {:else}
          <div class="year-trend">
            {#each visibleYears as bucket (bucket.period)}
              <div class="year-card">
                <div class="year-heading">
                  <strong>{formatPeriod(bucket.period, "year")}</strong><span
                    >{bucket.session_count} sessions</span
                  >
                </div>
                <div class="year-bar">
                  <span
                    style={`width: ${(bucket.session_count / yearMaxSessions) * 100}%`}
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
          {#if focusedBucket}
            <div class="focus-callout">
              <span
                >{selectedMonth !== ""
                  ? formatPeriod(focusedBucket.period, "month")
                  : selectedYear}</span
              ><strong
                >{focusedBucket.session_count} sessions · {formatDuration(
                  focusedBucket.average_asleep_s,
                )} average asleep</strong
              ><em
                >{focusedBucket.main_sleep_count} primary overnight sessions</em
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
              stages={architectureStages}
              {selectedStage}
              onStageSelect={(stage) => (selectedStage = stage)}
              onStageHover={(stage) => (hoveredStage = stage)}
            />
            <div class="stage-list">
              {#each architectureStages as stage (stage.name)}
                <button
                  class:selected={selectedStage === stage.name}
                  class="stage-button"
                  onclick={() => (selectedStage = stage.name)}
                  ><i class="dot" style={`background: ${stage.color}`}></i><span
                    >{stage.name}</span
                  ><b>{formatDuration(stage.value)}</b></button
                >
              {/each}
              <div class="stage-focus">
                <span>Selected stage</span><strong>{activeStage}</strong><b
                  >{formatDuration(activeStageSeconds)}</b
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
            {#each visibleMonths.slice(0, 12) as bucket (bucket.period)}
              <div class="month-row">
                <span>{formatPeriod(bucket.period, "month")}</span>
                <div class="month-bar">
                  <i
                    style={`width: ${(bucket.average_asleep_s / monthMaxAsleep) * 100}%`}
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
            <h2>{nightsHeading}</h2>
            <p class="muted">Select a night to see the stage timeline.</p>
          </div>
          <span class="section-note"
            >{sessions.length}{hasMore ? "+" : ""} loaded · {sleepSummary?.session_count ??
              sessions.length} total sessions</span
          >
        </header>
        <div class="night-layout">
          <div bind:this={nightListContainer} class="night-list">
            {#each sessions as session (session.id)}
              <button
                class:selected={selected.id === session.id}
                class="night-row"
                type="button"
                onclick={() => void goto(`/night/${session.id}`)}
                onmouseenter={() => selectSession(session)}
                onfocus={() => selectSession(session)}
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
              bind:this={loadMoreSentinel}
              class="load-more-sentinel"
              aria-live="polite"
            >
              {#if loadingMore}<span>Loading more nights…</span
                >{:else if hasMore}<span>Scroll for more nights</span>{/if}
            </div>
          </div>
          <div class="timeline-card">
            <div class="timeline-meta">
              <span>{formatDateOnly(selected.wake_date)}</span><span
                >{selected.is_main_sleep
                  ? "Primary overnight sleep"
                  : "Short session"}</span
              >
            </div>
            {#if segmentsLoading}<p class="muted panel-loading">
                Loading stages…
              </p>{:else}<SleepTimelineChart {segments} />{/if}
            <div class="timeline-axis">
              <span>{formatDate(selected.started_at)}</span><span
                >{formatDate(selected.ended_at)}</span
              >
            </div>
          </div>
        </div>
        {#if hasMore}
          <button
            class="load-more-button"
            type="button"
            disabled={loadingMore}
            onclick={() => loadSessions(true)}
          >
            {loadingMore ? "Loading…" : "Load more nights"}
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
  .timeline-meta,
  .timeline-axis {
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
