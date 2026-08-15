<script lang="ts">
  import type { TodayThemeProps } from "../../today-view";
  import { formatDistance, formatDuration, formatPace } from "../../format";
  import { mediaEventVerb } from "../../media";
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
  const sleepProgress = $derived(
    mainNight ? Math.min(100, mainNight.efficiency * 100) : 0,
  );

  // One cycle instead of two gauges: motion closes around recovery, the
  // same relationship phenology studies between activity and rest.
  const OUTER_R = 50;
  const INNER_R = 35;
  const OUTER_C = 2 * Math.PI * OUTER_R;
  const INNER_C = 2 * Math.PI * INNER_R;
</script>

<section
  class="bloom-today"
  data-theme={theme}
  aria-labelledby="bloom-today-title"
>
  <header class="today-opening">
    <div>
      <p class="bloom-kicker">○ Day {day}</p>
      <h1 id="bloom-today-title">What unfolded today.</h1>
      <p class="today-date">{dayLabel}</p>
    </div>
    <div
      class="cycle-gauge"
      aria-label={`${Math.round(moveProgress)} percent of move goal, ${Math.round(sleepProgress)} percent sleep efficiency`}
    >
      <svg viewBox="0 0 120 120" role="img" aria-hidden="true">
        <circle class="ring-track" cx="60" cy="60" r={OUTER_R} />
        <circle
          class="ring-value ring-move"
          cx="60"
          cy="60"
          r={OUTER_R}
          style={`stroke-dasharray: ${(moveProgress / 100) * OUTER_C} ${OUTER_C}`}
        />
        <circle class="ring-track" cx="60" cy="60" r={INNER_R} />
        <circle
          class="ring-value ring-sleep"
          cx="60"
          cy="60"
          r={INNER_R}
          style={`stroke-dasharray: ${(sleepProgress / 100) * INNER_C} ${INNER_C}`}
        />
      </svg>
      <ul class="cycle-legend">
        <li><i class="dot-move"></i>Move · {Math.round(moveProgress)}%</li>
        <li><i class="dot-sleep"></i>Rest · {Math.round(sleepProgress)}%</li>
      </ul>
    </div>
  </header>

  <div class="bloom-grid">
    <section class="bloom-card">
      <p class="bloom-kicker">◔ Body signals</p>
      <h2>{number(dRow?.steps)} <small>steps</small></h2>
      <p class="bloom-note">
        {number(dRow?.distance_km, 1)} km covered · {number(
          dRow?.ring?.exercise_min,
        )} min exercise.
      </p>
      <dl>
        <div>
          <dt>Resting heart</dt>
          <dd>{number(dRow?.resting_hr)} bpm</dd>
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

    <section class="bloom-card">
      <p class="bloom-kicker">◑ Recovery cycle</p>
      <h2>
        {mainNight ? formatDuration(mainNight.asleep_s) : "No entry"}
      </h2>
      {#if mainNight}
        <p class="bloom-note">
          {Math.round(mainNight.efficiency * 100)}% of the night in bed was
          spent asleep.
        </p>
        <div class="cycle-line">
          <i style={`width: ${mainNight.efficiency * 100}%`}></i>
        </div>
        <small
          >{formatDuration(mainNight.time_in_bed_s)} in bed · {mainNight.is_main_sleep
            ? "main sleep"
            : "short session"}</small
        >
      {:else}
        <p class="bloom-note">No sleep session was recorded for this date.</p>
      {/if}
    </section>
  </div>

  <section class="bloom-sessions">
    <div class="sessions-heading">
      <div>
        <p class="bloom-kicker">◕ Movement notes</p>
        <h2>Sessions recorded</h2>
      </div>
      <span>{acts.length} entries</span>
    </div>
    {#if acts.length}
      <ol>
        {#each acts.slice(0, 6) as activity, index}
          <li>
            <span class="session-mark" aria-hidden="true"
              >{String(index + 1).padStart(2, "0")}</span
            >
            <span class="session-name"
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
      <p class="bloom-empty">No movement session was recorded for this date.</p>
    {/if}
  </section>

  <section class="bloom-sessions">
    <div class="sessions-heading">
      <div>
        <p class="bloom-kicker">◕ Media notes</p>
        <h2>Media touched today</h2>
      </div>
      <span>{mediaEvents.length} entries</span>
    </div>
    {#if mediaEvents.length}
      <ul class="bloom-media-list">
        {#each mediaEvents as event (event.id)}
          <li>
            <button
              class="bloom-media-row"
              type="button"
              onclick={() => onOpenMedia(event.media_id)}
            >
              {#if event.cover_image_url}
                <img src={event.cover_image_url} alt="" loading="lazy" />
              {:else}
                <span class="bloom-media-thumb" aria-hidden="true"
                  >{(event.native_title || event.title).slice(0, 1)}</span
                >
              {/if}
              <span class="bloom-media-copy">
                <strong>{event.native_title || event.title}</strong>
                <span>{mediaEventVerb(event)}</span>
              </span>
              {#if event.rating != null}<span class="bloom-media-score"
                  >{event.rating.toFixed(1)}</span
                >{/if}
            </button>
          </li>
        {/each}
      </ul>
    {:else}
      <p class="bloom-empty">No media event was recorded for this date.</p>
    {/if}
  </section>

  {#if mediaUpdates.length}
    <section class="bloom-sessions">
      <div class="sessions-heading">
        <div>
          <p class="bloom-kicker">◔ Provider dates</p>
          <h2>Library updates</h2>
        </div>
        <span>{mediaUpdates.length} updates</span>
      </div>
      <MediaUpdateList updates={mediaUpdates} {onOpenMedia} />
    </section>
  {/if}

  <footer class="bloom-source">
    <span>Source: imported daily snapshot</span>
    <span>Presentation only · no readiness score inferred</span>
  </footer>
</section>

<style>
  .bloom-today {
    display: grid;
    gap: 1.75rem;
    font-family: var(--font-serif);
    min-width: 0;
  }
  .bloom-today > * {
    min-width: 0;
  }
  .bloom-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.15em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    font-weight: 400;
    letter-spacing: -0.02em;
  }
  h1 {
    max-width: 12ch;
    font-size: clamp(2.6rem, 6.5vw, 5.4rem);
    line-height: 0.95;
  }
  h2 {
    font-size: 1.6rem;
  }
  .today-opening {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: center;
    padding-bottom: 1.75rem;
    border-bottom: 1px solid var(--border);
  }
  .today-date {
    margin: 1rem 0 0;
    color: var(--text-muted);
    font-style: italic;
  }
  .cycle-gauge {
    display: grid;
    justify-items: center;
    gap: 0.75rem;
  }
  .cycle-gauge svg {
    display: block;
    width: 8rem;
    height: 8rem;
    transform: rotate(-90deg);
  }
  .cycle-gauge circle {
    fill: none;
    stroke-width: 7;
  }
  .ring-track {
    stroke: var(--border);
  }
  .ring-value {
    stroke-linecap: round;
    transition: stroke-dasharray 0.3s ease;
  }
  .ring-move {
    stroke: var(--accent);
  }
  .ring-sleep {
    stroke: var(--accent-2);
  }
  .cycle-legend {
    display: flex;
    gap: 1rem;
    margin: 0;
    padding: 0;
    list-style: none;
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .cycle-legend li {
    display: flex;
    align-items: center;
    gap: 0.35rem;
  }
  .cycle-legend i {
    display: inline-block;
    width: 0.55rem;
    height: 0.55rem;
    border-radius: 50%;
  }
  .dot-move {
    background: var(--accent);
  }
  .dot-sleep {
    background: var(--accent-2);
  }
  .bloom-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 1.25rem;
  }
  .bloom-card,
  .bloom-sessions {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
  }
  .bloom-card {
    padding: 1.5rem;
  }
  .bloom-card h2 {
    margin-top: 0.4rem;
  }
  .bloom-card h2 small {
    font-style: normal;
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .bloom-note {
    margin: 0.6rem 0 0;
    color: var(--text-muted);
    line-height: 1.6;
  }
  dl {
    display: grid;
    gap: 0.75rem;
    margin: 1.4rem 0 0;
  }
  dl div {
    display: flex;
    justify-content: space-between;
    border-bottom: 1px dotted
      color-mix(in srgb, var(--accent-2) 55%, var(--border));
    padding-bottom: 0.5rem;
  }
  dt {
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  dd {
    margin: 0;
  }
  .cycle-line {
    height: 0.4rem;
    margin: 1.4rem 0 0.65rem;
    border-radius: 999px;
    background: var(--border);
    overflow: hidden;
  }
  .cycle-line i {
    display: block;
    height: 100%;
    border-radius: 999px;
    background: linear-gradient(90deg, var(--accent-2), var(--accent));
  }
  .bloom-card small {
    color: var(--text-muted);
  }
  .bloom-sessions {
    padding: 1.5rem;
  }
  .sessions-heading {
    display: flex;
    justify-content: space-between;
    align-items: end;
    gap: 1rem;
  }
  .sessions-heading > span {
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  ol {
    margin: 1rem 0 0;
    padding: 0;
    list-style: none;
  }
  li {
    display: grid;
    grid-template-columns: 2.5rem minmax(0, 1fr) auto auto;
    gap: 1rem;
    align-items: baseline;
    border-top: 1px solid var(--border);
    padding: 0.85rem 0;
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  .session-mark {
    display: inline-grid;
    place-items: center;
    width: 1.7rem;
    height: 1.7rem;
    border-radius: 50%;
    background: color-mix(in srgb, var(--accent) 14%, transparent);
    color: var(--accent);
    font-size: 0.68rem;
  }
  .session-name {
    color: var(--text);
    font-style: italic;
    font-size: 1rem;
  }
  .bloom-empty {
    margin-top: 1rem;
    color: var(--text-muted);
  }
  .bloom-media-list {
    display: grid;
    gap: 0.6rem;
    margin: 1rem 0 0;
    padding: 0;
    list-style: none;
  }
  .bloom-media-row {
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
  .bloom-media-row:hover {
    color: var(--accent);
    text-decoration: none;
  }
  .bloom-media-row img,
  .bloom-media-thumb {
    width: 2.4rem;
    height: 2.4rem;
    border-radius: 4px;
    object-fit: cover;
  }
  .bloom-media-thumb {
    display: grid;
    place-items: center;
    background: var(--surface-2);
    color: var(--accent);
    font-weight: 800;
  }
  .bloom-media-copy {
    display: grid;
    gap: 0.15rem;
    min-width: 0;
  }
  .bloom-media-copy strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .bloom-media-copy span {
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .bloom-media-score {
    color: var(--accent);
    font-weight: 700;
  }
  .bloom-source {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding-top: 0.9rem;
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  @media (max-width: 768px) {
    .today-opening,
    .bloom-grid,
    .sessions-heading,
    .bloom-source {
      display: block;
    }
    .cycle-gauge {
      margin-top: 1.5rem;
    }
    .bloom-grid {
      display: grid;
      grid-template-columns: 1fr;
    }
    .bloom-card + .bloom-card {
      margin-top: 1.25rem;
    }
    li {
      grid-template-columns: 2rem minmax(0, 1fr);
    }
    li > span:nth-child(n + 3) {
      grid-column: 2;
    }
  }
</style>
