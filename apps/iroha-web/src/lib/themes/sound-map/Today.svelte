<script lang="ts">
  import type {
    Activity,
    DailyRow,
    MediaHomeEvent,
    SleepSession,
  } from "$lib/api";
  import {
    formatDistance,
    formatDuration,
    formatPace,
    mediaEventVerb,
  } from "$lib/format";

  let {
    dayLabel,
    day,
    dRow,
    mainNight,
    acts,
    mediaEvents,
  }: {
    dayLabel: string;
    day: string;
    dRow: DailyRow | undefined;
    mainNight: SleepSession | undefined;
    acts: Activity[];
    mediaEvents: MediaHomeEvent[];
  } = $props();

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

  // The move goal is read as a segmented level meter rather than a smooth
  // fill bar -- the same discrete-step convention a hardware VU meter uses,
  // with the top segments switching to the peak color once the signal gets
  // hot. Twenty-four segments is a common LED meter count.
  const METER_SEGMENTS = 24;
  const litSegments = $derived(
    Math.round((moveProgress / 100) * METER_SEGMENTS),
  );
</script>

<section class="mix-today" aria-labelledby="mix-today-title">
  <header class="mix-head">
    <div>
      <p class="mix-kicker">Signal log / today</p>
      <h1 id="mix-today-title">The rhythm of {day}.</h1>
      <p>
        A private waveform built from the cadence of movement and rest, not a
        score.
      </p>
      <small>{dayLabel} · imported snapshot</small>
    </div>
    <div
      class="level-readout"
      aria-label={`${Math.round(moveProgress)} percent move goal`}
    >
      <div class="level-meter" role="img" aria-hidden="true">
        {#each Array(METER_SEGMENTS) as _, index (index)}
          <i
            class:lit={index < litSegments}
            class:peak={index >= METER_SEGMENTS - 5}
          ></i>
        {/each}
      </div>
      <span>{Math.round(moveProgress)}% MOVE</span>
    </div>
  </header>

  <div class="mix-grid">
    <section class="mix-card">
      <p class="mix-kicker">CH.1 · body signals</p>
      <h2>{number(dRow?.steps)}<small>steps</small></h2>
      <p class="mix-note">
        {number(dRow?.distance_km, 1)} km traveled · {number(
          dRow?.ring?.exercise_min,
        )} min exercise
      </p>
      <dl>
        <div>
          <dt>Resting</dt>
          <dd>{number(dRow?.resting_hr)} <small>bpm</small></dd>
        </div>
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

    <section class="mix-card">
      <p class="mix-kicker">CH.2 · recovery channel</p>
      <h2>
        {mainNight ? `${Math.round(mainNight.efficiency * 100)}%` : "—"}<small
          >efficiency</small
        >
      </h2>
      {#if mainNight}
        <p class="mix-note">
          {formatDuration(mainNight.asleep_s)} asleep · {formatDuration(
            mainNight.time_in_bed_s,
          )} in bed
        </p>
        <div class="mix-line">
          <i style={`width: ${mainNight.efficiency * 100}%`}></i>
        </div>
      {:else}
        <p class="mix-note">No sleep session was recorded for this date.</p>
      {/if}
    </section>
  </div>

  <section class="mix-sessions">
    <header>
      <div>
        <p class="mix-kicker">Track list</p>
        <h2>Sessions recorded</h2>
      </div>
      <span>{acts.length} entries</span>
    </header>
    {#if acts.length}
      <ol>
        {#each acts.slice(0, 6) as activity, index (activity.id)}
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
      <p class="mix-empty">No movement sessions were recorded for this date.</p>
    {/if}
  </section>

  <section class="mix-sessions">
    <header>
      <div>
        <p class="mix-kicker">Now playing</p>
        <h2>Media logged today</h2>
      </div>
      <span>{mediaEvents.length} entries</span>
    </header>
    {#if mediaEvents.length}
      <ul class="mix-media-list">
        {#each mediaEvents as event (event.id)}
          <li>
            <a class="mix-media-row" href={`/library/${event.media_id}`}>
              {#if event.cover_image_url}
                <img src={event.cover_image_url} alt="" loading="lazy" />
              {:else}
                <span class="mix-media-thumb" aria-hidden="true"
                  >{(event.native_title || event.title).slice(0, 1)}</span
                >
              {/if}
              <span class="mix-media-copy">
                <strong>{event.native_title || event.title}</strong>
                <span>{mediaEventVerb(event)}</span>
              </span>
              {#if event.rating != null}<span class="mix-media-score"
                  >{event.rating.toFixed(1)}</span
                >{/if}
            </a>
          </li>
        {/each}
      </ul>
    {:else}
      <p class="mix-empty">No media events were recorded for this date.</p>
    {/if}
  </section>

  <footer class="mix-source">
    <span>Source: imported raw evidence</span>
    <span>No derived readiness score</span>
  </footer>
</section>

<style>
  .mix-today {
    display: grid;
    gap: 1.35rem;
  }
  .mix-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  .mix-head {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    padding-bottom: 1.75rem;
    border-bottom: 1px solid var(--border);
  }
  h1,
  h2 {
    margin: 0;
    font-weight: 700;
    letter-spacing: -0.03em;
  }
  h1 {
    max-width: 13ch;
    font-size: clamp(2.3rem, 6vw, 4.6rem);
    line-height: 0.98;
    text-transform: uppercase;
  }
  h2 {
    font-size: 1.9rem;
  }
  .mix-head > div > p:not(.mix-kicker) {
    max-width: 38rem;
    margin: 0.9rem 0 0;
    color: var(--text-muted);
    line-height: 1.6;
  }
  .mix-head small {
    display: block;
    margin-top: 1rem;
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  .level-readout {
    display: grid;
    justify-items: end;
    gap: 0.6rem;
  }
  .level-meter {
    display: flex;
    align-items: flex-end;
    gap: 0.16rem;
    height: 3.2rem;
  }
  .level-meter i {
    display: block;
    width: 0.4rem;
    height: 100%;
    border-radius: 1px;
    background: color-mix(in srgb, var(--border) 70%, transparent);
  }
  .level-meter i.lit {
    background: var(--accent);
  }
  .level-meter i.lit.peak {
    background: var(--accent-2);
  }
  .level-readout > span {
    color: var(--text-muted);
    font-size: 0.68rem;
    letter-spacing: 0.08em;
  }
  .mix-card,
  .mix-sessions {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
  }
  .mix-card {
    padding: 1.4rem;
  }
  .mix-card h2 {
    margin-top: 0.5rem;
  }
  .mix-card h2 small,
  .mix-card dd small {
    font-weight: 400;
    color: var(--text-muted);
    font-size: 0.75rem;
    letter-spacing: 0;
    text-transform: none;
  }
  .mix-note {
    margin: 0.6rem 0 0;
    color: var(--text-muted);
    line-height: 1.6;
  }
  .mix-line {
    height: 0.4rem;
    margin-top: 1rem;
    border-radius: 1px;
    background: var(--border);
    overflow: hidden;
  }
  .mix-line i {
    display: block;
    height: 100%;
    background: linear-gradient(90deg, var(--accent), var(--accent-2));
  }
  .mix-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 1.25rem;
  }
  dl {
    display: grid;
    gap: 0.7rem;
    margin: 1.4rem 0 0;
  }
  dl div {
    display: flex;
    justify-content: space-between;
    border-bottom: 1px dashed var(--border);
    padding-bottom: 0.5rem;
  }
  dt {
    color: var(--text-muted);
    font-size: 0.72rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  dd {
    margin: 0;
    font-variant-numeric: tabular-nums;
  }
  .mix-sessions {
    padding: 1.4rem;
  }
  .mix-sessions header {
    display: flex;
    justify-content: space-between;
    align-items: end;
    gap: 1rem;
  }
  .mix-sessions header > span {
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
    align-items: baseline;
    border-top: 1px solid var(--border);
    padding: 0.85rem 0;
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  li b {
    color: var(--accent);
    font-weight: 700;
  }
  li strong {
    color: var(--text);
    font-size: 0.92rem;
    font-weight: 600;
  }
  .mix-empty {
    margin-top: 1rem;
    color: var(--text-muted);
  }
  .mix-media-list {
    display: grid;
    gap: 0.6rem;
    margin: 1rem 0 0;
    padding: 0;
    list-style: none;
  }
  .mix-media-row {
    display: grid;
    grid-template-columns: 2.4rem minmax(0, 1fr) auto;
    align-items: center;
    gap: 0.7rem;
    min-width: 0;
    color: var(--text);
  }
  .mix-media-row:hover {
    color: var(--accent);
    text-decoration: none;
  }
  .mix-media-row img,
  .mix-media-thumb {
    width: 2.4rem;
    height: 2.4rem;
    border-radius: 4px;
    object-fit: cover;
  }
  .mix-media-thumb {
    display: grid;
    place-items: center;
    background: var(--surface-2);
    color: var(--accent);
    font-weight: 800;
  }
  .mix-media-copy {
    display: grid;
    gap: 0.15rem;
    min-width: 0;
  }
  .mix-media-copy strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .mix-media-copy span {
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .mix-media-score {
    color: var(--accent);
    font-weight: 700;
  }
  .mix-source {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding-top: 0.9rem;
    color: var(--text-muted);
    font-size: 0.66rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  @media (max-width: 680px) {
    .mix-head,
    .mix-grid,
    .mix-sessions header,
    .mix-source {
      display: block;
    }
    .level-readout {
      justify-items: start;
      margin-top: 1.5rem;
    }
    .mix-grid {
      display: grid;
      grid-template-columns: 1fr;
    }
    .mix-card + .mix-card {
      margin-top: 1.25rem;
    }
    .mix-sessions header > span {
      display: block;
      margin-top: 0.6rem;
    }
    li {
      grid-template-columns: 2rem minmax(0, 1fr);
    }
    li > span {
      grid-column: 2;
    }
  }
</style>
