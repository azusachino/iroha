<script lang="ts">
  import type { Activity, DailyRow, SleepSession } from "$lib/api";
  import { formatDistance, formatDuration, formatPace } from "$lib/format";

  export type TodayVariant = "atlas" | "phenology" | "sound-map" | "archive";

  let {
    variant,
    dayLabel,
    day,
    dRow,
    mainNight,
    acts,
  }: {
    variant: TodayVariant;
    dayLabel: string;
    day: string;
    dRow: DailyRow | undefined;
    mainNight: SleepSession | undefined;
    acts: Activity[];
  } = $props();

  const copy = {
    atlas: {
      eyebrow: "Atlas sheet / field coordinates",
      title: "A day plotted in place.",
      note: "Movement is a route through time, even when no map was recorded.",
      mark: "N",
    },
    phenology: {
      eyebrow: "Phenology / daily bloom",
      title: "What opened today?",
      note: "Rest, motion, and recovery form a cycle rather than a score.",
      mark: "◒",
    },
    "sound-map": {
      eyebrow: "Sound map / signal trace",
      title: "The rhythm of one day.",
      note: "A private waveform built from the cadence of movement and rest.",
      mark: "≈",
    },
    archive: {
      eyebrow: "Archive card / daily record",
      title: "Filed under {day}.",
      note: "An indexed record of the body, its routes, and the hours between.",
      mark: "№",
    },
  } as const;

  const signal = $derived(
    dRow && dRow.move_goal_kcal > 0
      ? Math.min(100, (dRow.move_kcal / dRow.move_goal_kcal) * 100)
      : 0,
  );

  function number(value: number | null | undefined, digits = 0): string {
    if (typeof value !== "number" || !Number.isFinite(value)) return "—";
    return value.toLocaleString(undefined, {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
    });
  }
</script>

<section
  class={`theme-today theme-today-${variant}`}
  aria-labelledby="theme-today-title"
>
  <header class="theme-today-head">
    <div>
      <p class="theme-kicker">{copy[variant].eyebrow}</p>
      <h1 id="theme-today-title">
        {copy[variant].title.replace("{day}", day)}
      </h1>
      <p>{copy[variant].note}</p>
      <small>{dayLabel} · imported snapshot</small>
    </div>
    <div class="theme-today-mark" aria-hidden="true">{copy[variant].mark}</div>
  </header>

  <section class="theme-today-focus">
    <div>
      <p class="theme-kicker">Primary observation</p>
      <strong>{number(dRow?.steps)} <small>steps</small></strong>
      <p>
        {number(dRow?.distance_km, 1)} km traveled · {number(
          dRow?.exercise_min,
        )} min exercise
      </p>
    </div>
    <div
      class="signal-meter"
      aria-label={`${Math.round(signal)} percent move goal`}
    >
      <i style={`width: ${signal}%`}></i>
      <span>{Math.round(signal)}% move goal</span>
    </div>
  </section>

  <div class="theme-today-grid">
    <section class="theme-today-card">
      <p class="theme-kicker">Body signals</p>
      <h2>{number(dRow?.resting_hr)} <small>bpm resting</small></h2>
      <dl>
        <div>
          <dt>HRV</dt>
          <dd>{number(dRow?.hrv_sdnn)} ms</dd>
        </div>
        <div>
          <dt>Respiration</dt>
          <dd>{number(dRow?.respiratory_rate, 1)} /min</dd>
        </div>
        <div>
          <dt>Body mass</dt>
          <dd>{number(dRow?.body_mass_kg, 1)} kg</dd>
        </div>
      </dl>
    </section>

    <section class="theme-today-card">
      <p class="theme-kicker">Recovery signal</p>
      <h2>
        {mainNight ? `${Math.round(mainNight.efficiency * 100)}%` : "—"}
        <small>sleep efficiency</small>
      </h2>
      {#if mainNight}
        <p>
          {formatDuration(mainNight.asleep_s)} asleep · {formatDuration(
            mainNight.time_in_bed_s,
          )} in bed
        </p>
      {:else}
        <p>No sleep session was recorded.</p>
      {/if}
    </section>
  </div>

  <section class="theme-today-sessions">
    <header>
      <div>
        <p class="theme-kicker">Recorded movement</p>
        <h2>Sessions</h2>
      </div>
      <span>{acts.length} entries</span>
    </header>
    {#if acts.length}
      <ol>
        {#each acts.slice(0, 6) as activity, index}
          <li>
            <b>{String(index + 1).padStart(2, "0")}</b>
            <strong>{activity.title || activity.sport_type}</strong>
            <span>{formatDistance(activity.distance_m)}</span>
            <span
              >{formatDuration(
                activity.duration_s ?? activity.moving_time_s,
              )}{#if activity.avg_pace_s_per_km}
                · {formatPace(activity.avg_pace_s_per_km)}{/if}</span
            >
          </li>
        {/each}
      </ol>
    {:else}
      <p>No movement sessions were recorded for this date.</p>
    {/if}
  </section>

  <footer class="theme-today-source">
    Source: imported raw evidence · no derived readiness score
  </footer>
</section>

<style>
  .theme-today {
    display: grid;
    gap: 1.25rem;
  }
  .theme-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  .theme-today-head {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    padding-bottom: 2rem;
    border-bottom: 1px solid var(--border);
  }
  h1,
  h2 {
    margin: 0;
    font-family: Georgia, serif;
    font-weight: 400;
    letter-spacing: -0.05em;
  }
  h1 {
    max-width: 12ch;
    font-size: clamp(2.8rem, 7vw, 6.5rem);
    line-height: 0.88;
  }
  h2 {
    font-size: 1.7rem;
  }
  .theme-today-head p:not(.theme-kicker) {
    max-width: 38rem;
    margin: 1rem 0 0;
    color: var(--text-muted);
    line-height: 1.6;
  }
  .theme-today-head small {
    display: block;
    margin-top: 1rem;
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  .theme-today-mark {
    display: grid;
    width: 5rem;
    height: 5rem;
    place-items: center;
    border: 1px solid var(--accent);
    color: var(--accent);
    font-family: Georgia, serif;
    font-size: 2rem;
  }
  .theme-today-focus,
  .theme-today-card,
  .theme-today-sessions {
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface-1) 88%, transparent);
  }
  .theme-today-focus {
    display: grid;
    grid-template-columns: 1fr minmax(12rem, 0.8fr);
    gap: 2rem;
    align-items: center;
    padding: clamp(1.25rem, 4vw, 2.5rem);
  }
  .theme-today-focus strong {
    display: block;
    font-family: Georgia, serif;
    font-size: clamp(2.6rem, 7vw, 5rem);
    font-weight: 400;
    letter-spacing: -0.08em;
  }
  .theme-today-focus strong small,
  .theme-today-card h2 small {
    color: var(--text-muted);
    font-family: inherit;
    font-size: 0.8rem;
    letter-spacing: 0;
  }
  .theme-today-focus p:last-child,
  .theme-today-card > p:last-child,
  .theme-today-sessions > p {
    color: var(--text-muted);
  }
  .signal-meter {
    height: 1.1rem;
    border: 1px solid var(--border);
    padding: 0.2rem;
  }
  .signal-meter i {
    display: block;
    height: 100%;
    background: var(--accent);
  }
  .signal-meter span {
    display: block;
    margin-top: 0.55rem;
    color: var(--text-muted);
    font-size: 0.65rem;
    text-transform: uppercase;
  }
  .theme-today-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 1.25rem;
  }
  .theme-today-card,
  .theme-today-sessions {
    padding: 1.5rem;
  }
  .theme-today-card h2 {
    margin-top: 1rem;
    font-size: 2.5rem;
  }
  .theme-today-card > p {
    line-height: 1.6;
  }
  dl {
    display: grid;
    gap: 0.75rem;
    margin: 1.5rem 0 0;
  }
  dl div {
    display: flex;
    justify-content: space-between;
    border-bottom: 1px dotted var(--border);
    padding-bottom: 0.5rem;
  }
  dt {
    color: var(--text-muted);
  }
  dd {
    margin: 0;
    font-family: Georgia, serif;
  }
  .theme-today-sessions header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: end;
  }
  .theme-today-sessions header > span {
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  ol {
    margin: 1rem 0 0;
    padding: 0;
    list-style: none;
  }
  li {
    display: grid;
    grid-template-columns: 2rem minmax(0, 1fr) auto auto;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding: 0.85rem 0;
    align-items: baseline;
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  li b {
    color: var(--accent);
    font-family: Georgia, serif;
    font-weight: 400;
  }
  li strong {
    color: var(--text);
    font-family: Georgia, serif;
    font-size: 1rem;
    font-weight: 400;
  }
  .theme-today-source {
    border-top: 1px solid var(--border);
    padding-top: 0.9rem;
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  .theme-today-atlas {
    font-family: "Avenir Next", sans-serif;
  }
  .theme-today-atlas .theme-today-focus {
    border-left: 0.35rem solid var(--accent);
  }
  .theme-today-phenology .theme-today-card,
  .theme-today-phenology .theme-today-sessions {
    border-radius: 1.2rem 0.3rem 1.2rem 0.3rem;
  }
  .theme-today-phenology h1,
  .theme-today-phenology h2 {
    font-style: italic;
  }
  .theme-today-sound-map {
    font-family: "IBM Plex Mono", monospace;
  }
  .theme-today-sound-map h1,
  .theme-today-sound-map h2 {
    font-family: inherit;
    letter-spacing: -0.08em;
  }
  .theme-today-sound-map .theme-today-focus {
    background:
      repeating-linear-gradient(
        90deg,
        transparent 0 1rem,
        color-mix(in srgb, var(--accent) 8%, transparent) 1rem 1.05rem
      ),
      var(--surface-1);
  }
  .theme-today-archive {
    font-family: Georgia, serif;
  }
  .theme-today-archive .theme-today-card,
  .theme-today-archive .theme-today-sessions {
    box-shadow: 0.4rem 0.4rem 0
      color-mix(in srgb, var(--accent) 18%, transparent);
  }
  @media (max-width: 680px) {
    .theme-today-head,
    .theme-today-focus,
    .theme-today-grid {
      display: block;
    }
    .theme-today-mark {
      margin-top: 1.5rem;
    }
    .signal-meter {
      margin-top: 1.5rem;
    }
    .theme-today-card + .theme-today-card {
      margin-top: 1.25rem;
    }
    li {
      grid-template-columns: 2rem minmax(0, 1fr);
    }
    li > span {
      grid-column: 2;
    }
  }
</style>
