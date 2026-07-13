<script lang="ts">
  import { onMount } from "svelte";
  import {
    listDaily,
    listSleep,
    listActivities,
    type DailyRow,
    type SleepSession,
    type Activity,
    type Page,
  } from "$lib/api";
  import RingGauge, { type Ring } from "$lib/components/RingGauge.svelte";
  import SportBadge from "$lib/components/SportBadge.svelte";
  import DayPicker from "$lib/components/DayPicker.svelte";
  import {
    formatDistance,
    formatDuration,
    formatPace,
    formatHr,
  } from "$lib/format";

  let dailyByDay = $state(new Map<string, DailyRow>());
  let sleepByDay = $state(new Map<string, SleepSession[]>());
  let actByDay = $state(new Map<string, Activity[]>());
  let daysWithData = $state<string[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  // The selected day — the spine everything on this page snapshots to.
  let day = $state<string>(new Date().toISOString().slice(0, 10));
  let pickerOpen = $state(false);

  async function sweep<T>(
    fn: (cursor?: string) => Promise<Page<T>>,
  ): Promise<T[]> {
    const out: T[] = [];
    let cursor: string | undefined;
    for (let i = 0; i < 30; i++) {
      const p = await fn(cursor);
      out.push(...p.items);
      if (!p.has_more || !p.next_cursor) break;
      cursor = p.next_cursor;
    }
    return out;
  }
  function group<T>(arr: T[], key: (t: T) => string): Map<string, T[]> {
    const m = new Map<string, T[]>();
    for (const t of arr) {
      const k = key(t);
      (m.get(k) ?? m.set(k, []).get(k)!).push(t);
    }
    return m;
  }

  const dRow = $derived(dailyByDay.get(day));
  const nights = $derived(sleepByDay.get(day) ?? []);
  const mainNight = $derived(nights.find((n) => n.is_main_sleep) ?? nights[0]);
  const acts = $derived(actByDay.get(day) ?? []);
  const hasRing = $derived(!!dRow && dRow.move_goal_kcal > 0);
  const ringData = $derived<Ring[]>(
    hasRing && dRow
      ? [
          {
            label: "Move",
            value: dRow.move_kcal,
            goal: dRow.move_goal_kcal,
            unit: "kcal",
            color: "var(--ring-move)",
          },
          {
            label: "Exercise",
            value: dRow.exercise_min,
            goal: dRow.exercise_goal_min,
            unit: "min",
            color: "var(--ring-exercise)",
          },
          {
            label: "Stand",
            value: dRow.stand_hours,
            goal: dRow.stand_goal_hours,
            unit: "h",
            color: "var(--ring-stand)",
          },
        ]
      : [],
  );

  const vitals = $derived(
    dRow
      ? [
          { l: "Resting HR", v: dRow.resting_hr, u: "bpm", d: 0 },
          { l: "HRV", v: dRow.hrv_sdnn, u: "ms", d: 0 },
          { l: "SpO₂", v: dRow.spo2_avg, u: "%", d: 1 },
          { l: "Respiratory", v: dRow.respiratory_rate, u: "/min", d: 1 },
          { l: "VO₂max", v: dRow.vo2max, u: "", d: 1 },
          { l: "Body mass", v: dRow.body_mass_kg, u: "kg", d: 1 },
        ].filter((m) => typeof m.v === "number")
      : [],
  );

  const dayLabel = $derived(
    new Date(day + "T00:00:00Z").toLocaleDateString(undefined, {
      weekday: "long",
      year: "numeric",
      month: "long",
      day: "numeric",
      timeZone: "UTC",
    }),
  );
  const isLatest = $derived(
    daysWithData.length > 0 && day === daysWithData[daysWithData.length - 1],
  );
  const dayHasData = $derived(
    dailyByDay.has(day) || sleepByDay.has(day) || actByDay.has(day),
  );
  const daysSet = $derived(new Set(daysWithData));
  const maxDay = $derived(daysWithData[daysWithData.length - 1]);

  function shift(delta: number) {
    // All in UTC: parsing local midnight then emitting toISOString (UTC)
    // silently dropped a day in +hh zones — hence "left = two days back".
    const d = new Date(day + "T00:00:00Z");
    d.setUTCDate(d.getUTCDate() + delta);
    day = d.toISOString().slice(0, 10);
  }
  // Arrow keys scrub days (ignored while typing in a field); Escape closes the picker.
  function onKey(e: KeyboardEvent) {
    const t = e.target as HTMLElement | null;
    if (
      t &&
      (t.tagName === "INPUT" || t.tagName === "TEXTAREA" || t.isContentEditable)
    )
      return;
    if (e.key === "ArrowLeft") shift(-1);
    else if (e.key === "ArrowRight") shift(1);
    else if (e.key === "Escape") pickerOpen = false;
  }
  function jumpLatest() {
    if (daysWithData.length) day = daysWithData[daysWithData.length - 1];
  }
  function num(v: number | null | undefined, digits: number): string {
    if (typeof v !== "number" || !Number.isFinite(v)) return "—";
    return v.toLocaleString(undefined, {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
    });
  }

  onMount(async () => {
    try {
      // 100 = the list endpoints' max page size (>100 is clamped to 50), so
      // this sweeps history in the fewest requests.
      const [daily, sleep, activities] = await Promise.all([
        sweep<DailyRow>((c) => listDaily({ limit: 100, cursor: c })),
        sweep<SleepSession>((c) => listSleep({ limit: 100, cursor: c })),
        sweep<Activity>((c) => listActivities({ limit: 100, cursor: c })),
      ]);
      dailyByDay = new Map(daily.map((r) => [r.day.slice(0, 10), r]));
      sleepByDay = group(sleep, (s) => s.wake_date.slice(0, 10));
      actByDay = group(activities, (a) => a.started_at.slice(0, 10));
      daysWithData = [
        ...new Set([
          ...dailyByDay.keys(),
          ...sleepByDay.keys(),
          ...actByDay.keys(),
        ]),
      ].sort();
      if (daysWithData.length) day = daysWithData[daysWithData.length - 1];
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  });
</script>

<svelte:window onkeydown={onKey} />

<section class="cockpit">
  <div class="scrubber tile glow">
    <button class="nav" aria-label="Previous day" onclick={() => shift(-1)}
      >‹</button
    >
    <button
      class="scrub-center"
      aria-haspopup="dialog"
      aria-expanded={pickerOpen}
      onclick={() => (pickerOpen = !pickerOpen)}
    >
      <span class="day-main">{dayLabel}</span>
      <span class="day-hint">tap to pick · ← → to scrub</span>
    </button>
    <button class="nav" aria-label="Next day" onclick={() => shift(1)}>›</button
    >
    {#if !isLatest}
      <button class="latest" onclick={jumpLatest}>Latest ›|</button>
    {/if}

    {#if pickerOpen}
      <button
        class="pk-backdrop"
        aria-label="Close picker"
        onclick={() => (pickerOpen = false)}
      ></button>
      <div class="pk-pop tile" role="dialog" aria-label="Pick a day">
        <DayPicker
          value={day}
          days={daysSet}
          max={maxDay}
          onselect={(d) => {
            day = d;
            pickerOpen = false;
          }}
        />
      </div>
    {/if}
  </div>

  {#if loading}
    <p class="muted status">Loading your history…</p>
  {:else if error}
    <p class="error status">Could not load data: {error}</p>
  {:else if !dayHasData}
    <p class="muted status">No data recorded for {dayLabel}.</p>
  {:else}
    <div class="bento">
      <!-- Rings -->
      <a class="card tile" href="/daily">
        <header><span class="ic">◎</span> Activity rings</header>
        {#if hasRing}
          <RingGauge rings={ringData} size={116} />
          <div class="mini-stats">
            <span>{num(dRow?.steps, 0)} steps</span>
            <span>{num(dRow?.distance_km, 1)} km</span>
            <span>{num(dRow?.flights, 0)} flights</span>
          </div>
        {:else}
          <p class="empty">No rings this day</p>
        {/if}
      </a>

      <!-- Vitals -->
      <a class="card tile" href="/daily">
        <header><span class="ic">♥</span> Body vitals</header>
        {#if vitals.length}
          <dl class="vitals">
            {#each vitals as m}
              <div>
                <dt>{m.l}</dt>
                <dd>{num(m.v, m.d)}<span class="u">{m.u}</span></dd>
              </div>
            {/each}
          </dl>
        {:else}
          <p class="empty">No vitals this day</p>
        {/if}
      </a>

      <!-- Sleep -->
      <a class="card tile" href="/sleep">
        <header><span class="ic">☾</span> Sleep</header>
        {#if mainNight}
          <div class="sleep-hero">{formatDuration(mainNight.asleep_s)}</div>
          <div class="mini-stats">
            <span>{Math.round(mainNight.efficiency * 100)}% efficiency</span>
            <span>{formatDuration(mainNight.time_in_bed_s)} in bed</span>
            <span>{mainNight.is_main_sleep ? "Main sleep" : "Nap"}</span>
          </div>
        {:else}
          <p class="empty">No sleep recorded</p>
        {/if}
      </a>

      <!-- Activities: each row links to its own detail page. -->
      <div class="card tile wide">
        <header>
          <span class="ic">⚡</span>
          <a class="hdr-link" href="/activities">Activities</a>
        </header>
        {#if acts.length}
          <ul class="acts">
            {#each acts as a}
              <li>
                <a class="act-row" href={`/activities/${a.id}`}>
                  <SportBadge sport={a.sport_type} />
                  <span class="a-title">{a.title || "Untitled"}</span>
                  <span class="a-metrics">
                    {#if a.distance_m}{formatDistance(a.distance_m)} ·
                    {/if}{formatDuration(
                      a.duration_s ?? a.moving_time_s,
                    )}{#if a.avg_pace_s_per_km}
                      · {formatPace(a.avg_pace_s_per_km)}{/if}{#if a.avg_hr}
                      · {formatHr(a.avg_hr)}{/if}
                  </span>
                </a>
              </li>
            {/each}
          </ul>
        {:else}
          <p class="empty">No activities this day</p>
        {/if}
      </div>

      <!-- Media placeholder -->
      <div class="card tile soon">
        <header><span class="ic">▤</span> Media</header>
        <p class="empty">Reading &amp; watching — coming soon</p>
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

  /* Date scrubber — the spine control, gets the neon chrome glow. */
  .scrubber {
    position: relative;
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.85rem 1rem;
  }
  .glow {
    border: 1px solid color-mix(in srgb, var(--accent) 34%, var(--border));
    box-shadow: var(--accent-glow), var(--tile-shadow);
  }
  .scrub-center {
    flex: 1;
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
    font-weight: 700;
    text-shadow: 0 0 16px color-mix(in srgb, var(--accent) 28%, transparent);
  }
  .day-hint {
    font-size: 0.72rem;
    color: var(--text-muted);
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
  .latest {
    appearance: none;
    border: 1px solid var(--border);
    background: var(--surface);
    color: var(--text-muted);
    padding: 0.35rem 0.6rem;
    border-radius: var(--radius);
    font-size: 0.78rem;
    cursor: pointer;
  }
  .latest:hover {
    color: var(--accent);
    border-color: var(--accent);
  }

  .bento {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 1rem;
  }
  .card {
    padding: 1.1rem;
    display: flex;
    flex-direction: column;
    gap: 0.85rem;
    color: var(--text);
    min-height: 11rem;
    overflow: hidden;
  }
  .card:hover {
    text-decoration: none;
    border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
  }
  .card.wide {
    grid-column: span 2;
  }
  .card.soon {
    opacity: 0.7;
  }
  .card header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.82rem;
    font-weight: 650;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  .ic {
    color: var(--accent);
    font-size: 1rem;
  }
  .empty {
    color: var(--text-muted);
    font-size: 0.88rem;
    margin: auto 0;
  }

  .mini-stats {
    display: flex;
    flex-wrap: wrap;
    gap: 0.35rem 1rem;
    color: var(--text-muted);
    font-size: 0.82rem;
  }
  .sleep-hero {
    font-size: 1.9rem;
    font-weight: 700;
    color: var(--accent);
    text-shadow: 0 0 18px color-mix(in srgb, var(--accent) 30%, transparent);
  }

  .vitals {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.55rem 1rem;
    margin: 0;
  }
  .vitals div {
    display: flex;
    flex-direction: column;
  }
  .vitals dt {
    font-size: 0.72rem;
    color: var(--text-muted);
  }
  .vitals dd {
    margin: 0;
    font-size: 1.05rem;
    font-weight: 650;
  }
  .vitals .u {
    font-size: 0.72rem;
    font-weight: 400;
    color: var(--text-muted);
    margin-left: 0.15rem;
  }

  .acts {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 0.6rem;
  }
  .acts li {
    min-width: 0;
  }
  .act-row {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr);
    align-items: center;
    gap: 0.3rem 0.6rem;
    min-width: 0;
    color: var(--text);
    padding: 0.2rem 0;
  }
  .act-row:hover {
    color: var(--accent);
    text-decoration: none;
  }
  .hdr-link {
    color: inherit;
  }
  .hdr-link:hover {
    color: var(--accent);
  }
  .a-title {
    min-width: 0;
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .a-metrics {
    grid-column: 2;
    min-width: 0;
    color: var(--text-muted);
    font-size: 0.8rem;
    overflow-wrap: anywhere;
  }

  @media (max-width: 820px) {
    .bento {
      grid-template-columns: 1fr;
    }
    .card.wide {
      grid-column: span 1;
    }
  }
</style>
