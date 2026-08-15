<script lang="ts">
  import { onMount } from "svelte";
  import { replaceState } from "$app/navigation";
  import { page } from "$app/state";
  import {
    getDailyBounds,
    listDaily,
    listDailyAggregates,
    type DailyRow,
    type DailyAggregateBucket,
  } from "$lib/api";
  import RingGauge, {
    type Ring,
  } from "@iroha/shared/theme-ui/components/RingGauge.svelte";
  import DailySmallMultiples, {
    type SmallMultiple,
  } from "@iroha/shared/theme-ui/components/DailySmallMultiples.svelte";
  import PeriodSelector from "$lib/components/PeriodSelector.svelte";
  import PeriodToolbar from "$lib/components/PeriodToolbar.svelte";
  import {
    formatDateOnly,
    formatMonth as formatCanonicalMonth,
  } from "$lib/format";
  import { currentYear, yearOptionsInRange } from "@iroha/shared/month";
  import {
    currentCalendarScope,
    parseCalendarScope,
    readCalendarScope,
    scopeBounds,
    scopeFromParts,
    serializeCalendarScope,
    writeCalendarScope,
    type DateBounds,
  } from "@iroha/shared/scope";
  import { IROHA_TIMEZONE } from "$lib/config";
  import LoadingBoundary from "$lib/components/LoadingBoundary.svelte";
  import RouteIntro from "$lib/components/RouteIntro.svelte";
  import { useTheme } from "$lib/themes/context.svelte";
  import ThemeRouteRenderer from "@iroha/shared/theme-ui/ThemeRouteRenderer.svelte";
  import { hasThemeRoute } from "$lib/themes/registry";
  import { createAsyncResource } from "$lib/asyncResource.svelte";

  type Gran = "day" | "month" | "year";
  const monthlyResource = createAsyncResource<DailyAggregateBucket[]>();
  const yearlyResource = createAsyncResource<DailyAggregateBucket[]>();
  const daysResource = createAsyncResource<DailyRow[]>();
  const latestDayResource = createAsyncResource<DailyRow | null>();
  const monthly = $derived(monthlyResource.data ?? []);
  const yearly = $derived(yearlyResource.data ?? []);
  const dayRows = $derived(daysResource.data ?? []);
  const latestDay = $derived(latestDayResource.data ?? null);
  const error = $derived(
    monthlyResource.error ??
      latestDayResource.error ??
      yearlyResource.error ??
      daysResource.error ??
      null,
  );
  function granFromUrl(): Gran {
    const value = page.url.searchParams.get("gran");
    return value === "day" || value === "year" ? value : "month";
  }

  let gran = $state<Gran>(granFromUrl());
  // Set directly by drilling into a bar/row (see drillIntoPeriod), and reset
  // to "" -- meaning "default to the latest" -- by the day/month/year tabs
  // themselves, so a tab always means "zoom all the way out to this level."
  const defaultScope = currentCalendarScope("year", new Date(), IROHA_TIMEZONE);
  const requestedScope = readCalendarScope(page.url.searchParams, {
    fallback: defaultScope,
    allowDay: false,
  });
  const initialMonth =
    requestedScope.kind === "month"
      ? (serializeCalendarScope(requestedScope) as string)
      : "";
  const initialYear =
    requestedScope.kind === "year" || requestedScope.kind === "month"
      ? String(requestedScope.year)
      : requestedScope.kind === "lifetime"
        ? ""
        : currentYear(new Date(), IROHA_TIMEZONE);
  let selectedMonth = $state(initialMonth);
  let selectedYear = $state(initialYear);
  let rangeFrom = $state<string | undefined>(undefined);
  let rangeTo = $state<string | undefined>(undefined);
  let monthlyLoadedKey = "";
  let yearlyLoadedKey = "";
  let loadedDayMonth = "";
  const theme = useTheme();

  // The real data range (fetched once, independent of the current
  // selection/granularity) -- drives which years/months are navigable, as
  // a continuous range rather than only the periods with a chart bucket.
  // That's what lets `scopedYear`/`scopedMonth` below equal the real
  // selection whenever it's genuinely within history, instead of silently
  // substituting whatever period happens to have data.
  let dailyBounds = $state<DateBounds>({});
  const availableYears = $derived(yearOptionsInRange(dailyBounds));
  const availableMonths = $derived.by(() => {
    if (!dailyBounds.min || !dailyBounds.max) return [];
    const months: string[] = [];
    let year = Number(dailyBounds.max.slice(0, 4));
    let month = Number(dailyBounds.max.slice(5, 7));
    const minYear = Number(dailyBounds.min.slice(0, 4));
    const minMonth = Number(dailyBounds.min.slice(5, 7));
    while (year > minYear || (year === minYear && month >= minMonth)) {
      months.push(`${year}-${String(month).padStart(2, "0")}`);
      month -= 1;
      if (month === 0) {
        month = 12;
        year -= 1;
      }
    }
    return months;
  });
  const scopedYear = $derived(
    availableYears.includes(selectedYear) ? selectedYear : "",
  );
  const activeYear = $derived(scopedYear || availableYears[0] || "");
  const monthsInScope = $derived(
    availableMonths.filter((month) => month.startsWith(activeYear)),
  );
  const periodYears = $derived(
    availableYears.map((year) => ({ value: year, label: year })),
  );
  const periodMonths = $derived(
    monthsInScope.map((month) => ({
      value: month,
      label: formatCanonicalMonth(month),
    })),
  );
  const scopedMonth = $derived(
    monthsInScope.includes(selectedMonth) ? selectedMonth : "",
  );
  const activeMonth = $derived(scopedMonth || monthsInScope[0] || "");
  const periodLabel = $derived(
    gran === "year"
      ? scopedYear || "Lifetime"
      : gran === "month"
        ? scopedMonth || `${activeYear} · all months`
        : activeMonth
          ? formatCanonicalMonth(activeMonth)
          : "No month selected",
  );

  // Hero uses the latest real ring day, independent of the chosen granularity.
  const latestRingDay = $derived(latestDay);
  const latestRing = $derived(latestRingDay?.ring);
  const ringData = $derived<Ring[]>(
    latestRing
      ? [
          {
            label: "Move",
            value: latestRing.move_kcal,
            goal: latestRing.move_goal_kcal,
            unit: "kcal",
            color: "var(--ring-move)",
          },
          {
            label: "Exercise",
            value: latestRing.exercise_min,
            goal: latestRing.exercise_goal_min,
            unit: "min",
            color: "var(--ring-exercise)",
          },
          {
            label: "Stand",
            value: latestRing.stand_hours,
            goal: latestRing.stand_goal_hours,
            unit: "h",
            color: "var(--ring-stand)",
          },
        ]
      : [],
  );

  // A granularity-agnostic display row so the table + trends share one shape.
  interface Disp {
    label: string;
    // Raw, unformatted period ("2026-08-10" / "2026-08" / "2026") -- what
    // drillIntoPeriod acts on; label is display-only and not safe to parse.
    period: string;
    days: number | null;
    move: number | null;
    exercise: number | null;
    stand: number | null;
    moveClosedPct: number | null;
    steps: number | null;
    distance: number | null;
    resting_hr: number | null;
    hrv_sdnn: number | null;
    spo2_avg: number | null;
    respiratory_rate: number | null;
    vo2max: number | null;
    body_mass_kg: number | null;
  }

  function fmtPeriod(iso: string): string {
    const d = new Date(iso);
    if (gran === "year") return String(d.getUTCFullYear());
    return formatCanonicalMonth(iso.slice(0, 7));
  }
  function dayToDisp(r: DailyRow): Disp {
    const ring = r.ring;
    const hasRing = ring != null && ring.move_goal_kcal > 0;
    return {
      label: formatDateOnly(r.day),
      period: r.day.slice(0, 10),
      days: null,
      move: hasRing ? ring.move_kcal : null,
      exercise: hasRing ? ring.exercise_min : null,
      stand: hasRing ? ring.stand_hours : null,
      moveClosedPct: hasRing
        ? ring.move_kcal >= ring.move_goal_kcal
          ? 100
          : 0
        : null,
      steps: r.steps ?? null,
      distance: r.distance_km ?? null,
      resting_hr: r.resting_hr ?? null,
      hrv_sdnn: r.hrv_sdnn ?? null,
      spo2_avg: r.spo2_avg ?? null,
      respiratory_rate: r.respiratory_rate ?? null,
      vo2max: r.vo2max ?? null,
      body_mass_kg: r.body_mass_kg ?? null,
    };
  }
  function aggToDisp(b: DailyAggregateBucket): Disp {
    const metricValue = (metric: string): number | null =>
      b.metrics.find((item) => item.metric === metric)?.value ?? null;
    const move = b.move_kcal_avg === 0 ? null : b.move_kcal_avg;
    return {
      label: fmtPeriod(b.period),
      period: gran === "year" ? b.period.slice(0, 4) : b.period.slice(0, 7),
      days: b.days,
      move,
      exercise: b.exercise_min_avg === 0 ? null : b.exercise_min_avg,
      stand: b.stand_hours_avg === 0 ? null : b.stand_hours_avg,
      moveClosedPct: move == null ? null : Math.round(b.move_closed_pct),
      steps: metricValue("steps"),
      distance: metricValue("distance_km"),
      resting_hr: metricValue("resting_hr"),
      hrv_sdnn: metricValue("hrv_sdnn"),
      spo2_avg: metricValue("spo2_avg"),
      respiratory_rate: metricValue("respiratory_rate"),
      vo2max: metricValue("vo2max"),
      body_mass_kg: metricValue("body_mass_kg"),
    };
  }

  // Chronological (oldest→newest) for sparklines; table is its reverse.
  const chrono = $derived.by<Disp[]>(() => {
    if (gran === "day") {
      return [...dayRows]
        .filter((row) => !activeMonth || row.day.startsWith(activeMonth))
        .reverse()
        .map(dayToDisp);
    }
    if (gran === "month") {
      return monthly
        .filter(
          (bucket) =>
            (!activeYear || bucket.period.startsWith(activeYear)) &&
            (!scopedMonth || bucket.period.startsWith(scopedMonth)),
        )
        .map(aggToDisp);
    }
    return yearly
      .filter((bucket) => !scopedYear || bucket.period.startsWith(scopedYear))
      .map(aggToDisp);
  });
  const table = $derived([...chrono].reverse());
  const aggregated = $derived(gran !== "day");

  function ser(pick: (d: Disp) => number | null): (number | null)[] {
    return chrono.map((d) => pick(d));
  }
  const trendCharts = $derived.by<SmallMultiple[]>(() => [
    {
      label: "Steps",
      values: ser((d) => d.steps),
      color: "var(--accent)",
      unit: "steps",
    },
    {
      label: "Resting HR",
      values: ser((d) => d.resting_hr),
      color: "var(--sport-run)",
      unit: "bpm",
    },
    {
      label: "HRV",
      values: ser((d) => d.hrv_sdnn),
      color: "var(--sport-cycling)",
      unit: "ms",
    },
    {
      label: "Move ring closed",
      values: ser((d) => d.moveClosedPct),
      color: "var(--ring-move)",
      unit: "%",
    },
  ]);
  function fmt(v: number | null | undefined, digits: number): string {
    if (typeof v !== "number" || !Number.isFinite(v)) return "—";
    return v.toLocaleString(undefined, {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
    });
  }

  async function loadMonthly() {
    const key = selectedYear || "lifetime";
    if (monthlyLoadedKey === key) return;
    const result = await monthlyResource.run(() =>
      listDailyAggregates(
        "month",
        selectedYear ? { date: selectedYear } : {},
      ).then((r) => r.buckets),
    );
    if (result) monthlyLoadedKey = key;
  }

  async function loadBounds() {
    try {
      dailyBounds = await getDailyBounds();
    } catch {
      dailyBounds = {};
    }
  }

  async function loadLatestDay() {
    await latestDayResource.run(async () => {
      const result = await listDaily({ limit: 1 });
      return result.items[0] ?? null;
    });
  }

  async function loadYearly() {
    const key = selectedYear || "lifetime";
    if (yearlyLoadedKey === key) return;
    const result = await yearlyResource.run(() =>
      listDailyAggregates(
        "year",
        selectedYear ? { date: selectedYear } : {},
      ).then((r) => r.buckets),
    );
    if (result) yearlyLoadedKey = key;
  }

  async function loadDays(month: string) {
    if (!month || loadedDayMonth === month || daysResource.loading) return;
    const bounds = scopeBounds(parseCalendarScope(month)!);
    if (!bounds) return;
    rangeFrom = bounds.from;
    rangeTo = bounds.to;
    const result = await daysResource.run(async () => {
      const r = await listDaily({ date: month, limit: 31 });
      return r.items;
    });
    if (result) loadedDayMonth = month;
  }

  // A day/month/year tab always means "zoom all the way out to this level" --
  // reset any scope a bar/row click drilled into, rather than keeping it.
  async function changeGranularity(value: Gran) {
    gran = value;
    if (value === "year") await loadYearly();
    if (value === "day") await loadDays(activeMonth);
    syncUrl();
  }

  function selectYear(value: string) {
    selectedYear = value;
    selectedMonth = "";
    monthlyLoadedKey = "";
    yearlyLoadedKey = "";
    void loadMonthly();
    if (gran === "year") void loadYearly();
    if (gran === "day") void loadDays(activeMonth);
    syncUrl();
  }

  function selectMonth(value: string) {
    selectedMonth = value;
    if (value && selectedYear !== value.slice(0, 4)) {
      selectedYear = value.slice(0, 4);
      monthlyLoadedKey = "";
      yearlyLoadedKey = "";
      void loadMonthly();
      if (gran === "year") void loadYearly();
    }
    if (gran === "day") void loadDays(activeMonth);
    syncUrl();
  }

  function syncUrl() {
    const url = new URL(window.location.href);
    url.searchParams.set("gran", gran);
    writeCalendarScope(
      url.searchParams,
      scopedMonth
        ? (parseCalendarScope(scopedMonth) ?? scopeFromParts(scopedYear))
        : scopeFromParts(scopedYear),
    );
    if (url.search !== window.location.search) replaceState(url, page.state);
  }

  // Clicking a bar or table row zooms in one level: a year scopes month
  // view to it, a month scopes day view to it. Day is already the finest
  // granularity, so a day period has nothing further to drill into.
  function drillIntoPeriod(period: string) {
    if (gran === "year") {
      selectedYear = period;
      gran = "month";
    } else if (gran === "month") {
      selectedMonth = period;
      gran = "day";
      void loadDays(period);
    }
    syncUrl();
  }

  function drillIntoIndex(index: number) {
    const period = chrono[index]?.period;
    if (period) drillIntoPeriod(period);
  }

  onMount(async () => {
    await Promise.all([loadMonthly(), loadLatestDay(), loadBounds()]);
    if (gran === "year") await loadYearly();
    if (gran === "day") await loadDays(activeMonth);
  });

  // Only the resources the current granularity actually fetches -- yearly
  // and day-row resources are never run outside their own tab, so including
  // them unconditionally would leave the boundary waiting on data that's
  // never coming.
  const activeResources = $derived(
    gran === "year"
      ? [monthlyResource, latestDayResource, yearlyResource]
      : gran === "day"
        ? [monthlyResource, latestDayResource, daysResource]
        : [monthlyResource, latestDayResource],
  );
</script>

<svelte:head>
  <title>Patterns · iroha</title>
</svelte:head>

<section class="daily">
  {#if hasThemeRoute(theme.definition(), "daily")}
    <LoadingBoundary
      resource={activeResources}
      preserveLayout
      label="Loading time-series data…"
    >
      {#if error}
        <p class="error status" role="alert">
          Could not load daily data: {error}
        </p>
      {/if}
      <ThemeRouteRenderer
        route="daily"
        props={{
          chrono,
          gran,
          onGran: (value: Gran) => void changeGranularity(value),
          onDrillIndex: drillIntoIndex,
          onDrillPeriod: drillIntoPeriod,
          ringData,
          latestRingDay,
        }}
      >
        {#snippet children()}
          <PeriodToolbar title="Daily pattern scope" ariaLabel="Daily period">
            <PeriodSelector
              years={periodYears}
              months={periodMonths}
              year={gran === "year" ? selectedYear : activeYear}
              month={gran === "day" ? activeMonth : selectedMonth}
              bounds={dailyBounds}
              showAllYears={gran === "year"}
              surface="inline"
              onYear={selectYear}
              onMonth={selectMonth}
            />
          </PeriodToolbar>
        {/snippet}
      </ThemeRouteRenderer>
    </LoadingBoundary>
  {:else}
    <RouteIntro
      eyebrow="Patterns / personal history"
      title="Patterns & Vitals"
      description="Rings, movement, and body signals across your history. Start with the latest day, then zoom out to see the pattern."
      actionHref="/"
      actionLabel="Today"
    />

    <PeriodToolbar title="Daily pattern scope" ariaLabel="Daily period">
      <PeriodSelector
        years={periodYears}
        months={periodMonths}
        year={gran === "year" ? selectedYear : activeYear}
        month={gran === "day" ? activeMonth : selectedMonth}
        bounds={dailyBounds}
        showAllYears={gran === "year"}
        surface="inline"
        onYear={selectYear}
        onMonth={selectMonth}
      />
    </PeriodToolbar>

    {#if activeResources.some((r) => r.loading) && dayRows.length === 0}
      <p class="muted status">Loading daily history…</p>
    {:else if error}
      <p class="error status">Could not load daily data: {error}</p>
    {:else if dayRows.length === 0}
      <p class="muted status">No daily data imported yet.</p>
    {:else}
      <div class="hero tile glow">
        <div class="hero-head">
          <span class="hero-kicker">Latest rings</span>
          {#if latestRingDay}<span class="hero-date"
              >{formatDateOnly(latestRingDay.day)}</span
            >{/if}
        </div>
        <RingGauge rings={ringData} />
      </div>

      <div class="controls">
        <div class="seg" role="tablist" aria-label="Aggregation granularity">
          {#each ["day", "month", "year"] as const as g}
            <button
              role="tab"
              aria-selected={gran === g}
              class:active={gran === g}
              onclick={() => void changeGranularity(g)}
            >
              {g[0].toUpperCase() + g.slice(1)}
            </button>
          {/each}
        </div>
        <span class="muted small">
          {#if aggregated}
            {chrono.length}
            {gran}s in {gran === "year" ? "view" : activeYear} · per-day averages
          {:else}
            {chrono.length} days in {formatCanonicalMonth(activeMonth)}
          {/if}
        </span>
        {#if rangeFrom && rangeTo}<span class="muted small"
            >Range: {rangeFrom} → before {rangeTo}</span
          >{/if}
      </div>
      <div class="trend-panel tile">
        <div class="trend-heading">
          <div>
            <span class="t-label">Daily signals</span>
            <h2>Small multiples</h2>
          </div>
          <span class="muted small"
            >Move the crosshair to compare one period</span
          >
        </div>
        <DailySmallMultiples
          labels={chrono.map((d) => d.label)}
          charts={trendCharts}
        />
      </div>

      <section class="atlas-note tile" aria-label="Pattern reading guide">
        <div>
          <span class="t-label">Reading the atlas</span>
          <h2>{gran[0].toUpperCase() + gran.slice(1)} view</h2>
        </div>
        <p>
          Compare movement and body signals at this scale, then use the table as
          the precise record. Missing values remain unfilled rather than being
          treated as zero.
        </p>
        <span class="muted small">{chrono.length} periods in view</span>
      </section>

      <div class="table-wrap tile">
        <table>
          <thead>
            <tr>
              <th class="l"
                >{gran === "day"
                  ? "Day"
                  : gran === "month"
                    ? "Month"
                    : "Year"}</th
              >
              {#if aggregated}<th>Days</th>{/if}
              <th>Move</th><th>Exer</th><th>Stand</th><th>Move ✓</th>
              <th>Steps{aggregated ? "/d" : ""}</th><th
                >Dist{aggregated ? "/d" : ""}</th
              >
              <th>rHR</th><th>HRV</th><th>SpO₂</th><th>Resp</th><th>VO₂</th><th
                >Mass</th
              >
            </tr>
          </thead>
          <tbody>
            {#each table as d}
              <tr
                class:drillable={gran !== "day"}
                onclick={gran !== "day"
                  ? () => drillIntoPeriod(d.period)
                  : undefined}
              >
                <td class="l">{d.label}</td>
                {#if aggregated}<td>{d.days}</td>{/if}
                <td>{fmt(d.move, 0)}</td>
                <td>{fmt(d.exercise, 0)}</td>
                <td>{fmt(d.stand, 0)}</td>
                <td
                  >{d.moveClosedPct == null
                    ? "—"
                    : `${Math.round(d.moveClosedPct)}%`}</td
                >
                <td>{fmt(d.steps, 0)}</td>
                <td>{fmt(d.distance, 1)}</td>
                <td>{fmt(d.resting_hr, 0)}</td>
                <td>{fmt(d.hrv_sdnn, 0)}</td>
                <td>{fmt(d.spo2_avg, 1)}</td>
                <td>{fmt(d.respiratory_rate, 1)}</td>
                <td>{fmt(d.vo2max, 1)}</td>
                <td>{fmt(d.body_mass_kg, 1)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  {/if}
</section>

<style>
  .daily {
    display: grid;
    gap: 1.25rem;
  }
  .status {
    padding: 2rem 0;
  }
  .error {
    color: var(--danger);
  }
  .small {
    font-size: 0.78rem;
  }

  .hero {
    padding: 1.25rem;
    width: fit-content;
  }
  .glow {
    border: 1px solid color-mix(in srgb, var(--accent) 32%, var(--border));
    box-shadow: var(--accent-glow), var(--tile-shadow);
  }
  .hero-head {
    display: flex;
    align-items: baseline;
    gap: 1rem;
    justify-content: space-between;
    margin-bottom: 0.9rem;
  }
  .hero-kicker {
    color: var(--text-muted);
    font-size: 0.72rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .hero-date {
    color: var(--text);
    font-weight: 650;
  }

  .controls {
    display: flex;
    align-items: center;
    gap: 1rem;
    flex-wrap: wrap;
  }
  .seg {
    display: inline-flex;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    overflow: hidden;
  }
  .seg button {
    appearance: none;
    border: 0;
    background: var(--surface);
    color: var(--text-muted);
    padding: 0.45rem 0.9rem;
    font-size: 0.85rem;
    cursor: pointer;
  }
  .seg button + button {
    border-left: 1px solid var(--border);
  }
  .seg button.active {
    background: color-mix(in srgb, var(--accent) 18%, var(--surface));
    color: var(--accent);
    font-weight: 650;
  }

  .trend-panel {
    padding: 1rem;
  }

  .atlas-note {
    display: grid;
    grid-template-columns: minmax(12rem, 0.8fr) minmax(0, 1.6fr) auto;
    align-items: center;
    gap: 1rem;
    padding: 1rem 1.15rem;
    border-left: 3px solid var(--accent);
  }

  .atlas-note h2 {
    margin: 0.2rem 0 0;
    font-size: 1.05rem;
  }

  .atlas-note p {
    margin: 0;
    color: var(--text-muted);
    font-size: 0.86rem;
    line-height: 1.5;
  }
  .trend-heading {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    gap: 1rem;
  }
  .trend-heading h2 {
    margin: 0.2rem 0 0;
    font-size: 1rem;
  }
  .t-label {
    font-size: 0.78rem;
    color: var(--text-muted);
  }

  .table-wrap {
    padding: 0.4rem 0.4rem 0.6rem;
    overflow-x: auto;
  }
  table {
    width: 100%;
    border-collapse: collapse;
    font-variant-numeric: tabular-nums;
    font-size: 0.84rem;
  }
  th,
  td {
    padding: 0.5rem 0.6rem;
    text-align: right;
    white-space: nowrap;
  }
  th {
    color: var(--text-muted);
    font-weight: 600;
    border-bottom: 1px solid var(--border);
    position: sticky;
    top: 0;
    background: var(--surface);
  }
  td {
    color: var(--text);
  }
  tbody tr + tr td {
    border-top: 1px solid color-mix(in srgb, var(--border) 55%, transparent);
  }
  tbody tr.drillable {
    cursor: pointer;
  }
  tbody tr.drillable:hover td {
    background: color-mix(in srgb, var(--accent) 8%, transparent);
  }
  .l {
    text-align: left;
  }

  @media (max-width: 768px) {
    .atlas-note {
      grid-template-columns: 1fr;
    }

    .trend-heading {
      align-items: flex-start;
      flex-direction: column;
    }
  }
</style>
