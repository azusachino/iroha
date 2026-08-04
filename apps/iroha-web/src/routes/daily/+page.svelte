<script lang="ts">
  import { onMount } from "svelte";
  import {
    listDaily,
    listDailyAggregates,
    type DailyRow,
    type DailyAggregateBucket,
  } from "$lib/api";
  import RingGauge, { type Ring } from "$lib/components/RingGauge.svelte";
  import DailyScopeControls from "$lib/components/DailyScopeControls.svelte";
  import DailySmallMultiples, {
    type SmallMultiple,
  } from "$lib/components/DailySmallMultiples.svelte";
  import { formatDateOnly } from "$lib/format";
  import RouteIntro from "$lib/components/RouteIntro.svelte";
  import { useTheme } from "$lib/themes/context.svelte";
  import ThemeRouteRenderer from "$lib/themes/ThemeRouteRenderer.svelte";
  import { hasThemeRoute } from "$lib/themes/registry";

  type Gran = "day" | "month" | "year";
  let dayRows = $state<DailyRow[]>([]);
  let latestDay = $state<DailyRow | null>(null);
  let monthly = $state<DailyAggregateBucket[]>([]);
  let yearly = $state<DailyAggregateBucket[]>([]);
  let loading = $state(true);
  let dayRowsLoading = $state(false);
  let error = $state<string | null>(null);
  let gran = $state<Gran>("month");
  let selectedMonth = $state("");
  let selectedYear = $state("");
  let rangeFrom = $state<string | undefined>(undefined);
  let rangeTo = $state<string | undefined>(undefined);
  let monthlyLoaded = false;
  let yearlyLoaded = false;
  let loadedDayMonth = "";
  const theme = useTheme();

  const availableMonths = $derived(
    monthly
      .map((bucket) => bucket.period.slice(0, 7))
      .sort()
      .reverse(),
  );
  const availableYears = $derived(
    [...new Set(monthly.map((bucket) => bucket.period.slice(0, 4)))]
      .sort()
      .reverse(),
  );
  const activeMonth = $derived(selectedMonth || availableMonths[0] || "");
  const activeYear = $derived(selectedYear || availableYears[0] || "");
  const monthOptions = $derived(
    availableMonths.map((value) => ({
      value,
      label: new Date(`${value}-01T00:00:00Z`).toLocaleDateString(undefined, {
        month: "long",
        year: "numeric",
        timeZone: "UTC",
      }),
    })),
  );
  const yearOptions = $derived(
    availableYears.map((value) => ({ value, label: value })),
  );

  // Hero uses the latest real ring day, independent of the chosen granularity.
  const latestRingDay = $derived(latestDay);
  const ringData = $derived<Ring[]>(
    latestRingDay
      ? [
          {
            label: "Move",
            value: latestRingDay.move_kcal,
            goal: latestRingDay.move_goal_kcal,
            unit: "kcal",
            color: "var(--ring-move)",
          },
          {
            label: "Exercise",
            value: latestRingDay.exercise_min,
            goal: latestRingDay.exercise_goal_min,
            unit: "min",
            color: "var(--ring-exercise)",
          },
          {
            label: "Stand",
            value: latestRingDay.stand_hours,
            goal: latestRingDay.stand_goal_hours,
            unit: "h",
            color: "var(--ring-stand)",
          },
        ]
      : [],
  );

  // A granularity-agnostic display row so the table + trends share one shape.
  interface Disp {
    label: string;
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
    return d.toLocaleDateString(undefined, {
      year: "numeric",
      month: "short",
      timeZone: "UTC",
    });
  }
  function dayToDisp(r: DailyRow): Disp {
    const ring = r.move_goal_kcal > 0;
    return {
      label: formatDateOnly(r.day),
      days: null,
      move: ring ? r.move_kcal : null,
      exercise: ring ? r.exercise_min : null,
      stand: ring ? r.stand_hours : null,
      moveClosedPct: ring ? (r.move_kcal >= r.move_goal_kcal ? 100 : 0) : null,
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
    const m = b.metrics ?? {};
    const move = b.move_kcal_avg || null;
    return {
      label: fmtPeriod(b.period),
      days: b.days,
      move,
      exercise: b.exercise_min_avg || null,
      stand: b.stand_hours_avg || null,
      moveClosedPct: move == null ? null : Math.round(b.move_closed_pct),
      steps: m.steps ?? null,
      distance: m.distance_km ?? null,
      resting_hr: m.resting_hr ?? null,
      hrv_sdnn: m.hrv_sdnn ?? null,
      spo2_avg: m.spo2_avg ?? null,
      respiratory_rate: m.respiratory_rate ?? null,
      vo2max: m.vo2max ?? null,
      body_mass_kg: m.body_mass_kg ?? null,
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
        .filter((bucket) => !activeYear || bucket.period.startsWith(activeYear))
        .map(aggToDisp);
    }
    return yearly.map(aggToDisp);
  });
  const table = $derived([...chrono].reverse());
  const aggregated = $derived(gran !== "day");

  function ser(pick: (d: Disp) => number | null): number[] {
    return chrono.map((d) => pick(d) ?? Number.NaN);
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
    if (monthlyLoaded) return;
    try {
      const result = await listDailyAggregates("month");
      monthly = result.buckets;
      monthlyLoaded = true;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    }
  }

  async function loadLatestDay() {
    try {
      const result = await listDaily({ limit: 1 });
      latestDay = result.items[0] ?? null;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    }
  }

  async function loadYearly() {
    if (yearlyLoaded) return;
    try {
      const result = await listDailyAggregates("year");
      yearly = result.buckets;
      yearlyLoaded = true;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    }
  }

  async function loadDays(month: string) {
    if (!month || loadedDayMonth === month || dayRowsLoading) return;
    const [year, monthNumber] = month.split("-").map(Number);
    const lastDay = new Date(Date.UTC(year, monthNumber, 0)).getUTCDate();
    rangeFrom = `${month}-01`;
    rangeTo = `${month}-${String(lastDay).padStart(2, "0")}`;
    dayRowsLoading = true;
    try {
      const result = await listDaily({
        from: rangeFrom,
        to: rangeTo,
        limit: 31,
      });
      dayRows = result.items;
      loadedDayMonth = month;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      dayRowsLoading = false;
    }
  }

  async function changeGranularity(value: Gran) {
    gran = value;
    if (value === "year") await loadYearly();
    if (value === "day") await loadDays(activeMonth);
  }

  function changeDayMonth(value: string) {
    selectedMonth = value;
    void loadDays(value);
  }

  onMount(async () => {
    await Promise.all([loadMonthly(), loadLatestDay()]);
    loading = false;
  });
</script>

<section class="daily">
  {#if hasThemeRoute(theme.definition(), "daily")}
    {#if loading}
      <p class="muted status">Loading time-series data…</p>
    {:else if error}
      <p class="error status">Could not load daily data: {error}</p>
    {:else if monthly.length === 0 && dayRows.length === 0}
      <p class="muted status">No daily data imported yet.</p>
    {:else}
      {#if gran === "day"}
        <DailyScopeControls
          label="Month"
          options={monthOptions}
          value={activeMonth}
          summary={dayRowsLoading ? "Loading…" : `${chrono.length} days`}
          onChange={changeDayMonth}
        />
      {:else if gran === "month"}
        <DailyScopeControls
          label="Year"
          options={yearOptions}
          value={activeYear}
          summary={`${chrono.length} months`}
          onChange={(value) => (selectedYear = value)}
        />
      {/if}
      <ThemeRouteRenderer
        route="daily"
        props={{
          chrono,
          gran,
          onGran: (value: Gran) => void changeGranularity(value),
        }}
      />
    {/if}
  {:else}
    <RouteIntro
      eyebrow="Patterns / personal history"
      title="Daily & Vitals"
      description="Rings, movement, and body signals across your history. Start with the latest day, then zoom out to see the pattern."
      actionHref="/"
      actionLabel="Today"
    />

    {#if loading}
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

      {#if gran === "day"}
        <DailyScopeControls
          label="Month"
          options={monthOptions}
          value={activeMonth}
          summary={`${chrono.length} days`}
          onChange={changeDayMonth}
        />
      {:else if gran === "month"}
        <DailyScopeControls
          label="Year"
          options={yearOptions}
          value={activeYear}
          summary={`${chrono.length} months`}
          onChange={(value) => (selectedYear = value)}
        />
      {/if}

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
            {chrono.length} {gran}s in selected year · per-day averages
          {:else}
            {chrono.length} days in selected month
          {/if}
        </span>
        {#if rangeFrom && rangeTo}<span class="muted small"
            >Range: {rangeFrom} → {rangeTo}</span
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
              <tr>
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
  .l {
    text-align: left;
  }

  @media (max-width: 720px) {
    .atlas-note {
      grid-template-columns: 1fr;
    }

    .trend-heading {
      align-items: flex-start;
      flex-direction: column;
    }
  }
</style>
