<script lang="ts">
  import { browser } from "$app/env";
  import { page } from "$app/state";
  import { onMount, untrack } from "svelte";
  import {
    activeDaysFromActivities,
    cityGroupsForRoutes,
    filterByYearAndSport,
    monthlyBuckets,
    yearsFromActivities,
  } from "$lib/public/aggregate";
  import {
    formatDateOnly,
    formatDistance,
    formatDuration,
    formatHumanDuration,
    formatMetricValue,
    formatSport,
  } from "$lib/format";
  import {
    MONTH_LABELS,
    buildPublicSportBuckets,
    sortPublicActivities,
    summarizePublicActivities,
    type PublicSortEntry,
    type PublicSortKey,
  } from "$lib/public/showcase-helpers";
  import { site } from "$lib/public/site";
  import { sportColor } from "@iroha/shared/domain/sport";
  import type {
    Activity,
    ActivityDetail as ActivityDetailData,
    Book,
    Meta,
    RouteFeatureCollection,
    Summary,
  } from "$lib/public/types";
  import PublicReading from "@iroha/shared/theme-ui/grapher/PublicReading.svelte";
  import ActivityHeatmap from "@iroha/shared/theme-ui/components/ActivityHeatmap.svelte";
  import MonthlyBarChart from "@iroha/shared/components/MonthlyBarChart.svelte";
  import YearProgressChart from "@iroha/shared/components/YearProgressChart.svelte";
  import SportBadge from "@iroha/shared/components/SportBadge.svelte";
  import StatTile from "@iroha/shared/components/StatTile.svelte";
  import RouteHeader from "@iroha/shared/components/RouteHeader.svelte";
  import PanelFrame from "@iroha/shared/components/PanelFrame.svelte";
  import RetryNotice from "@iroha/shared/theme-ui/components/RetryNotice.svelte";
  import BrandMark from "$lib/components/BrandMark.svelte";
  import PublicRoutesSection from "$lib/components/PublicRoutesSection.svelte";
  import PublicActivityTable from "$lib/components/PublicActivityTable.svelte";

  let {
    initialData,
  }: {
    initialData?: {
      summary?: Summary | null;
      activities?: Activity[];
      meta?: Meta;
      error?: string;
    };
  } = $props();

  let summary = $state<Summary | null>(
    untrack(() => initialData?.summary ?? null),
  );
  let activities = $state<Activity[]>(
    untrack(() =>
      Array.isArray(initialData?.activities) ? initialData.activities : [],
    ),
  );
  let meta = $state<Meta>(
    untrack(() => initialData?.meta ?? { generated_at: "" }),
  );
  let loading = $state(
    untrack(() => !initialData?.summary && !initialData?.error),
  );
  let loadError = $state<string | null>(
    untrack(() => initialData?.error ?? null),
  );

  let routes = $state<RouteFeatureCollection>({
    type: "FeatureCollection",
    features: [],
  });
  let routesLoading = $state(true);

  // Fetch the reading list when its tab becomes visible.
  let books = $state<Book[]>([]);
  let booksLoading = $state(true);
  let booksUnavailable = $state(false);
  let booksSectionEl = $state<HTMLElement>();
  let booksRequested = false;
  let _booksPromise: Promise<Book[]> | null = null;

  function ensureBooksLoaded() {
    if (booksRequested) return;
    booksRequested = true;
    if (!_booksPromise) {
      _booksPromise = fetch("/public/v1/books").then(async (r) => {
        if (!r.ok) throw new Error("reading list unavailable");
        return (await r.json()) as Book[];
      });
    }
    _booksPromise
      .then((list) => {
        books = Array.isArray(list) ? list : [];
        booksLoading = false;
      })
      .catch(() => {
        booksUnavailable = true;
        booksLoading = false;
      });
  }

  const activeDays = $derived(activeDaysFromActivities(activities));
  const latestDay = $derived(
    activities[0]?.started_at.slice(0, 10) ??
      new Date().toISOString().slice(0, 10),
  );

  const years = $derived(yearsFromActivities(activities));
  let selectedYear = $state<string>(
    untrack(
      () => yearsFromActivities(initialData?.activities ?? [])[0] ?? "2026",
    ),
  );
  let archiveTab = $state<"run" | "books">("run");
  function navigateArchiveTabs(event: KeyboardEvent) {
    if (!["ArrowLeft", "ArrowRight", "Home", "End"].includes(event.key)) return;
    event.preventDefault();
    archiveTab =
      event.key === "Home"
        ? "run"
        : event.key === "End"
          ? "books"
          : archiveTab === "run"
            ? "books"
            : "run";
    const tabs = (event.currentTarget as HTMLElement).parentElement;
    tabs
      ?.querySelector<HTMLButtonElement>(`#archive-tab-${archiveTab}`)
      ?.focus();
  }
  let sportFilter = $state<string | null>(null);
  let cityFilter = $state<string | null>(null);
  let selectedActivityId = $state<string | null>(
    browser
      ? new URLSearchParams(window.location.search).get("activity")
      : null,
  );
  const activityQuery = $derived(page.url.search);
  const selectedActivity = $derived(
    Array.isArray(activities)
      ? activities.find((activity) => activity.id === selectedActivityId)
      : undefined,
  );
  // Details are fetched per activity on selection instead of shipping every
  // activity's route and samplings up front.
  let selectedActivityDetail = $state<ActivityDetailData | undefined>();
  // Issue #116 review fix: list absence alone is not proof of not-found.
  // The notice is only confirmed by an actual 404 detail response for the
  // currently selected id; pending or failing reads keep the archive
  // fallback without a not-found claim, and the state is reset per selection
  // so a stale 404 can never carry into a new id.
  let confirmedMissingActivityId = $state<string | null>(null);
  $effect(() => {
    const id = selectedActivityId;
    selectedActivityDetail = undefined;
    confirmedMissingActivityId = null;
    if (!id) return;
    let cancelled = false;
    void fetch(`/public/v1/activities/${encodeURIComponent(id)}`)
      .then((res) => {
        if (cancelled) return undefined;
        if (res.status === 404) confirmedMissingActivityId = id;
        return res.ok ? (res.json() as Promise<ActivityDetailData>) : undefined;
      })
      .then((detail) => {
        if (!cancelled) selectedActivityDetail = detail;
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  });

  $effect(() => {
    const activityId = new URLSearchParams(
      activityQuery || window.location.search,
    ).get("activity");
    if (activityId !== selectedActivityId) selectedActivityId = activityId;
  });

  function activityHref(id: string): string {
    const url = new URL(page.url.href);
    url.searchParams.set("activity", id);
    return `${url.pathname}${url.search}`;
  }

  function selectYear(year: string) {
    selectedYear = year;
    cityFilter = null;
    visibleCount = batchSize;
  }

  function toggleSport(sport: string) {
    sportFilter = sportFilter === sport ? null : sport;
    cityFilter = null;
    visibleCount = batchSize;
  }

  function toggleCity(city: string) {
    cityFilter = cityFilter === city ? null : city;
  }

  // --- Summary ---
  const selectedYearTotals = $derived(
    summarizePublicActivities(
      filterByYearAndSport(activities, selectedYear, sportFilter),
    ),
  );
  const distanceSub = $derived.by(() => {
    const { distance_known_count, distance_unknown_count } = selectedYearTotals;
    if (distance_unknown_count === 0) return undefined;
    if (distance_known_count === 0)
      return `Distance unavailable for ${distance_unknown_count} ${distance_unknown_count === 1 ? "activity" : "activities"}`;
    return `Known distance for ${distance_known_count} of ${selectedYearTotals.activity_count} activities; ${distance_unknown_count} unavailable`;
  });
  const summaryContext = $derived(
    `${selectedYear || "No records"} · ${sportFilter ? formatSport(sportFilter) : "All sports"}`,
  );
  const distanceValue = $derived(
    selectedYearTotals.distance_unknown_count > 0 &&
      selectedYearTotals.distance_known_count === 0
      ? "—"
      : formatDistance(selectedYearTotals.distance_m),
  );
  const selectedYearRunningCount = $derived(
    activities.filter(
      (activity) =>
        activity.started_at.slice(0, 4) === selectedYear &&
        activity.sport_type.toLowerCase().includes("run"),
    ).length,
  );

  const monthlyAll = $derived(
    monthlyBuckets(filterByYearAndSport(activities, null, sportFilter)),
  );

  const monthSlots = $derived.by(() => {
    if (!selectedYear) return [];
    const byKey = new Map(monthlyAll.map((b) => [b.key, b]));
    return MONTH_LABELS.map((label, idx) => {
      const key = `${selectedYear}-${String(idx + 1).padStart(2, "0")}`;
      return { label, key, bucket: byKey.get(key) };
    });
  });

  const monthMetric = $derived(
    monthSlots.some((slot) => (slot.bucket?.distance_unknown_count ?? 0) > 0) ||
      !monthSlots.some((slot) => (slot.bucket?.distance_known_count ?? 0) > 0)
      ? "activity_count"
      : "distance_m",
  );

  const sportBuckets = $derived(
    buildPublicSportBuckets(
      filterByYearAndSport(activities, selectedYear, null),
    ),
  );

  const sportMax = $derived(
    Math.max(1, ...sportBuckets.map((s) => s.activity_count)),
  );

  // --- Routes & cities ---
  const selectedYearRoutes = $derived(
    selectedYear && routes?.features
      ? routes.features.filter((f) => f.properties.year === selectedYear)
      : (routes?.features ?? []),
  );
  const filteredRoutes = $derived(
    sportFilter && selectedYearRoutes
      ? selectedYearRoutes.filter(
          (f) => f.properties.sport_type === sportFilter,
        )
      : (selectedYearRoutes ?? []),
  );
  const cityGroups = $derived(cityGroupsForRoutes(filteredRoutes));
  const maxRunCount = $derived(
    Math.max(1, ...cityGroups.map((g) => g.runCount)),
  );
  const hasPendingLocations = $derived(
    cityGroups.some((g) => g.status === "pending"),
  );
  const mappedRoutes = $derived(
    cityFilter && filteredRoutes
      ? filteredRoutes.filter(
          (f) => (f.properties.city || "Unknown") === cityFilter,
        )
      : (filteredRoutes ?? []),
  );

  // --- Activities table ---
  const filteredActivities = $derived(
    filterByYearAndSport(activities, selectedYear, sportFilter),
  );

  let sorting = $state<PublicSortEntry[]>([{ id: "started_at", desc: true }]);
  const batchSize = 50;
  let visibleCount = $state(batchSize);

  // Preserve the former TanStack v8 toggle defaults, including Shift state.
  // The existing row ordering below deliberately uses only the leading key.
  function toggleSort(id: PublicSortKey, event: MouseEvent) {
    const current = sorting.find((sort) => sort.id === id);
    const firstDesc = typeof filteredActivities[0]?.[id] !== "string";
    const remove = current && current.desc !== firstDesc;
    const next = {
      id,
      desc: remove ? false : current ? !current.desc : firstDesc,
    };
    if (event.shiftKey && sorting.length) {
      sorting = remove
        ? sorting.filter((sort) => sort.id !== id)
        : current
          ? sorting.map((sort) => (sort.id === id ? next : sort))
          : [...sorting, next];
    } else {
      sorting = remove && sorting.at(-1)?.id === id ? [] : [next];
    }
  }

  const sortedActivities = $derived(
    sortPublicActivities(filteredActivities, sorting),
  );
  const visibleActivities = $derived(sortedActivities.slice(0, visibleCount));
  const hasMoreActivities = $derived(visibleCount < sortedActivities.length);

  function loadMore() {
    visibleCount = Math.min(visibleCount + batchSize, sortedActivities.length);
  }

  function loadMoreNearBottom() {
    if (!hasMoreActivities) return;
    const remaining =
      document.documentElement.scrollHeight -
      (window.scrollY + window.innerHeight);
    if (remaining < 600) loadMore();
  }

  let routesSectionEl = $state<HTMLElement>();
  let routesRequested = false;
  let _routesPromise: Promise<RouteFeatureCollection> | null = null;

  function ensureRoutesLoaded() {
    if (routesRequested) return;
    routesRequested = true;
    if (!_routesPromise) {
      _routesPromise = fetch("/public/v1/routes").then(
        (r) => r.json() as Promise<RouteFeatureCollection>,
      );
    }
    _routesPromise
      .then((r) => {
        routes = r;
        routesLoading = false;
      })
      .catch(() => {
        routesLoading = false;
      });
  }

  $effect(() => {
    if (selectedActivityId) {
      ensureRoutesLoaded();
    }
  });

  $effect(() => {
    const el = routesSectionEl;
    if (!el || !browser) return;
    if (typeof IntersectionObserver === "undefined") {
      ensureRoutesLoaded();
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) {
          ensureRoutesLoaded();
          observer.disconnect();
        }
      },
      { rootMargin: "400px" },
    );
    observer.observe(el);
    return () => observer.disconnect();
  });

  $effect(() => {
    const el = booksSectionEl;
    if (!el || !browser) return;
    if (typeof IntersectionObserver === "undefined") {
      ensureBooksLoaded();
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) {
          ensureBooksLoaded();
          observer.disconnect();
        }
      },
      { rootMargin: "400px" },
    );
    observer.observe(el);
    return () => observer.disconnect();
  });

  async function loadData() {
    loading = true;
    loadError = null;
    try {
      const [s, a, m] = await Promise.all([
        fetch("/public/v1/summary").then(async (r) => {
          if (!r.ok)
            throw new Error("Public activity data is temporarily unavailable.");
          return r.json();
        }),
        fetch("/public/v1/activities").then(async (r) => {
          if (!r.ok)
            throw new Error("Public activity data is temporarily unavailable.");
          return r.json();
        }),
        fetch("/public/v1/meta").then(async (r) => {
          if (!r.ok)
            throw new Error("Public activity data is temporarily unavailable.");
          return r.json();
        }),
      ]);
      summary = s as Summary | null;
      activities = Array.isArray(a) ? (a as Activity[]) : [];
      meta = m as Meta;
      if (a && a.length > 0 && !selectedYear) {
        selectedYear = yearsFromActivities(a)[0] ?? "";
      }
    } catch {
      loadError =
        "Public activity data is temporarily unavailable. No activity totals can be shown.";
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    if (!initialData?.summary && !initialData?.error) {
      void loadData();
    }
    ensureRoutesLoaded();
    window.addEventListener("scroll", loadMoreNearBottom, { passive: true });
    return () => window.removeEventListener("scroll", loadMoreNearBottom);
  });
</script>

<svelte:head>
  <title>{site.name} · public archive</title>
</svelte:head>

{#if loading}
  <div class="content public-loading">
    <div class="public-loading-box">
      <BrandMark size={36} />
      <p role="status">Loading public activity data…</p>
    </div>
  </div>
{:else if loadError}
  <div class="content">
    <h1>Public activity data unavailable</h1>
    <RetryNotice
      message={loadError}
      retryLabel="Try again"
      onRetry={() => void loadData()}
    />
  </div>
{:else}
  {#if selectedActivity}
    <header class="hero tile">
      <div class="hero-topline">
        <a
          class="brand"
          href={page.url.pathname === "/showcase" ? "/showcase" : "/"}
          aria-label={`${site.name} home`}
        >
          <BrandMark size={22} />
          <span class="brand-title">{site.name}</span>
          <span class="eyebrow visually-hidden">{site.name}</span>
        </a>
      </div>
      <h1>The shape of the miles.</h1>
      <p class="hero-summary">
        A public field guide to the routes and rhythms made visible.
      </p>
      <div class="hero-meta" aria-label="Archive metadata">
        <span>Live</span>
        <span>Updated {formatDateOnly(meta.generated_at)}</span>
        <span>iroha v{site.version}</span>
      </div>
    </header>
  {:else}
    <RouteHeader
      title={site.name}
      context={archiveTab === "run"
        ? `Observed year: ${selectedYear || "No records"} · ${sportFilter ? formatSport(sportFilter) : "All sports"}`
        : "A record of books and reading"}
    >
      {#snippet titleSnippet()}
        <a
          class="brand"
          href={page.url.pathname === "/showcase" ? "/showcase" : "/"}
          aria-label={`${site.name} home`}
        >
          <BrandMark size={22} />
          <h1 class="brand-title">{site.name}</h1>
          <span class="eyebrow visually-hidden">{site.name}</span>
        </a>
      {/snippet}
      {#if archiveTab === "run" && years.length > 0}
        <nav class="year-tabs" aria-label="Select year">
          {#each years as year (year)}
            <button
              type="button"
              class:active={selectedYear === year}
              onclick={() => selectYear(year)}>{year}</button
            >
          {/each}
        </nav>
      {/if}
      {#snippet actions()}
        <div class="public-actions">
          <div class="hero-meta" aria-label="Archive metadata">
            <span>Live</span><span
              >Updated {formatDateOnly(meta.generated_at)}</span
            ><span>iroha v{site.version}</span>
          </div>
        </div>
      {/snippet}
    </RouteHeader>
  {/if}

  {#snippet selectedRecord(activity: Activity)}
    <a href={page.url.pathname}>← Back to archive</a>
    <h2>{activity.title || formatSport(activity.sport_type)}</h2>
    <p>
      Distance: {formatDistance(activity.distance_m)} · Duration:
      {formatDuration(activity.duration_s)}
    </p>
  {/snippet}

  {#if selectedActivity}
    {#await import("$lib/components/ActivityDetailView.svelte")}
      <section class="tile" aria-busy="true">
        {@render selectedRecord(selectedActivity)}
        <p role="status">Loading activity details…</p>
      </section>
    {:then { default: DetailView }}
      <DetailView
        activity={selectedActivity}
        detail={selectedActivityDetail}
        {routes}
        backHref={page.url.pathname}
      />
    {:catch}
      <section class="tile" aria-busy="false">
        {@render selectedRecord(selectedActivity)}
        <p role="alert">
          Activity detail view unavailable. Record summary remains.
        </p>
        <button type="button" onclick={() => window.location.reload()}
          >Reload page</button
        >
      </section>
    {/await}
  {:else}
    {#if selectedActivityId && !selectedActivity && confirmedMissingActivityId === selectedActivityId}
      <section
        class="tile notice-tile"
        role="status"
        aria-label="Activity not found"
      >
        <p>
          This activity is not in the public archive. Showing the full archive
          instead.
        </p>
        <a href={page.url.pathname}>View the full archive</a>
      </section>
    {/if}
    <div class="archive-tabs" role="tablist" aria-label="Archive sections">
      {#each ["run", "books"] as tab (tab)}
        <button
          type="button"
          role="tab"
          id={`archive-tab-${tab}`}
          aria-selected={archiveTab === tab}
          aria-controls={`archive-panel-${tab}`}
          tabindex={archiveTab === tab ? 0 : -1}
          onclick={() => (archiveTab = tab as "run" | "books")}
          onkeydown={navigateArchiveTabs}
          >{tab === "run" ? "Run" : "Books"}</button
        >
      {/each}
    </div>
    {#if archiveTab === "run"}
      <div
        class="dashboard"
        role="tabpanel"
        id="archive-panel-run"
        aria-labelledby="archive-tab-run"
        tabindex="0"
      >
        <div class="stat-grid">
          <div class="stat-tile-wrap" style="--accent: #00e5bf;">
            <StatTile
              label="Distance"
              value={distanceValue}
              sub={distanceSub}
              context={summaryContext}
            />
          </div>
          <div class="stat-tile-wrap" style="--accent: #f59e0b;">
            <StatTile
              label="Activities"
              context={summaryContext}
              value={formatMetricValue(
                selectedYearTotals.activity_count,
                "count",
              )}
            />
          </div>
          <div class="stat-tile-wrap" style="--accent: #38bdf8;">
            <StatTile
              label="Running count"
              value={formatMetricValue(selectedYearRunningCount, "count")}
              context={selectedYear
                ? `${selectedYear} · All running records`
                : "No records"}
            />
          </div>
          <div class="stat-tile-wrap" style="--accent: #f43f5e;">
            <StatTile
              label="Total time"
              context={summaryContext}
              value={formatHumanDuration(
                selectedYearTotals.moving_time_s ||
                  selectedYearTotals.duration_s,
              )}
            />
          </div>
        </div>

        <ActivityHeatmap
          days={activeDays}
          endDay={latestDay}
          embedded={false}
        />

        {#if years.length > 0 && selectedYear}
          <div class="analytics-grid">
            <PanelFrame label="Cumulative distance">
              <YearProgressChart
                embedded
                byMonth={monthlyAll}
                year={selectedYear}
                sportName={sportFilter ? formatSport(sportFilter) : undefined}
              />
            </PanelFrame>
            <PanelFrame label="Monthly observations">
              <MonthlyBarChart
                embedded
                points={monthSlots.map((slot) => ({
                  label: slot.label,
                  value: slot.bucket?.[monthMetric] ?? 0,
                }))}
                metric={monthMetric}
                year={selectedYear}
              />
            </PanelFrame>
          </div>
        {/if}

        {#if sportBuckets.length > 0}
          <PanelFrame label="Activities by sport">
            <section class="by-sport">
              <div class="section-kicker eyebrow">
                {selectedYear} by sport
              </div>
              {#each sportBuckets as sport (sport.key)}
                <button
                  type="button"
                  class="sport-row"
                  class:active={sportFilter === sport.key}
                  onclick={() => toggleSport(sport.key)}
                >
                  <SportBadge sport={sport.key} />
                  <span class="bar">
                    <i
                      style={`width: ${Math.max(2, (sport.activity_count / sportMax) * 100)}%; background: ${sportColor(sport.key)}`}
                    ></i>
                  </span>
                  <span class="count">{sport.activity_count}×</span>
                </button>
              {/each}
              {#if sportFilter}
                <button
                  type="button"
                  class="clear-filter"
                  onclick={() => toggleSport(sportFilter!)}
                >
                  Clear sport filter ({formatSport(sportFilter)})
                </button>
              {/if}
            </section>
          </PanelFrame>
        {/if}

        <PublicRoutesSection
          bind:sectionEl={routesSectionEl}
          {routesLoading}
          {filteredRoutes}
          {mappedRoutes}
          {cityGroups}
          {maxRunCount}
          {hasPendingLocations}
          {selectedYear}
          {sportFilter}
          {cityFilter}
          onToggleCity={toggleCity}
          onClearCity={() => (cityFilter = null)}
        />

        <PublicActivityTable
          filteredCount={filteredActivities.length}
          totalSortedCount={sortedActivities.length}
          {visibleActivities}
          {sorting}
          hasMore={hasMoreActivities}
          {activityHref}
          onToggleSort={toggleSort}
          onLoadMore={loadMore}
        />
      </div>
    {:else}
      <div
        class="dashboard"
        role="tabpanel"
        id="archive-panel-books"
        aria-labelledby="archive-tab-books"
        tabindex="0"
      >
        <section class="section-heading" bind:this={booksSectionEl}>
          <h2>Reading</h2>
        </section>

        {#if booksLoading}
          <p class="muted">Loading reading list…</p>
        {:else if booksUnavailable}
          <p class="muted">Reading list is temporarily unavailable.</p>
        {:else if books.length === 0}
          <p class="muted">No reading records yet.</p>
        {:else}
          <PublicReading {books} />
        {/if}
      </div>
    {/if}
  {/if}
{/if}

<style>
  .archive-tabs {
    display: flex;
    gap: 1.5rem;
    border-bottom: 1px solid var(--border);
    margin-bottom: 1.5rem;
  }
  .archive-tabs button {
    padding: 0.75rem 0.25rem;
    border: 0;
    border-bottom: 2px solid transparent;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    font: inherit;
  }
  .archive-tabs button[aria-selected="true"] {
    border-bottom-color: var(--accent);
    color: var(--text);
    font-weight: 600;
  }
  .archive-tabs button:focus-visible {
    outline: 2px solid var(--color-focus);
    outline-offset: 2px;
  }
  .public-loading {
    display: grid;
    place-items: center;
    min-height: 55vh;
  }
  .public-loading-box {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1.25rem;
    color: var(--text-muted);
    font-size: 0.95rem;
    font-weight: 500;
  }
  .public-loading-box p {
    margin: 0;
  }
  .notice-tile {
    margin-bottom: 1.25rem;
  }
  .notice-tile a {
    color: var(--accent);
  }
  .hero {
    position: relative;
    overflow: hidden;
    padding: 2.25rem 1.75rem;
    margin-bottom: 1.25rem;
    background:
      radial-gradient(
        ellipse 70% 60% at 85% 15%,
        color-mix(in srgb, var(--accent) 14%, transparent),
        transparent 70%
      ),
      radial-gradient(
        ellipse 50% 50% at 15% 85%,
        color-mix(in srgb, #38bdf8 10%, transparent),
        transparent 70%
      ),
      var(--tile-surface);
  }
  .brand {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    border-radius: 6px;
    color: inherit;
    text-decoration: none;
  }
  .brand :global(.brand-mark) {
    flex: none;
    border-radius: 6px;
  }
  .brand:hover .brand-title {
    color: var(--accent);
  }
  .brand:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: 3px;
  }
  .hero-topline {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 0.4rem;
  }
  .hero h1 {
    margin: 0;
    font-size: clamp(var(--type-title), 5vw, var(--type-display));
    letter-spacing: -0.03em;
    font-weight: 800;
    background: linear-gradient(135deg, var(--text) 40%, var(--accent) 100%);
    -webkit-background-clip: text;
    background-clip: text;
    -webkit-text-fill-color: transparent;
  }
  .hero-summary {
    max-width: 34rem;
    margin: 0.7rem 0 1.15rem;
    color: var(--text-muted);
    font-size: var(--type-body);
    line-height: 1.5;
  }
  .hero-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 0.45rem 1rem;
    color: var(--text-muted);
    font-size: var(--type-caption);
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }
  .hero-meta span + span {
    padding-left: 1rem;
    border-left: 1px solid var(--border);
  }
  @media (max-width: 640px) {
    .hero-meta span + span {
      padding-left: 0;
      border-left: 0;
    }
  }
  .stat-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 0.75rem;
    margin-bottom: 1.25rem;
  }
  .stat-tile-wrap {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .stat-tile-wrap :global(.stat-tile) {
    height: 100%;
    border-color: color-mix(in srgb, var(--accent) 30%, var(--border));
    transition:
      transform 0.15s ease,
      box-shadow 0.15s ease;
  }
  .stat-tile-wrap :global(.stat-tile:hover) {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px -3px
      color-mix(in srgb, var(--accent) 25%, transparent);
  }
  .brand-title {
    margin: 0;
    font-size: 1.05rem;
    font-weight: 750;
    letter-spacing: -0.02em;
    color: var(--text);
    overflow-wrap: anywhere;
    transition: color 0.15s ease;
  }
  .public-actions {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 0.35rem;
  }
  .public-actions .hero-meta {
    letter-spacing: normal;
    text-transform: none;
  }
  .public-actions .hero-meta span + span {
    padding-left: 0;
    border-left: 0;
  }
  @media (max-width: 640px) {
    .public-actions {
      align-items: flex-start;
    }
  }
  .year-tabs {
    display: flex;
    align-items: center;
    flex-wrap: nowrap;
    gap: 0.45rem;
    max-width: 100%;
    overflow-x: auto;
    overscroll-behavior-x: contain;
    scrollbar-width: none;
    -webkit-overflow-scrolling: touch;
    padding: 0.25rem 0;
    margin-bottom: 0;
  }
  .year-tabs::-webkit-scrollbar {
    display: none;
  }
  .year-tabs button {
    flex-shrink: 0;
    padding: 0.42rem 0.9rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--surface);
    color: var(--text-muted);
    font-size: var(--type-label);
    font-weight: 600;
    cursor: pointer;
    transition:
      border-color 0.15s ease,
      background 0.15s ease,
      color 0.15s ease;
  }
  .year-tabs button:hover {
    border-color: var(--text);
    color: var(--text);
  }
  .year-tabs button:focus-visible,
  .year-tabs button:focus {
    outline: 2px solid var(--color-focus, var(--accent)) !important;
    outline-offset: 3px !important;
  }
  .year-tabs button.active {
    border-color: var(--tab-active-border);
    background: var(--tab-active-bg);
    color: var(--tab-active-text);
    font-weight: 750;
    box-shadow: var(--tab-active-shadow);
  }
  .dashboard {
    display: grid;
    gap: 1rem;
  }
  .analytics-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
  }
  .by-sport {
    min-width: 0;
  }
  .eyebrow,
  .section-kicker {
    margin-bottom: 0.65rem;
    color: var(--accent);
    font-size: var(--type-caption);
    font-weight: 750;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .sport-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    width: 100%;
    padding: 0.45rem 0.6rem;
    border-radius: 8px;
    border: none;
    background: none;
    cursor: pointer;
    transition: background 0.15s ease;
  }
  .sport-row:hover {
    background: color-mix(in srgb, var(--accent) 8%, var(--surface-2));
  }
  .sport-row .bar {
    flex: 1;
    height: 0.6rem;
    overflow: hidden;
    border-radius: 999px;
    background: var(--surface-2);
  }
  .sport-row .bar i {
    display: block;
    height: 100%;
    border-radius: 999px;
    box-shadow: 0 0 10px currentColor;
  }
  .sport-row .count {
    min-width: 3rem;
    text-align: right;
    color: var(--text-muted);
    font-size: var(--type-label);
  }
  .sport-row.active .count {
    color: var(--accent);
    font-weight: 700;
  }
  .clear-filter {
    margin-top: 0.4rem;
    border: none;
    background: none;
    color: var(--accent);
    font-size: var(--type-label);
    cursor: pointer;
    text-decoration: underline;
  }
  .section-heading {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 1rem;
    margin: 1.5rem 0 0.75rem;
  }
  .section-heading h2 {
    margin: 0;
    font-size: var(--type-title);
    letter-spacing: -0.02em;
  }
  @media (max-width: 768px) {
    .stat-grid {
      grid-template-columns: 1fr;
    }
    .analytics-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
