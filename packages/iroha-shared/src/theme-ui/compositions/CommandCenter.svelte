<script lang="ts">
  import {
    designActivitySummary,
    designDateLabel,
    designDistance,
    designDuration,
    designPercent,
    designSportLabel,
    designTimeLabel,
    type DesignCompositionProps,
  } from "../../theme/design-compositions";
  import DesignRingGauge from "./DesignRingGauge.svelte";

  let { today, readiness, links }: DesignCompositionProps = $props();
  const ring = $derived(today.daily?.ring);
  const sleep = $derived(today.sleep);
  const media = $derived(today.media[0]);
</script>

<section class="command-composition" aria-labelledby="command-title">
  <aside class="command-rail">
    <div class="rail-mark">i / 04</div>
    <p class="rail-label">Today</p>
    <strong class="rail-date">{designDateLabel(today.date)}</strong>
    <div class="rail-score">
      <span>Readiness</span>
      <strong>{readiness}</strong>
      <small>balanced signal</small>
    </div>
    <dl class="rail-facts">
      <div>
        <dt>Steps</dt>
        <dd>{today.daily?.steps?.toLocaleString() ?? "—"}</dd>
      </div>
      <div>
        <dt>Resting HR</dt>
        <dd>{today.daily?.resting_hr ?? "—"} bpm</dd>
      </div>
      <div>
        <dt>Distance</dt>
        <dd>{designDistance((today.daily?.distance_km ?? 0) * 1000)}</dd>
      </div>
    </dl>
    <span class="rail-hint">⌘K <small>jump anywhere</small></span>
  </aside>

  <div class="command-main">
    <header class="command-heading">
      <div>
        <p class="eyebrow">Overview / signal control</p>
        <h2 id="command-title">Keep the signal visible.</h2>
      </div>
      <a href={links.patterns}>Open patterns ↗</a>
    </header>

    <div class="command-kpis">
      <article>
        <span>Readiness</span><strong>{readiness}<small>/100</small></strong><i
          style={`--fill:${readiness}%`}
        ></i>
      </article>
      <article>
        <span>Move target</span><strong
          >{ring?.move_kcal ?? "—"}<small>
            / {ring?.move_goal_kcal ?? "—"} kcal</small
          ></strong
        ><i
          style={`--fill:${Math.min(100, ((ring?.move_kcal ?? 0) / (ring?.move_goal_kcal || 1)) * 100)}%`}
        ></i>
      </article>
      <article>
        <span>Resting heart</span><strong
          >{today.daily?.resting_hr ?? "—"}</strong
        ><small class="kpi-note">{today.daily?.hrv_sdnn ?? "—"} ms HRV</small>
      </article>
      <article>
        <span>Open threads</span><strong
          >{today.activities.length + today.media.length}</strong
        ><small class="kpi-note">activity + media</small>
      </article>
    </div>

    <div class="command-grid">
      <section class="command-panel rings-panel">
        <header>
          <span>Activity rings</span><a href={links.patterns}>Open daily →</a>
        </header>
        <DesignRingGauge {today} size={168} />
      </section>

      <section class="command-panel sleep-panel">
        <header><span>Sleep</span><a href={links.night}>Details →</a></header>
        <strong>{sleep ? designDuration(sleep.asleep_s) : "—"}</strong>
        <p>
          {sleep
            ? `${designPercent(sleep.efficiency * 100)} efficiency · ${designDuration(sleep.deep_s)} deep`
            : "No record"}
        </p>
        <div class="sleep-bar">
          <span style={`width:${Math.min(100, (sleep?.asleep_s ?? 0) / 288)}%`}
          ></span>
        </div>
      </section>

      <section class="command-panel stream-panel">
        <header>
          <span>Activity stream</span><a href={links.motion}>View all →</a>
        </header>
        <div class="activity-stream">
          {#each today.activities as activity (activity.id)}
            <a href={links.activity(activity.id)}>
              <span class="sport-mark"
                >{activity.sport_type.slice(0, 2).toUpperCase()}</span
              >
              <span
                ><strong>{activity.title}</strong><small
                  >{designTimeLabel(activity.started_at)} · {designActivitySummary(
                    activity,
                  )}</small
                ></span
              >
              <b>↗</b>
            </a>
          {:else}
            <p class="muted">No activity traces in this day.</p>
          {/each}
        </div>
      </section>

      <section class="command-panel vitals-panel">
        <header>
          <span>Vitals</span><a href={links.patterns}>Trends →</a>
        </header>
        <dl>
          <div>
            <dt>HRV</dt>
            <dd>{today.daily?.hrv_sdnn ?? "—"} ms</dd>
          </div>
          <div>
            <dt>SpO₂</dt>
            <dd>{today.daily?.spo2_avg ?? "—"}%</dd>
          </div>
          <div>
            <dt>VO₂max</dt>
            <dd>{today.daily?.vo2max?.toFixed(1) ?? "—"}</dd>
          </div>
        </dl>
      </section>

      <section class="command-panel next-panel">
        <header><span>Next move</span><em>low effort</em></header>
        <strong
          >{today.activities.length
            ? "Close the loop gently."
            : "Make the first mark."}</strong
        >
        <p>
          {ring?.exercise_min &&
          ring.exercise_goal_min &&
          ring.exercise_min >= ring.exercise_goal_min
            ? "Exercise is covered. A short walk keeps the evening open."
            : "A small block of movement will change the shape of the day."}
        </p>
        <a href={links.motion}>Plan an activity ↗</a>
      </section>

      <section class="command-panel media-panel">
        <header>
          <span>Recently touched</span><a href={links.library}>Open library →</a
          >
        </header>
        <strong>{media?.title ?? "No media signal yet."}</strong>
        <p>
          {media
            ? `${designPercent(media.progress_percent)} through · last touched today`
            : "Your library will appear here when something changes."}
        </p>
        <div class="media-progress">
          <span
            style={`width:${Math.min(100, Math.max(0, media?.progress_percent ?? 0))}%`}
          ></span>
        </div>
      </section>
    </div>

    <footer class="command-footer">
      <span
        >{designSportLabel(today.activities[0]?.sport_type ?? "quiet")} channel</span
      >
      <span>{today.date} · command surface</span>
    </footer>
  </div>
</section>

<style>
  .command-composition {
    display: grid;
    grid-template-columns: minmax(11rem, 15rem) minmax(0, 1fr);
    gap: 1rem;
    padding: clamp(1rem, 3vw, 2rem);
    background: var(--surface);
    color: var(--text);
    font-family: var(--font-sans, system-ui, sans-serif);
  }

  .command-rail,
  .command-panel {
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface-1) 94%, var(--accent));
  }

  .command-rail {
    display: grid;
    align-content: start;
    gap: 0.7rem;
    min-height: 34rem;
    padding: 1rem;
  }

  .rail-mark,
  .eyebrow,
  .rail-label,
  .rail-facts dt,
  .rail-hint,
  .command-kpis span,
  .command-panel header,
  .kpi-note,
  .command-footer {
    color: var(--text-muted);
    font-size: 0.66rem;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }

  .rail-mark {
    margin-bottom: 4rem;
    color: var(--accent);
    font-weight: 800;
  }

  .rail-label,
  .eyebrow {
    margin: 0;
  }

  .rail-date {
    font-size: 1.2rem;
    letter-spacing: -0.04em;
  }

  .rail-score {
    display: grid;
    gap: 0.2rem;
    margin-top: 1.5rem;
    border-top: 2px solid var(--accent);
    padding-top: 0.7rem;
  }

  .rail-score strong {
    color: var(--accent);
    font-size: 3.2rem;
    letter-spacing: -0.12em;
  }

  .rail-score span,
  .rail-score small {
    color: var(--text-muted);
    font-size: 0.7rem;
  }

  .rail-facts,
  .vitals-panel dl {
    display: grid;
    gap: 0.7rem;
    margin: 0;
  }

  .rail-facts {
    margin-top: 2rem;
  }

  .rail-facts div,
  .vitals-panel dl div {
    display: flex;
    justify-content: space-between;
    gap: 0.5rem;
    border-bottom: 1px solid var(--border);
    padding-bottom: 0.45rem;
  }

  .rail-facts dt,
  .rail-facts dd,
  .vitals-panel dt,
  .vitals-panel dd {
    margin: 0;
  }

  .rail-facts dd,
  .vitals-panel dd {
    font-size: 0.78rem;
    font-weight: 700;
    text-align: right;
  }

  .rail-hint {
    margin-top: auto;
    border-top: 1px solid var(--border);
    padding-top: 0.8rem;
  }

  .rail-hint small {
    letter-spacing: 0;
    text-transform: none;
  }

  .command-main {
    display: grid;
    gap: 1rem;
    min-width: 0;
  }

  .command-heading,
  .command-panel header {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
  }

  .command-heading {
    border-bottom: 2px solid var(--text);
    padding: 0.4rem 0 1rem;
  }

  h2,
  p {
    margin: 0;
  }

  h2 {
    margin-top: 0.35rem;
    font-size: clamp(1.7rem, 4vw, 3.5rem);
    letter-spacing: -0.08em;
  }

  a {
    color: var(--accent);
    text-decoration: none;
  }

  .command-heading a,
  .command-panel header a,
  .next-panel a {
    font-size: 0.72rem;
  }

  .command-kpis {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 0.6rem;
  }

  .command-kpis article {
    display: grid;
    gap: 0.35rem;
    min-width: 0;
    border: 1px solid var(--border);
    padding: 0.8rem;
    background: var(--surface-strong, var(--surface));
  }

  .command-kpis strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: clamp(1.35rem, 3vw, 2.2rem);
    letter-spacing: -0.08em;
  }

  .command-kpis strong small {
    color: var(--text-muted);
    font-size: 0.4em;
    letter-spacing: 0;
  }

  .command-kpis i,
  .sleep-bar,
  .media-progress {
    display: block;
    height: 0.25rem;
    overflow: hidden;
    background: var(--border);
  }

  .command-kpis i::after,
  .sleep-bar span,
  .media-progress span {
    display: block;
    width: var(--fill, 0%);
    height: 100%;
    background: var(--accent);
    content: "";
  }

  .command-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.7rem;
  }

  .command-panel {
    display: grid;
    align-content: start;
    gap: 0.9rem;
    min-width: 0;
    padding: 1rem;
  }

  .command-panel header {
    padding-bottom: 0.5rem;
    border-bottom: 1px solid var(--border);
  }

  .rings-panel {
    place-items: center;
  }

  .rings-panel header {
    width: 100%;
  }

  .sleep-panel > strong {
    font-size: clamp(1.8rem, 4vw, 3.4rem);
    letter-spacing: -0.1em;
  }

  .sleep-panel > p,
  .next-panel p,
  .media-panel p {
    color: var(--text-muted);
    font-size: 0.78rem;
    line-height: 1.5;
  }

  .activity-stream {
    display: grid;
  }

  .activity-stream a {
    display: grid;
    grid-template-columns: 2rem minmax(0, 1fr) auto;
    gap: 0.6rem;
    align-items: center;
    border-bottom: 1px solid var(--border);
    padding: 0.55rem 0;
    color: var(--text);
  }

  .activity-stream a:last-child {
    border-bottom: 0;
  }

  .sport-mark {
    display: grid;
    place-items: center;
    width: 1.8rem;
    height: 1.8rem;
    border-radius: 0.3rem;
    background: var(--accent);
    color: var(--bg);
    font-size: 0.58rem;
    font-weight: 800;
  }

  .activity-stream strong,
  .activity-stream small {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .activity-stream strong {
    font-size: 0.82rem;
  }

  .activity-stream small {
    margin-top: 0.2rem;
    color: var(--text-muted);
    font-size: 0.66rem;
  }

  .vitals-panel dl {
    margin-top: 0.4rem;
  }

  .vitals-panel dt {
    color: var(--text-muted);
    font-size: 0.76rem;
  }

  .next-panel {
    background: color-mix(in srgb, var(--accent) 9%, var(--surface-1));
  }

  .next-panel em {
    border: 1px solid var(--accent);
    border-radius: 999px;
    padding: 0.2rem 0.45rem;
    color: var(--accent);
    font-size: 0.62rem;
    font-style: normal;
    text-transform: uppercase;
  }

  .next-panel > strong,
  .media-panel > strong {
    font-size: 1.2rem;
    letter-spacing: -0.04em;
  }

  .command-footer {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding-top: 0.8rem;
  }

  @media (max-width: 1024px) {
    .command-composition {
      grid-template-columns: 1fr;
    }

    .command-rail {
      min-height: 0;
    }

    .rail-mark {
      margin-bottom: 0;
    }

    .rail-facts {
      grid-template-columns: repeat(3, 1fr);
      margin-top: 1rem;
    }
  }

  @media (max-width: 640px) {
    .command-kpis,
    .command-grid {
      grid-template-columns: 1fr;
    }

    .rail-facts {
      grid-template-columns: 1fr;
    }
  }
</style>
