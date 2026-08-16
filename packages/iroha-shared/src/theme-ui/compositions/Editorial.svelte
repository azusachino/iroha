<script lang="ts">
  import {
    designActivitySummary,
    designDateLabel,
    designDistance,
    designDuration,
    designPercent,
    designSportLabel,
    type DesignCompositionProps,
  } from "../../theme/design-compositions";
  import DesignRingGauge from "./DesignRingGauge.svelte";

  let { today, readiness, links }: DesignCompositionProps = $props();
  const sleep = $derived(today.sleep);
  const ring = $derived(today.daily?.ring);
  const media = $derived(today.media[0]);
</script>

<section class="editorial-composition" aria-labelledby="editorial-title">
  <header class="editorial-heading">
    <div>
      <p class="eyebrow">{designDateLabel(today.date)}</p>
      <h2 id="editorial-title">A good day has a shape.</h2>
      <p class="intro">
        A composed read of movement, recovery, and the things you touched.
      </p>
    </div>
    <a class="date-link" href={links.patterns}>Open the longer pattern →</a>
  </header>

  <article class="editorial-hero">
    <div class="hero-copy">
      <span class="eyebrow">Daily signal · {today.date}</span>
      <strong class="readiness">{readiness}<small>/100</small></strong>
      <h3>Steady, with room to move.</h3>
      <p>
        {sleep
          ? `${designDuration(sleep.asleep_s)} asleep and ${designPercent(sleep.efficiency * 100)} efficient.`
          : "No recovery record has arrived for this day."}
        {ring?.exercise_min &&
        ring.exercise_goal_min &&
        ring.exercise_min >= ring.exercise_goal_min
          ? " Exercise is already covered."
          : " A small block of movement would change the shape gently."}
      </p>
    </div>
    <div class="hero-gauge"><DesignRingGauge {today} size={184} /></div>
  </article>

  <div class="editorial-grid">
    <a class="editorial-card" href={links.night}>
      <span class="card-label">Recovery</span>
      <strong>{sleep ? designDuration(sleep.asleep_s) : "—"}</strong>
      <p>
        {sleep
          ? `${designPercent(sleep.efficiency * 100)} sleep efficiency`
          : "No sleep recorded"}
      </p>
    </a>
    <a class="editorial-card" href={links.motion}>
      <span class="card-label">Training</span>
      <strong>{today.activities.length} <small>sessions</small></strong>
      <p>
        {today.activities[0]
          ? designActivitySummary(today.activities[0])
          : "No activity recorded"}
      </p>
    </a>
    <a class="editorial-card" href={links.library}>
      <span class="card-label">In progress</span>
      <strong>{designPercent(media?.progress_percent)}</strong>
      <p>{media?.title ?? "Nothing logged today"}</p>
    </a>
  </div>

  <footer class="editorial-footnote">
    <span>{today.daily?.steps?.toLocaleString() ?? "—"} steps</span>
    <span>{designDistance((today.daily?.distance_km ?? 0) * 1000)} covered</span
    >
    <span
      >{today.activities.length
        ? designSportLabel(today.activities[0].sport_type)
        : "quiet"} first</span
    >
  </footer>
</section>

<style>
  .editorial-composition {
    --paper: color-mix(in srgb, var(--surface) 94%, var(--accent) 6%);
    display: grid;
    gap: 1.4rem;
    padding: clamp(1.25rem, 4vw, 3rem);
    background:
      radial-gradient(
        circle at 82% 10%,
        color-mix(in srgb, var(--accent) 18%, transparent),
        transparent 26rem
      ),
      var(--paper);
    color: var(--text);
    font-family: var(--font-serif, Georgia, serif);
  }

  .editorial-heading,
  .editorial-hero,
  .editorial-footnote {
    display: flex;
    justify-content: space-between;
    gap: 1.5rem;
    align-items: end;
  }

  .editorial-heading {
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.5rem;
  }

  .eyebrow,
  .card-label {
    margin: 0 0 0.45rem;
    color: var(--accent);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.68rem;
    font-weight: 700;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  h2,
  h3,
  p {
    margin: 0;
  }

  h2 {
    max-width: 12ch;
    font-size: clamp(2.3rem, 6vw, 5.6rem);
    font-weight: 400;
    letter-spacing: -0.07em;
    line-height: 0.9;
  }

  .intro,
  .hero-copy p,
  .editorial-card p {
    color: var(--text-muted);
    line-height: 1.6;
  }

  .intro {
    max-width: 35rem;
    margin-top: 0.8rem;
  }

  .date-link,
  .editorial-card {
    color: var(--text);
    text-decoration: none;
  }

  .date-link {
    flex: 0 0 auto;
    border-bottom: 1px solid var(--accent);
    padding-bottom: 0.25rem;
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.78rem;
  }

  .editorial-hero {
    min-height: 18rem;
    border-radius: var(--radius, 0.9rem);
    padding: clamp(1.2rem, 4vw, 2.4rem);
    background:
      linear-gradient(
        120deg,
        color-mix(in srgb, var(--accent) 15%, transparent),
        transparent 48%
      ),
      color-mix(in srgb, var(--surface-1) 94%, var(--accent));
    box-shadow: var(--tile-shadow, 0 0.8rem 2rem rgb(0 0 0 / 0.1));
  }

  .hero-copy {
    display: grid;
    align-content: center;
    gap: 0.55rem;
    max-width: 35rem;
  }

  .readiness {
    color: var(--accent);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: clamp(4rem, 10vw, 8rem);
    letter-spacing: -0.12em;
    line-height: 0.8;
  }

  .readiness small,
  .editorial-card strong small {
    color: var(--text-muted);
    font-size: 0.3em;
    font-weight: 500;
    letter-spacing: 0;
  }

  h3 {
    font-size: clamp(1.4rem, 3vw, 2.2rem);
    font-weight: 400;
  }

  .hero-gauge {
    flex: 0 0 auto;
  }

  .editorial-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 0.75rem;
  }

  .editorial-card {
    display: grid;
    gap: 0.4rem;
    border-top: 2px solid var(--accent);
    padding: 1rem 0.2rem 0.8rem;
  }

  .editorial-card strong {
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: clamp(1.4rem, 3vw, 2.4rem);
    letter-spacing: -0.07em;
  }

  .editorial-footnote {
    justify-content: flex-start;
    flex-wrap: wrap;
    border-top: 1px solid var(--border);
    padding-top: 0.75rem;
    color: var(--text-muted);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.7rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }

  @media (max-width: 768px) {
    .editorial-heading,
    .editorial-hero {
      align-items: flex-start;
      flex-direction: column;
    }

    .editorial-grid {
      grid-template-columns: 1fr;
    }

    .hero-gauge {
      align-self: center;
    }
  }
</style>
