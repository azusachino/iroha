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

<section class="journal-today" aria-labelledby="journal-title">
  <header class="journal-opening">
    <div>
      <p class="journal-kicker">Entry {day}</p>
      <h1 id="journal-title">A day worth keeping.</h1>
      <p class="journal-date">{dayLabel}</p>
    </div>
    <div class="journal-stamp" aria-label="Imported evidence">
      <span>iroha</span>
      <strong>field<br />note</strong>
      <small>imported</small>
    </div>
  </header>

  <div class="journal-rule"><span>observations</span></div>

  <section class="signal-card">
    <div class="signal-copy">
      <p class="journal-kicker">The day in motion</p>
      <h2>{number(dRow?.steps)} steps, {number(dRow?.distance_km, 1)} km.</h2>
      <p>
        The record holds what happened without turning it into a verdict.
        Movement closed at {number(dRow?.move_kcal)} of
        {number(dRow?.move_goal_kcal)} kcal.
      </p>
    </div>
    <div
      class="signal-gauge"
      aria-label={`${Math.round(moveProgress)} percent of move goal`}
    >
      <svg viewBox="0 0 120 120" role="img" aria-hidden="true">
        <circle class="gauge-track" cx="60" cy="60" r="48" />
        <circle
          class="gauge-value"
          cx="60"
          cy="60"
          r="48"
          style={`stroke-dasharray: ${moveProgress * 3.02} 302`}
        />
      </svg>
      <strong>{Math.round(moveProgress)}%</strong>
      <span>move goal</span>
    </div>
  </section>

  <div class="journal-grid">
    <section class="journal-entry">
      <p class="journal-kicker">01 · body note</p>
      <h2>Small signals</h2>
      <dl class="signal-list">
        <div>
          <dt>Exercise</dt>
          <dd>{number(dRow?.exercise_min)} min</dd>
        </div>
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
      </dl>
    </section>

    <section class="journal-entry sleep-entry">
      <p class="journal-kicker">02 · night note</p>
      <h2>{mainNight ? formatDuration(mainNight.asleep_s) : "No entry"}</h2>
      {#if mainNight}
        <p>
          {Math.round(mainNight.efficiency * 100)}% of the night was asleep.
        </p>
        <div class="sleep-line">
          <i style={`width: ${mainNight.efficiency * 100}%`}></i>
        </div>
        <small
          >{formatDuration(mainNight.time_in_bed_s)} in bed · {mainNight.is_main_sleep
            ? "main sleep"
            : "short session"}</small
        >
      {:else}
        <p>No sleep session was recorded for this date.</p>
      {/if}
    </section>
  </div>

  <section class="session-entry">
    <div class="session-heading">
      <div>
        <p class="journal-kicker">03 · movement notes</p>
        <h2>Sessions remembered</h2>
      </div>
      <span>{acts.length} recorded</span>
    </div>
    {#if acts.length}
      <ol>
        {#each acts.slice(0, 6) as activity}
          <li>
            <span class="session-index"
              >{String(acts.indexOf(activity) + 1).padStart(2, "0")}</span
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
      <p class="journal-empty">
        No movement session was recorded for this date.
      </p>
    {/if}
  </section>

  <footer class="journal-source">
    <span>Source: imported snapshot</span>
    <span>Presentation only · no readiness score inferred</span>
  </footer>
</section>

<style>
  .journal-today {
    display: grid;
    gap: 2rem;
  }
  .journal-opening {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: start;
  }
  .journal-kicker {
    margin: 0 0 0.55rem;
    color: var(--accent);
    font-size: 0.68rem;
    letter-spacing: 0.16em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    font-family: var(--font-serif);
    font-weight: 400;
    letter-spacing: -0.04em;
  }
  h1 {
    max-width: 10ch;
    margin: 0;
    font-size: clamp(2.8rem, 7vw, 5.9rem);
    line-height: 0.92;
  }
  h2 {
    margin: 0.25rem 0 0.8rem;
    font-size: clamp(1.5rem, 3vw, 2.25rem);
  }
  .journal-date {
    margin: 1rem 0 0;
    color: var(--text-muted);
    font-family: var(--font-serif);
    font-size: 1.05rem;
  }
  .journal-stamp {
    display: grid;
    width: 7rem;
    aspect-ratio: 1;
    place-items: center;
    align-content: center;
    border: 1px solid var(--accent);
    border-radius: 50%;
    color: var(--accent);
    text-align: center;
    transform: rotate(8deg);
  }
  .journal-stamp span,
  .journal-stamp small {
    font-size: 0.55rem;
    letter-spacing: 0.15em;
    text-transform: uppercase;
  }
  .journal-stamp strong {
    margin: 0.25rem 0;
    font-family: var(--font-serif);
    font-size: 1.1rem;
    font-weight: 400;
    line-height: 0.9;
  }
  .journal-rule {
    display: flex;
    align-items: center;
    gap: 1rem;
    color: var(--text-muted);
    font-family: var(--font-serif);
    font-size: 0.8rem;
    font-style: italic;
  }
  .journal-rule::after {
    content: "";
    height: 1px;
    flex: 1;
    background: var(--border);
  }
  .signal-card,
  .journal-entry,
  .session-entry {
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface-1) 84%, transparent);
  }
  .signal-card {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 2rem;
    align-items: center;
    padding: clamp(1.25rem, 4vw, 3rem);
    border-color: color-mix(in srgb, var(--accent) 42%, var(--border));
  }
  .signal-copy h2 {
    max-width: 16ch;
  }
  .signal-copy p:last-child {
    max-width: 42rem;
    color: var(--text-muted);
    line-height: 1.7;
  }
  .signal-gauge {
    position: relative;
    width: 9rem;
    text-align: center;
  }
  .signal-gauge svg {
    display: block;
    transform: rotate(-90deg);
  }
  .signal-gauge circle {
    fill: none;
    stroke-width: 7;
  }
  .gauge-track {
    stroke: var(--border);
  }
  .gauge-value {
    stroke: var(--accent);
    stroke-linecap: round;
  }
  .signal-gauge strong,
  .signal-gauge span {
    position: absolute;
    left: 0;
    width: 100%;
    display: block;
  }
  .signal-gauge strong {
    top: 3.35rem;
    font-family: var(--font-serif);
    font-size: 1.55rem;
    font-weight: 400;
  }
  .signal-gauge span {
    top: 5.15rem;
    color: var(--text-muted);
    font-size: 0.62rem;
    text-transform: uppercase;
  }
  .journal-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 1rem;
  }
  .journal-entry {
    min-height: 15rem;
    padding: 1.5rem;
  }
  .signal-list {
    display: grid;
    gap: 0.85rem;
    margin: 1.5rem 0 0;
  }
  .signal-list div {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-bottom: 1px dotted var(--border);
    padding-bottom: 0.55rem;
  }
  dt,
  .sleep-entry p,
  .sleep-entry small {
    color: var(--text-muted);
  }
  dd {
    margin: 0;
    font-family: var(--font-serif);
  }
  .sleep-entry > p {
    line-height: 1.6;
  }
  .sleep-line {
    height: 0.35rem;
    margin: 1.5rem 0 0.7rem;
    background: var(--border);
  }
  .sleep-line i {
    display: block;
    height: 100%;
    background: var(--accent);
  }
  .session-entry {
    padding: 1.5rem;
  }
  .session-heading {
    display: flex;
    justify-content: space-between;
    align-items: end;
    gap: 1rem;
  }
  .session-heading > span {
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .session-entry ol {
    margin: 1rem 0 0;
    padding: 0;
    list-style: none;
  }
  .session-entry li {
    display: grid;
    grid-template-columns: 2.5rem minmax(0, 1fr) auto auto;
    gap: 1rem;
    align-items: baseline;
    border-top: 1px solid var(--border);
    padding: 0.9rem 0;
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  .session-index {
    color: var(--accent);
    font-family: var(--font-serif);
  }
  .session-name {
    color: var(--text);
    font-family: var(--font-serif);
    font-size: 1rem;
  }
  .journal-empty {
    color: var(--text-muted);
  }
  .journal-source {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding-top: 1rem;
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  @media (max-width: 680px) {
    .journal-opening,
    .signal-card,
    .session-heading,
    .journal-source {
      display: block;
    }
    .journal-stamp {
      margin-top: 1.5rem;
    }
    .signal-gauge {
      margin: 1.5rem auto 0;
    }
    .journal-grid {
      grid-template-columns: 1fr;
    }
    .session-entry li {
      grid-template-columns: 2rem minmax(0, 1fr);
    }
    .session-entry li > span:nth-child(n + 3) {
      grid-column: 2;
    }
  }
</style>
