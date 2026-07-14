<script lang="ts">
  import {
    getBriefing,
    type DailyRow,
    type SleepSession,
    type Activity,
    type MediaHomeEvent,
    type BriefingResponse,
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

  let briefing = $state<BriefingResponse | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);

  // The selected day — the spine everything on this page snapshots to.
  let day = $state<string>(new Date().toISOString().slice(0, 10));
  let pickerOpen = $state(false);

  type BriefingList<T> = { items: T[]; has_more: boolean };
  function sectionData<T>(key: string): BriefingList<T> {
    const section = briefing?.sections.find((item) => item.key === key);
    return (
      (section?.data as BriefingList<T> | undefined) ?? {
        items: [],
        has_more: false,
      }
    );
  }

  const daily = $derived(sectionData<DailyRow>("daily"));
  const sleep = $derived(sectionData<SleepSession>("sleep"));
  const activities = $derived(sectionData<Activity>("activities"));
  const media = $derived(sectionData<MediaHomeEvent>("media"));
  const dRow = $derived(daily.items[0]);
  const nights = $derived(sleep.items);
  const mainNight = $derived(nights.find((n) => n.is_main_sleep) ?? nights[0]);
  const acts = $derived(activities.items);
  const mediaEvents = $derived(media.items);
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
  const dayHasData = $derived(
    briefing?.sections.some(
      (section) =>
        section.state === "ready" &&
        (section.data as { items?: unknown[] }).items?.length,
    ) ?? false,
  );
  const daysSet = $derived(new Set([day]));

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
  function num(v: number | null | undefined, digits: number): string {
    if (typeof v !== "number" || !Number.isFinite(v)) return "—";
    return v.toLocaleString(undefined, {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
    });
  }

  async function loadBriefing(selectedDay: string) {
    loading = true;
    error = null;
    try {
      const next = await getBriefing(selectedDay);
      if (selectedDay === day) briefing = next;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    void loadBriefing(day);
  });

  function mediaEventVerb(event: MediaHomeEvent): string {
    if (event.rating != null) return "Rated";
    if (event.progress_percent != null && event.progress_percent >= 100)
      return "Finished";
    if (event.position != null || event.progress_percent != null)
      return "Progressed";
    return "Updated library";
  }
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

      <!-- Media events: the selected-day slice of the media history. -->
      <div class="card tile wide media-card">
        <header>
          <span class="ic">▤</span>
          <a class="hdr-link" href="/media">Media</a>
        </header>
        {#if mediaEvents.length}
          <ul class="media-events">
            {#each mediaEvents as event (event.id)}
              <li>
                <a class="media-event-row" href={`/media/${event.media_id}`}>
                  {#if event.cover_image_url}
                    <img src={event.cover_image_url} alt="" loading="lazy" />
                  {:else}
                    <span class="media-thumb" aria-hidden="true"
                      >{event.title.slice(0, 1)}</span
                    >
                  {/if}
                  <span class="media-event-copy">
                    <strong>{event.title}</strong>
                    <span>{mediaEventVerb(event)}</span>
                  </span>
                  {#if event.rating != null}<span class="media-score"
                      >{event.rating.toFixed(1)}</span
                    >{/if}
                </a>
              </li>
            {/each}
          </ul>
        {:else}
          <p class="empty">No media events this day</p>
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
  .media-events {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 0.55rem;
  }
  .media-event-row {
    display: grid;
    grid-template-columns: 2.2rem minmax(0, 1fr) auto;
    align-items: center;
    gap: 0.65rem;
    min-width: 0;
    color: var(--text);
  }
  .media-event-row:hover {
    color: var(--accent);
    text-decoration: none;
  }
  .media-event-row img,
  .media-thumb {
    width: 2.2rem;
    height: 2.2rem;
    object-fit: cover;
    border-radius: 4px;
  }
  .media-thumb {
    display: grid;
    place-items: center;
    background: var(--surface-2);
    color: var(--accent);
    font-weight: 800;
  }
  .media-event-copy {
    min-width: 0;
    display: grid;
    gap: 0.15rem;
  }
  .media-event-copy strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 0.82rem;
  }
  .media-event-copy span {
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .media-score {
    color: var(--accent);
    font-size: 0.78rem;
    font-weight: 750;
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
