<script lang="ts">
  import {
    designActivitySummary,
    designDateLabel,
    designDistance,
    designDuration,
    designSportLabel,
    designTimeLabel,
    type DesignCompositionProps,
  } from "../../design-compositions";

  let { today, readiness, links }: DesignCompositionProps = $props();
</script>

<section class="journal-composition" aria-labelledby="journal-title">
  <header class="journal-header">
    <div class="journal-number">
      No. {String(today.activities.length + 18).padStart(3, "0")}
    </div>
    <div>
      <p class="eyebrow">Field journal / {designDateLabel(today.date)}</p>
      <h2 id="journal-title">Notes from a body in motion.</h2>
    </div>
    <span class="journal-traces"
      >private · {today.activities.length} traces</span
    >
  </header>

  <div class="journal-layout">
    <div class="journal-paper">
      <div class="journal-rule"></div>
      {#if today.sleep}
        <article class="journal-entry">
          <time>07:04</time>
          <div>
            <span class="journal-tag">Recovery</span>
            <h3>The night held.</h3>
            <p>
              {designDuration(today.sleep.asleep_s)} asleep, {Math.round(
                today.sleep.efficiency * 100,
              )}% efficient. The day began with enough reserve.
            </p>
          </div>
        </article>
      {/if}
      {#each today.activities as activity (activity.id)}
        <article class="journal-entry">
          <time>{designTimeLabel(activity.started_at)}</time>
          <div>
            <span class="journal-tag"
              >{designSportLabel(activity.sport_type)}</span
            >
            <h3>{activity.title}.</h3>
            <p>{designActivitySummary(activity)}. Recorded activity.</p>
            <a href={links.activity(activity.id)}>Open the trace →</a>
          </div>
        </article>
      {:else}
        <article class="journal-entry">
          <time>now</time>
          <div>
            <span class="journal-tag">Observation</span>
            <h3>Stillness is data.</h3>
            <p>No movement was recorded for this entry.</p>
          </div>
        </article>
      {/each}
      <article class="journal-entry">
        <time>now</time>
        <div>
          <span class="journal-tag">Observation</span>
          <h3>Leave one margin.</h3>
          <p>
            The useful shape of a day is not a perfect score. It is the room
            left for the unexpected.
          </p>
        </div>
      </article>
    </div>

    <aside class="journal-index">
      <span class="card-label">Today's index</span>
      <strong>{readiness}<small>/100</small></strong>
      <dl>
        <div>
          <dt>movement</dt>
          <dd>{today.daily?.ring?.exercise_min ?? "—"} min</dd>
        </div>
        <div>
          <dt>distance</dt>
          <dd>{designDistance((today.daily?.distance_km ?? 0) * 1000)}</dd>
        </div>
        <div>
          <dt>heartbeat</dt>
          <dd>{today.daily?.resting_hr ?? "—"} bpm</dd>
        </div>
      </dl>
      <p>Collected from your canonical data sources.</p>
      <a href={links.patterns}>Continue through the journal →</a>
    </aside>
  </div>
</section>

<style>
  .journal-composition {
    display: grid;
    gap: 1.5rem;
    padding: clamp(1.25rem, 4vw, 3rem);
    background: color-mix(in srgb, var(--surface) 97%, var(--accent));
    color: var(--text);
    font-family: var(--font-serif, Georgia, serif);
  }

  .journal-header {
    display: grid;
    grid-template-columns: 5rem minmax(0, 1fr) auto;
    gap: 1.2rem;
    align-items: start;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1rem;
  }

  .journal-number,
  .journal-traces,
  .eyebrow,
  .journal-tag,
  .card-label {
    color: var(--accent);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.66rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .journal-number {
    padding-top: 0.3rem;
  }

  .journal-traces {
    color: var(--text-muted);
  }

  .eyebrow,
  .card-label {
    margin: 0 0 0.4rem;
    font-weight: 750;
  }

  h2,
  h3,
  p {
    margin: 0;
  }

  h2 {
    max-width: 15ch;
    font-size: clamp(2.1rem, 5vw, 4.8rem);
    font-weight: 400;
    letter-spacing: -0.08em;
    line-height: 0.9;
  }

  .journal-layout {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(14rem, 18rem);
    gap: 2rem;
  }

  .journal-paper {
    position: relative;
    display: grid;
    gap: 1.5rem;
    padding: 0.8rem 1rem 1rem 0;
    background: repeating-linear-gradient(
      0deg,
      transparent 0 2.35rem,
      color-mix(in srgb, var(--accent) 10%, transparent) 2.35rem 2.4rem
    );
  }

  .journal-rule {
    position: absolute;
    top: 0;
    bottom: 0;
    left: 4.7rem;
    border-left: 1px solid color-mix(in srgb, var(--accent) 28%, var(--border));
  }

  .journal-entry {
    position: relative;
    display: grid;
    grid-template-columns: 3.9rem minmax(0, 1fr);
    gap: 1.4rem;
  }

  .journal-entry time {
    padding-top: 0.25rem;
    color: var(--text-muted);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.68rem;
    text-align: right;
  }

  .journal-entry > div {
    display: grid;
    gap: 0.35rem;
    padding-left: 0.5rem;
  }

  .journal-tag {
    width: fit-content;
    color: var(--accent-2, var(--accent));
  }

  h3 {
    font-size: 1.35rem;
    font-weight: 400;
  }

  .journal-entry p,
  .journal-index p {
    max-width: 42rem;
    color: var(--text-muted);
    line-height: 1.65;
  }

  .journal-entry a,
  .journal-index a {
    color: var(--accent);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.72rem;
    text-decoration: none;
  }

  .journal-index {
    align-self: start;
    display: grid;
    gap: 0.8rem;
    border: 1px solid var(--border);
    border-radius: 0.2rem;
    padding: 1rem;
    background: color-mix(in srgb, var(--surface-1) 92%, var(--accent));
    box-shadow: 0.45rem 0.45rem 0
      color-mix(in srgb, var(--accent) 12%, transparent);
  }

  .journal-index > strong {
    color: var(--accent);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 4rem;
    letter-spacing: -0.12em;
    line-height: 0.8;
  }

  .journal-index > strong small {
    color: var(--text-muted);
    font-size: 0.3em;
    letter-spacing: 0;
  }

  .journal-index dl {
    display: grid;
    gap: 0.6rem;
    margin: 1rem 0 0;
  }

  .journal-index dl div {
    display: flex;
    justify-content: space-between;
    gap: 0.5rem;
    border-bottom: 1px solid var(--border);
    padding-bottom: 0.4rem;
  }

  dt,
  dd {
    margin: 0;
  }

  dt {
    color: var(--text-muted);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.68rem;
  }

  dd {
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.74rem;
    font-weight: 700;
  }

  @media (max-width: 768px) {
    .journal-header,
    .journal-layout {
      grid-template-columns: 1fr;
    }

    .journal-number,
    .journal-traces {
      padding: 0;
    }
  }
</style>
