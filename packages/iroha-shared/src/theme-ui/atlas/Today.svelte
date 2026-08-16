<script lang="ts">
  import type { TodayThemeProps } from "../../view-contracts/today-view";
  import { formatDistance, formatDuration, formatPace } from "../../format/format";
  import { mediaEventVerb } from "../../domain/media";
  import MediaUpdateList from "../components/MediaUpdateList.svelte";

  let {
    dayLabel,
    day,
    dRow,
    mainNight,
    acts,
    mediaEvents,
    mediaUpdates,
    theme,
    onOpenActivity,
    onOpenMedia,
  }: TodayThemeProps = $props();

  function number(value: number | null | undefined, digits = 0): string {
    if (typeof value !== "number" || !Number.isFinite(value)) return "—";
    return value.toLocaleString(undefined, {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
    });
  }

  const moveProgress = $derived(
    dRow?.ring && dRow.ring.move_goal_kcal > 0
      ? Math.min(100, (dRow.ring.move_kcal / dRow.ring.move_goal_kcal) * 100)
      : 0,
  );
</script>

<section
  class="atlas-today"
  data-theme={theme}
  aria-labelledby="atlas-today-title"
>
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
        Movement closed at {number(dRow?.ring?.move_kcal)} of {number(
          dRow?.ring?.move_goal_kcal,
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
          <dd>{number(dRow?.ring?.exercise_min)} min</dd>
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

  <section class="atlas-plate media-log">
    <header class="route-log-heading">
      <div>
        <p class="atlas-kicker">Media log</p>
        <h2>Media plotted today</h2>
      </div>
      <span>{mediaEvents.length} entries</span>
    </header>
    {#if mediaEvents.length}
      <ul class="atlas-media-list">
        {#each mediaEvents as event (event.id)}
          <li>
            <button
              class="atlas-media-row"
              type="button"
              onclick={() => onOpenMedia(event.media_id)}
            >
              {#if event.cover_image_url}
                <img src={event.cover_image_url} alt="" loading="lazy" />
              {:else}
                <span class="atlas-media-thumb" aria-hidden="true"
                  >{(event.native_title || event.title).slice(0, 1)}</span
                >
              {/if}
              <span class="atlas-media-copy">
                <strong>{event.native_title || event.title}</strong>
                <span>{mediaEventVerb(event)}</span>
              </span>
              {#if event.rating != null}<span class="atlas-media-score"
                  >{event.rating.toFixed(1)}</span
                >{/if}
            </button>
          </li>
        {/each}
      </ul>
    {:else}
      <p class="atlas-empty">No media events were logged for this date.</p>
    {/if}
  </section>

  {#if mediaUpdates.length}
    <section class="atlas-plate media-log">
      <header class="route-log-heading">
        <div>
          <p class="atlas-kicker">Provider dates</p>
          <h2>Library updates</h2>
        </div>
        <span>{mediaUpdates.length} updates</span>
      </header>
      <MediaUpdateList updates={mediaUpdates} {onOpenMedia} />
    </section>
  {/if}

  <footer class="atlas-source">
    Source: imported snapshot · presentation only, no readiness score inferred
  </footer>
</section>

<style>
  .atlas-today {
    display: grid;
    gap: 1.5rem;
    font-family: var(--font-sans);
    min-width: 0;
  }
  .atlas-today > * {
    min-width: 0;
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
  .atlas-media-list {
    display: grid;
    gap: 0.6rem;
    margin: 1.25rem 0 0;
    padding: 0;
    list-style: none;
  }
  .atlas-media-row {
    display: grid;
    grid-template-columns: 2.4rem minmax(0, 1fr) auto;
    align-items: center;
    gap: 0.7rem;
    min-width: 0;
    border: 0;
    padding: 0;
    background: transparent;
    color: var(--text);
    font: inherit;
    text-align: left;
    cursor: pointer;
  }
  .atlas-media-row:hover {
    color: var(--accent);
    text-decoration: none;
  }
  .atlas-media-row img,
  .atlas-media-thumb {
    width: 2.4rem;
    height: 2.4rem;
    border-radius: 4px;
    object-fit: cover;
  }
  .atlas-media-thumb {
    display: grid;
    place-items: center;
    background: var(--surface-2);
    color: var(--accent);
    font-weight: 800;
  }
  .atlas-media-copy {
    display: grid;
    gap: 0.15rem;
    min-width: 0;
  }
  .atlas-media-copy strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .atlas-media-copy span {
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .atlas-media-score {
    color: var(--accent);
    font-weight: 700;
  }
  .atlas-source {
    border-top: 1px solid var(--border);
    padding-top: 0.9rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.66rem;
  }
  @media (max-width: 768px) {
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
