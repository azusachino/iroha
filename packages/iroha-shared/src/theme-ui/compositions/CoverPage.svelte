<script lang="ts">
  import {
    designDateLabel,
    designDistance,
    designDuration,
    designPercent,
    type DesignCompositionProps,
  } from "../../theme/design-compositions";

  let { today, readiness, links }: DesignCompositionProps = $props();
  const sleep = $derived(today.sleep);
  const ring = $derived(today.daily?.ring);
</script>

<section class="cover-composition" aria-labelledby="cover-title">
  <header class="cover-masthead">
    <div>
      <p class="eyebrow">Iroha / personal field notes</p>
      <h2 id="cover-title">A day<br /><em>in motion.</em></h2>
    </div>
    <div class="cover-index">
      <span>{designDateLabel(today.date)}</span><strong
        >01—{String(today.activities.length + 1).padStart(2, "0")}</strong
      ><span>private edition</span>
    </div>
  </header>

  <div class="cover-grid">
    <article class="cover-art">
      <div class="orbit orbit-one"></div>
      <div class="orbit orbit-two"></div>
      <span class="cover-sun">i</span>
      <span class="cover-art-label">Daily composition / 01</span>
      <strong>{readiness}<small> readiness</small></strong>
      <p>
        {sleep
          ? `${designDuration(sleep.asleep_s)} of rest gave the day a soft start.`
          : "The day is waiting for its first record."} Movement is already in the
        frame; leave some space for whatever comes next.
      </p>
    </article>
    <div class="cover-notes">
      <article class="cover-note">
        <span class="card-label">The body</span><strong
          >{today.daily?.steps?.toLocaleString() ?? "—"}</strong
        >
        <p>
          steps across {designDistance((today.daily?.distance_km ?? 0) * 1000)}
        </p>
      </article>
      <article class="cover-note">
        <span class="card-label">The night</span><strong
          >{sleep ? designDuration(sleep.asleep_s) : "—"}</strong
        >
        <p>
          {sleep
            ? `${designPercent(sleep.efficiency * 100)} efficient`
            : "No sleep record"}
        </p>
      </article>
      <article class="cover-note cover-note-wide">
        <span class="card-label">The next small thing</span><strong
          >{ring?.exercise_min &&
          ring.exercise_goal_min &&
          ring.exercise_min >= ring.exercise_goal_min
            ? "Take the long way home."
            : "Make the first mark."}</strong
        >
        <p>One gentle choice is enough to give the day a shape.</p>
      </article>
    </div>
  </div>

  <footer class="cover-footer">
    <span>{today.date} / personal data cockpit</span><a href={links.patterns}
      >Read the longer pattern →</a
    >
  </footer>
</section>

<style>
  .cover-composition {
    display: grid;
    gap: 2rem;
    padding: clamp(1.25rem, 5vw, 4rem);
    background: var(--surface);
    color: var(--text);
    font-family: var(--font-serif, Georgia, serif);
  }

  .cover-masthead,
  .cover-footer {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.5rem;
  }

  .eyebrow,
  .card-label,
  .cover-art-label {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.13em;
    text-transform: uppercase;
  }

  h2,
  p {
    margin: 0;
  }

  h2 {
    font-size: clamp(3rem, 9vw, 8rem);
    font-weight: 400;
    letter-spacing: -0.1em;
    line-height: 0.78;
  }

  h2 em {
    color: var(--accent);
    font-weight: 400;
  }

  .cover-index {
    display: grid;
    justify-items: end;
    gap: 0.25rem;
    color: var(--text-muted);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.68rem;
    text-align: right;
    text-transform: uppercase;
  }

  .cover-index strong {
    color: var(--text);
    font-family: var(--font-serif, Georgia, serif);
    font-size: 2rem;
    letter-spacing: -0.08em;
  }

  .cover-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.25fr) minmax(15rem, 0.75fr);
    gap: 1rem;
  }

  .cover-art,
  .cover-note {
    position: relative;
    overflow: hidden;
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface-1) 92%, var(--accent));
  }

  .cover-art {
    display: grid;
    align-content: end;
    min-height: 28rem;
    gap: 0.65rem;
    padding: clamp(1.25rem, 4vw, 2.5rem);
  }

  .orbit {
    position: absolute;
    border: 1px solid color-mix(in srgb, var(--accent) 52%, transparent);
    border-radius: 50%;
    transform: rotate(-25deg);
  }

  .orbit-one {
    top: 10%;
    right: -10%;
    width: 28rem;
    height: 15rem;
  }

  .orbit-two {
    top: 21%;
    right: 0;
    width: 20rem;
    height: 11rem;
    border-color: color-mix(in srgb, var(--accent-2) 50%, transparent);
  }

  .cover-sun {
    position: absolute;
    top: 24%;
    right: 30%;
    display: grid;
    place-items: center;
    width: 3.5rem;
    height: 3.5rem;
    border-radius: 50%;
    background: var(--accent);
    color: var(--bg);
    font-size: 1.6rem;
  }

  .cover-art > *:not(.orbit):not(.cover-sun) {
    position: relative;
    z-index: 1;
  }

  .cover-art > strong {
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: clamp(4rem, 10vw, 8rem);
    letter-spacing: -0.13em;
    line-height: 0.8;
  }

  .cover-art > strong small {
    color: var(--text-muted);
    font-size: 0.22em;
    letter-spacing: 0;
  }

  .cover-art > p,
  .cover-note p {
    color: var(--text-muted);
    line-height: 1.55;
  }

  .cover-art > p {
    max-width: 30rem;
  }

  .cover-notes {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
  }

  .cover-note {
    display: grid;
    align-content: start;
    gap: 0.45rem;
    min-height: 11rem;
    padding: 1rem;
  }

  .cover-note-wide {
    grid-column: 1 / -1;
    min-height: 0;
  }

  .cover-note strong {
    font-size: clamp(1.4rem, 3vw, 2.4rem);
    font-weight: 400;
    letter-spacing: -0.07em;
  }

  .cover-footer {
    align-items: center;
    border-top: 1px solid var(--border);
    border-bottom: 0;
    padding-top: 0.8rem;
    padding-bottom: 0;
    color: var(--text-muted);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.7rem;
    text-transform: uppercase;
  }

  a {
    color: var(--accent);
    text-decoration: none;
  }

  @media (max-width: 768px) {
    .cover-masthead,
    .cover-footer {
      align-items: flex-start;
      flex-direction: column;
    }

    .cover-index {
      justify-items: start;
      text-align: left;
    }

    .cover-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
