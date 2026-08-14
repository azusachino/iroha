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
  } from "../../design-compositions";

  let { today, readiness, links }: DesignCompositionProps = $props();
  const moments = $derived(
    (today.sleep ? 1 : 0) + today.activities.length + today.media.length,
  );
</script>

<section class="chronicle-composition" aria-labelledby="chronicle-title">
  <header class="chronicle-heading">
    <div>
      <p class="eyebrow">{designDateLabel(today.date)}</p>
      <h2 id="chronicle-title">A day, in sequence.</h2>
      <p class="intro">
        A chronological record that makes the day feel lived in, not measured.
      </p>
    </div>
    <div class="moment-count">
      <strong>{moments}</strong><span>moments kept</span>
    </div>
  </header>

  <div class="chronicle-layout">
    <div class="timeline">
      {#if today.sleep}
        <article class="moment">
          <time>07:04</time><span class="pin recovery">◐</span>
          <div class="moment-card">
            <span class="card-label">Recovery</span>
            <h3>The night held.</h3>
            <p>
              {designDuration(today.sleep.asleep_s)} asleep · {designPercent(
                today.sleep.efficiency * 100,
              )} efficiency
            </p>
            <i style={`--fill:${today.sleep.efficiency * 100}%`}></i>
          </div>
        </article>
      {/if}
      {#each today.activities as activity (activity.id)}
        <article class="moment">
          <time>{designTimeLabel(activity.started_at)}</time><span
            class="pin movement">↗</span
          >
          <a class="moment-card" href={links.activity(activity.id)}
            ><span class="card-label"
              >{designSportLabel(activity.sport_type)}</span
            >
            <h3>{activity.title}</h3>
            <p>
              {designActivitySummary(activity)} · {activity.avg_hr ?? "—"} bpm
            </p>
            <strong>Open activity →</strong></a
          >
        </article>
      {:else}
        <p class="empty">No movement moments recorded.</p>
      {/each}
      {#each today.media as item (item.id)}
        <article class="moment">
          <time>{designTimeLabel(today.date + "T21:05:00")}</time><span
            class="pin library">▧</span
          >
          <a class="moment-card" href={links.library}
            ><span class="card-label">Library</span>
            <h3>{item.title}</h3>
            <p>{designPercent(item.progress_percent)} complete</p>
            <i style={`--fill:${item.progress_percent ?? 0}%`}></i></a
          >
        </article>
      {/each}
    </div>

    <aside class="chronicle-summary">
      <span class="card-label">Day summary</span>
      <strong>{readiness}<small>/100</small></strong>
      <p>Enough movement, enough rest, and one more small story.</p>
      <dl>
        <div>
          <dt>Steps</dt>
          <dd>{today.daily?.steps?.toLocaleString() ?? "—"}</dd>
        </div>
        <div>
          <dt>Exercise</dt>
          <dd>{today.daily?.ring?.exercise_min ?? "—"} min</dd>
        </div>
        <div>
          <dt>Distance</dt>
          <dd>{designDistance((today.daily?.distance_km ?? 0) * 1000)}</dd>
        </div>
      </dl>
      <a href={links.patterns}>See the longer pattern →</a>
    </aside>
  </div>
</section>

<style>
  .chronicle-composition {
    display: grid;
    gap: 2rem;
    padding: clamp(1.25rem, 4vw, 3rem);
    background:
      linear-gradient(
        120deg,
        color-mix(in srgb, var(--accent) 7%, transparent),
        transparent 45%
      ),
      var(--surface);
    color: var(--text);
    font-family: var(--font-serif, Georgia, serif);
  }

  .chronicle-heading,
  .chronicle-layout {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(14rem, 18rem);
    gap: 3rem;
  }

  .chronicle-heading {
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.5rem;
  }

  .eyebrow,
  .card-label {
    margin: 0 0 0.45rem;
    color: var(--accent);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  h2,
  h3,
  p {
    margin: 0;
  }

  h2 {
    max-width: 11ch;
    font-size: clamp(2.4rem, 6vw, 5.8rem);
    font-weight: 400;
    letter-spacing: -0.08em;
    line-height: 0.9;
  }

  .intro,
  .moment-card p,
  .chronicle-summary p {
    color: var(--text-muted);
    line-height: 1.6;
  }

  .intro {
    max-width: 38rem;
    margin-top: 0.8rem;
  }

  .moment-count {
    display: grid;
    justify-items: end;
  }

  .moment-count strong {
    color: var(--accent);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 4rem;
    letter-spacing: -0.12em;
    line-height: 0.8;
  }

  .moment-count span {
    margin-top: 0.5rem;
    color: var(--text-muted);
    font-size: 0.72rem;
  }

  .timeline {
    position: relative;
    display: grid;
    gap: 1.2rem;
  }

  .timeline::before {
    position: absolute;
    top: 0.8rem;
    bottom: 0.8rem;
    left: 5.4rem;
    border-left: 1px dashed var(--border);
    content: "";
  }

  .moment {
    position: relative;
    display: grid;
    grid-template-columns: 4.5rem 1.8rem minmax(0, 1fr);
    gap: 0.7rem;
    align-items: start;
  }

  time {
    padding-top: 0.65rem;
    color: var(--text-muted);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.68rem;
    text-align: right;
  }

  .pin {
    position: relative;
    z-index: 1;
    display: grid;
    place-items: center;
    width: 1.8rem;
    height: 1.8rem;
    border: 1px solid var(--border);
    border-radius: 50%;
    background: var(--surface-1);
    color: var(--accent);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.85rem;
  }

  .movement {
    color: var(--ring-move, var(--accent));
  }

  .library {
    color: var(--accent-2, var(--accent));
  }

  .moment-card {
    display: grid;
    gap: 0.45rem;
    border: 1px solid var(--border);
    border-radius: var(--radius, 0.7rem);
    padding: 1rem;
    background: color-mix(in srgb, var(--surface-1) 96%, var(--accent));
    color: var(--text);
    text-decoration: none;
  }

  .moment-card:hover {
    border-color: var(--accent);
  }

  h3 {
    font-size: 1.25rem;
    font-weight: 400;
  }

  .moment-card strong {
    color: var(--accent);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.72rem;
  }

  .moment-card i {
    display: block;
    height: 0.22rem;
    overflow: hidden;
    background: var(--border);
  }

  .moment-card i::after {
    display: block;
    width: var(--fill);
    height: 100%;
    background: var(--accent);
    content: "";
  }

  .chronicle-summary {
    align-self: start;
    display: grid;
    gap: 0.8rem;
    border-top: 3px solid var(--accent);
    padding-top: 1rem;
  }

  .chronicle-summary > strong {
    color: var(--accent);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 4rem;
    letter-spacing: -0.12em;
    line-height: 0.8;
  }

  .chronicle-summary > strong small {
    color: var(--text-muted);
    font-size: 0.3em;
    letter-spacing: 0;
  }

  .chronicle-summary dl {
    display: grid;
    gap: 0.7rem;
    margin: 1rem 0;
  }

  .chronicle-summary dl div {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-bottom: 1px solid var(--border);
    padding-bottom: 0.45rem;
  }

  dt,
  dd {
    margin: 0;
  }

  dt {
    color: var(--text-muted);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.7rem;
  }

  dd {
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.8rem;
    font-weight: 700;
  }

  a {
    color: var(--accent);
  }

  .chronicle-summary > a {
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.75rem;
    text-decoration: none;
  }

  .empty {
    grid-column: 1 / -1;
    color: var(--text-muted);
  }

  @media (max-width: 700px) {
    .chronicle-heading,
    .chronicle-layout {
      grid-template-columns: 1fr;
      gap: 1.5rem;
    }

    .moment-count {
      justify-items: start;
    }
  }
</style>
