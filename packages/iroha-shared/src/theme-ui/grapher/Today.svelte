<script lang="ts">
  import type { TodayThemeProps } from "../../today-view";
  import { formatDistance, formatDuration, formatPace } from "../../format";

  let {
    dayLabel,
    day,
    dRow,
    mainNight,
    acts,
    theme,
    mediaEvents,
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

  const points = $derived([
    {
      label: "Move",
      value: dRow?.ring?.move_kcal ?? null,
      unit: "kcal",
      color: "var(--ring-move)",
    },
    {
      label: "Exercise",
      value: dRow?.ring?.exercise_min ?? null,
      unit: "min",
      color: "var(--ring-exercise)",
    },
    {
      label: "Steps",
      value: dRow?.steps ?? null,
      unit: "steps",
      color: "var(--ring-stand)",
    },
  ]);

  const maxPoint = $derived(
    Math.max(1, ...points.map((point) => point.value ?? 0)),
  );
</script>

<section
  class="grapher-today"
  data-theme={theme}
  aria-labelledby="grapher-today-title"
>
  <header class="grapher-intro">
    <div>
      <p class="grapher-kicker">Daily data explorer / {day}</p>
      <h1 id="grapher-today-title">What did this day contain?</h1>
      <p>
        One imported day, rendered as comparable evidence rather than a score.
      </p>
    </div>
    <div class="view-switch" aria-label="Available data views">
      <span class="selected">Chart</span>
      <span>Table</span>
      <span>Notes</span>
    </div>
  </header>

  <div class="provenance-line">
    <span>Selected day</span><strong>{dayLabel}</strong>
    <span>Source</span><strong>Imported snapshot</strong>
    <span>Interpretation</span><strong
      >Raw and derived values shown separately</strong
    >
  </div>

  <section class="plot-panel" aria-labelledby="activity-chart-title">
    <div class="panel-heading">
      <div>
        <p class="grapher-kicker">Activity indicators</p>
        <h2 id="activity-chart-title">Movement across the day</h2>
      </div>
      <span class="panel-note">Values preserve source units</span>
    </div>
    <div
      class="plot"
      role="img"
      aria-label="Bar comparison of move calories, exercise minutes, and steps"
    >
      {#each points as point}
        <div class="plot-column">
          <div class="plot-value">
            {number(point.value)} <small>{point.unit}</small>
          </div>
          <div class="plot-track">
            <i
              style={`height: ${Math.max(3, ((point.value ?? 0) / maxPoint) * 100)}%; background: ${point.color}`}
            ></i>
          </div>
          <strong>{point.label}</strong>
        </div>
      {/each}
    </div>
  </section>

  <div class="data-grid">
    <section class="data-panel" aria-labelledby="recovery-title">
      <p class="grapher-kicker">Recovery indicator</p>
      <h2 id="recovery-title">Sleep efficiency</h2>
      {#if mainNight}
        <strong class="large-value"
          >{Math.round(mainNight.efficiency * 100)}%</strong
        >
        <dl>
          <div>
            <dt>Asleep</dt>
            <dd>{formatDuration(mainNight.asleep_s)}</dd>
          </div>
          <div>
            <dt>Time in bed</dt>
            <dd>{formatDuration(mainNight.time_in_bed_s)}</dd>
          </div>
        </dl>
      {:else}
        <p class="muted">No sleep session recorded for this day.</p>
      {/if}
    </section>

    <section class="data-panel" aria-labelledby="activity-record-title">
      <p class="grapher-kicker">Activity record</p>
      <h2 id="activity-record-title">Sessions</h2>
      {#if acts.length}
        <table>
          <thead><tr><th>Type</th><th>Distance</th><th>Duration</th></tr></thead
          >
          <tbody>
            {#each acts.slice(0, 5) as activity}
              <tr>
                <td>{activity.title || activity.sport_type}</td>
                <td>{formatDistance(activity.distance_m)}</td>
                <td
                  >{formatDuration(
                    activity.duration_s ?? activity.moving_time_s,
                  )}{#if activity.avg_pace_s_per_km}<small>
                      · {formatPace(activity.avg_pace_s_per_km)}</small
                    >{/if}</td
                >
              </tr>
            {/each}
          </tbody>
        </table>
      {:else}
        <p class="muted">No activity sessions recorded for this day.</p>
      {/if}
    </section>
  </div>

  <footer class="source-note">
    This view is a presentation of imported facts. It does not calculate a
    readiness score.
  </footer>
</section>

<style>
  .grapher-today {
    display: grid;
    gap: 1rem;
  }
  .grapher-intro {
    display: flex;
    justify-content: space-between;
    align-items: end;
    gap: 2rem;
    padding-bottom: 2rem;
    border-bottom: 3px solid var(--text);
  }
  .grapher-kicker {
    margin: 0 0 0.45rem;
    color: var(--accent);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    letter-spacing: -0.06em;
  }
  h1 {
    max-width: 15ch;
    font-size: clamp(2.8rem, 7vw, 6.5rem);
    line-height: 0.9;
  }
  h2 {
    font-size: 1.25rem;
  }
  .grapher-intro p:not(.grapher-kicker) {
    max-width: 35rem;
    margin: 0.85rem 0 0;
    color: var(--text-muted);
  }
  .view-switch {
    display: flex;
    gap: 0.8rem;
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .view-switch span {
    padding-bottom: 0.35rem;
    border-bottom: 1px solid transparent;
  }
  .view-switch .selected {
    border-color: var(--accent);
    color: var(--text);
  }
  .provenance-line {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: 0.25rem 0.6rem;
    padding: 0.75rem 0;
    border-bottom: 1px solid var(--border);
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .provenance-line strong {
    margin-right: 1rem;
    color: var(--text);
    font-weight: 650;
  }
  .plot-panel,
  .data-panel {
    padding: 1.25rem;
    border: 1px solid var(--border);
    background: var(--surface);
  }
  .panel-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .panel-note,
  .source-note {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .plot {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 2rem;
    height: 20rem;
    margin-top: 2rem;
    padding: 0 1rem;
    border-bottom: 1px solid var(--border);
  }
  .plot-column {
    display: grid;
    grid-template-rows: auto 1fr auto;
    gap: 0.5rem;
    min-width: 0;
    text-align: center;
  }
  .plot-value {
    font-size: 1rem;
    font-weight: 700;
  }
  .plot-value small,
  table small {
    color: var(--text-muted);
    font-size: 0.7em;
    font-weight: 400;
  }
  .plot-track {
    position: relative;
    display: flex;
    align-items: end;
    justify-content: center;
    min-height: 0;
    background: repeating-linear-gradient(
      to top,
      transparent 0 3.2rem,
      color-mix(in srgb, var(--border) 65%, transparent) 3.2rem 3.25rem
    );
  }
  .plot-track i {
    display: block;
    width: min(5rem, 70%);
    min-height: 0.35rem;
  }
  .plot-column strong {
    padding-top: 0.6rem;
    font-size: 0.74rem;
  }
  .data-grid {
    display: grid;
    grid-template-columns: minmax(0, 0.8fr) minmax(0, 1.2fr);
    gap: 1rem;
  }
  .large-value {
    display: block;
    margin: 1.5rem 0;
    color: var(--accent);
    font-size: clamp(3rem, 8vw, 6rem);
    letter-spacing: -0.1em;
    line-height: 0.8;
  }
  dl {
    display: grid;
    gap: 0.55rem;
    margin: 0;
  }
  dl div {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    padding-top: 0.55rem;
    border-top: 1px solid var(--border);
  }
  dt {
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  dd {
    margin: 0;
    font-size: 0.8rem;
    font-weight: 700;
  }
  table {
    width: 100%;
    margin-top: 1.2rem;
    border-collapse: collapse;
    font-size: 0.78rem;
  }
  th,
  td {
    padding: 0.65rem 0.35rem;
    border-bottom: 1px solid var(--border);
    text-align: left;
  }
  th {
    color: var(--text-muted);
    font-size: 0.65rem;
    font-weight: 650;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .muted {
    color: var(--text-muted);
    font-size: 0.85rem;
  }
  .source-note {
    padding: 0.8rem 0;
    border-top: 1px solid var(--border);
  }
  @media (max-width: 768px) {
    .grapher-intro,
    .panel-heading {
      align-items: start;
      flex-direction: column;
    }
    .data-grid {
      grid-template-columns: 1fr;
    }
    .plot {
      gap: 0.75rem;
      padding: 0;
    }
  }
</style>
