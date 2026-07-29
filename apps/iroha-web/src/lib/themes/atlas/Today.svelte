<script lang="ts">
  import type { Activity, DailyRow, SleepSession } from "$lib/api";
  import { formatDistance, formatDuration, formatPace } from "$lib/format";

  let {
    dayLabel,
    day,
    dRow,
    mainNight,
    acts,
  }: {
    dayLabel: string;
    day: string;
    dRow: DailyRow | undefined;
    mainNight: SleepSession | undefined;
    acts: Activity[];
  } = $props();

  function number(value: number | null | undefined, digits = 0): string {
    if (typeof value !== "number" || !Number.isFinite(value)) return "—";
    return value.toLocaleString(undefined, {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
    });
  }

  const moveProgress = $derived(
    dRow && dRow.move_goal_kcal > 0
      ? Math.min(100, (dRow.move_kcal / dRow.move_goal_kcal) * 100)
      : 0,
  );
</script>

<section class="atlas-today" aria-labelledby="atlas-today-title">
  <header class="today-header">
    <div>
      <p class="atlas-kicker">Sheet 001 · today's plot</p>
      <h1 id="atlas-today-title">Where the day went.</h1>
      <p class="today-sub">
        A position report, not a verdict — movement, rest, and vitals fixed to a
        single date.
      </p>
    </div>
    <div class="grid-ref" aria-label="Sheet reference">
      <span>{day}</span>
      <small>{dayLabel}</small>
    </div>
  </header>

  <section class="atlas-plate distance-plate">
    <div class="plate-copy">
      <p class="atlas-kicker">Ground covered</p>
      <h2>{number(dRow?.steps)} steps · {number(dRow?.distance_km, 1)} km</h2>
      <p>
        Movement closed at {number(dRow?.move_kcal)} of {number(
          dRow?.move_goal_kcal,
        )} kcal against the daily move goal.
      </p>
      <div
        class="scale-bar"
        aria-label={`${Math.round(moveProgress)} percent of move goal`}
      >
        <div class="scale-track">
          <i class="scale-fill" style={`width: ${moveProgress}%`}></i>
          <span class="scale-flag" style={`left: ${moveProgress}%`}
            >{Math.round(moveProgress)}%</span
          >
        </div>
        <div class="scale-ticks">
          <span>0</span><span>25</span><span>50</span><span>75</span><span
            >100%</span
          >
        </div>
      </div>
    </div>
  </section>

  <div class="today-grid">
    <section class="atlas-plate">
      <p class="atlas-kicker">Station readings</p>
      <h2>{number(dRow?.resting_hr)} <small>bpm resting</small></h2>
      <dl class="reading-list">
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
        <div>
          <dt>Exercise</dt>
          <dd>{number(dRow?.exercise_min)} min</dd>
        </div>
      </dl>
    </section>

    <section class="atlas-plate">
      <p class="atlas-kicker">Overnight bearing</p>
      <h2>
        {mainNight ? `${Math.round(mainNight.efficiency * 100)}%` : "—"}
        <small>sleep efficiency</small>
      </h2>
      {#if mainNight}
        <p>
          {formatDuration(mainNight.asleep_s)} asleep of {formatDuration(
            mainNight.time_in_bed_s,
          )} in bed.
        </p>
      {:else}
        <p>No sleep session was recorded for this sheet.</p>
      {/if}
    </section>
  </div>

  <section class="atlas-plate route-log">
    <header class="route-log-heading">
      <div>
        <p class="atlas-kicker">Route log</p>
        <h2>Sessions plotted</h2>
      </div>
      <span>{acts.length} waypoints</span>
    </header>
    {#if acts.length}
      <ol class="waypoint-list">
        {#each acts.slice(0, 6) as activity, index}
          <li>
            <span class="waypoint-index"
              >{String(index + 1).padStart(2, "0")}</span
            >
            <span class="waypoint-name"
              >{activity.title || activity.sport_type}</span
            >
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
      <p class="atlas-empty">No movement sessions were logged for this date.</p>
    {/if}
  </section>

  <footer class="atlas-source">
    Source: imported snapshot · presentation only, no readiness score inferred
  </footer>
</section>

<style>
  .atlas-today {
    display: grid;
    gap: 1.5rem;
    font-family: var(--font-sans);
  }
  .atlas-kicker {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.66rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .atlas-kicker::before {
    content: "⌖";
  }
  h1,
  h2 {
    margin: 0;
    font-weight: 600;
    letter-spacing: -0.03em;
  }
  h1 {
    max-width: 13ch;
    font-size: clamp(2.6rem, 6.5vw, 5.2rem);
    line-height: 0.98;
  }
  h2 {
    font-size: 1.5rem;
  }
  .today-header {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    padding-bottom: 1.75rem;
    border-bottom: 1px solid var(--border);
  }
  .today-sub {
    max-width: 40rem;
    margin: 1rem 0 0;
    color: var(--text-muted);
    line-height: 1.65;
  }
  .grid-ref {
    display: grid;
    justify-items: end;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    padding: 0.6rem 0.9rem;
    color: var(--accent);
    font-family: var(--font-mono);
    text-align: right;
  }
  .grid-ref span {
    font-size: 1.1rem;
  }
  .grid-ref small {
    margin-top: 0.2rem;
    color: var(--text-muted);
    font-size: 0.62rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  .atlas-plate {
    position: relative;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface-1) 88%, transparent);
    padding: 1.5rem;
  }
  .atlas-plate::before,
  .atlas-plate::after {
    content: "";
    position: absolute;
    width: 0.7rem;
    height: 0.7rem;
    border-color: var(--accent);
    opacity: 0.7;
  }
  .atlas-plate::before {
    top: -1px;
    left: -1px;
    border-top: 2px solid var(--accent);
    border-left: 2px solid var(--accent);
  }
  .atlas-plate::after {
    right: -1px;
    bottom: -1px;
    border-right: 2px solid var(--accent);
    border-bottom: 2px solid var(--accent);
  }
  .distance-plate {
    padding: clamp(1.5rem, 4vw, 2.5rem);
  }
  .plate-copy h2 {
    max-width: 20ch;
    margin-top: 0.5rem;
  }
  .plate-copy > p:not(.atlas-kicker) {
    max-width: 44rem;
    margin: 0.9rem 0 0;
    color: var(--text-muted);
    line-height: 1.65;
  }
  .scale-bar {
    margin-top: 1.75rem;
  }
  .scale-track {
    position: relative;
    height: 0.65rem;
    border: 1px solid var(--border);
    background: repeating-linear-gradient(
      90deg,
      color-mix(in srgb, var(--border) 70%, transparent) 0 1px,
      transparent 1px 10%
    );
  }
  .scale-fill {
    display: block;
    height: 100%;
    background: var(--accent);
  }
  .scale-flag {
    position: absolute;
    top: -1.55rem;
    transform: translateX(-50%);
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.72rem;
    white-space: nowrap;
  }
  .scale-ticks {
    display: flex;
    justify-content: space-between;
    margin-top: 0.4rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.62rem;
  }
  .today-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 1.25rem;
  }
  .today-grid h2 {
    margin-top: 0.75rem;
    font-size: 2rem;
  }
  .today-grid h2 small {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.75rem;
    font-weight: 400;
    letter-spacing: 0;
    text-transform: none;
  }
  .today-grid > .atlas-plate > p:last-child {
    margin-top: 0.6rem;
    color: var(--text-muted);
    line-height: 1.6;
  }
  .reading-list {
    display: grid;
    gap: 0.7rem;
    margin: 1.25rem 0 0;
  }
  .reading-list div {
    display: flex;
    justify-content: space-between;
    border-bottom: 1px dashed var(--border);
    padding-bottom: 0.5rem;
  }
  dt {
    color: var(--text-muted);
    font-size: 0.82rem;
  }
  dd {
    margin: 0;
    font-family: var(--font-mono);
  }
  .route-log-heading {
    display: flex;
    justify-content: space-between;
    align-items: end;
    gap: 1rem;
  }
  .route-log-heading > span {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.72rem;
  }
  .waypoint-list {
    margin: 1.25rem 0 0;
    padding: 0;
    list-style: none;
  }
  .waypoint-list li {
    display: grid;
    grid-template-columns: 2.1rem minmax(0, 1fr) auto auto;
    gap: 1rem;
    align-items: baseline;
    border-top: 1px solid var(--border);
    padding: 0.85rem 0;
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  .waypoint-index {
    display: inline-block;
    border: 1px solid var(--accent);
    border-radius: 50%;
    padding: 0.15rem 0;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.65rem;
    text-align: center;
  }
  .waypoint-name {
    color: var(--text);
    font-weight: 600;
  }
  .atlas-empty {
    margin-top: 1rem;
    color: var(--text-muted);
  }
  .atlas-source {
    border-top: 1px solid var(--border);
    padding-top: 0.9rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.66rem;
  }
  @media (max-width: 680px) {
    .today-header,
    .route-log-heading {
      display: block;
    }
    .grid-ref {
      margin-top: 1.5rem;
      justify-items: start;
      text-align: left;
    }
    .today-grid {
      grid-template-columns: 1fr;
    }
    .route-log-heading > span {
      display: block;
      margin-top: 0.5rem;
    }
    .waypoint-list li {
      grid-template-columns: 2.1rem minmax(0, 1fr);
    }
    .waypoint-list li > span:nth-child(n + 3) {
      grid-column: 2;
    }
  }
</style>
