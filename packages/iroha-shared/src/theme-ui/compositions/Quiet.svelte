<script lang="ts">
  import {
    designDateLabel,
    designDuration,
    designPercent,
    type DesignCompositionProps,
  } from "../../design-compositions";

  let { today, readiness, links }: DesignCompositionProps = $props();
</script>

<section class="quiet-composition" aria-labelledby="quiet-title">
  <header class="quiet-header">
    <div>
      <p class="eyebrow">{designDateLabel(today.date)}</p>
      <h2 id="quiet-title">A quieter way<br />to notice.</h2>
    </div>
    <span class="quiet-mark">iroha / 01</span>
  </header>
  <article class="quiet-hero">
    <div>
      <span class="card-label">Your pace today</span><strong>{readiness}</strong
      >
      <p>There is energy here, but nothing needs to be forced.</p>
    </div>
    <div class="quiet-bloom" aria-hidden="true">
      <span></span><span></span><span></span>
    </div>
  </article>
  <div class="quiet-grid">
    <article>
      <span>sleep</span><strong
        >{today.sleep ? designDuration(today.sleep.asleep_s) : "—"}</strong
      >
      <p>
        {today.sleep
          ? `${designPercent(today.sleep.efficiency * 100)} efficiency`
          : "awaiting a record"}
      </p>
    </article>
    <article>
      <span>movement</span><strong
        >{today.daily?.ring?.exercise_min ?? "—"} <small>min</small></strong
      >
      <p>
        {today.daily?.ring?.move_kcal ?? "—"} kcal · {today.daily?.steps?.toLocaleString() ??
          "—"} steps
      </p>
    </article>
    <article>
      <span>body signal</span><strong>{today.daily?.resting_hr ?? "—"}</strong>
      <p>resting heart rate · {today.daily?.hrv_sdnn ?? "—"} ms HRV</p>
    </article>
  </div>
  <footer class="quiet-footer">
    <span>Nothing to optimize right now.</span><a href={links.patterns}
      >See the longer pattern →</a
    >
  </footer>
</section>

<style>
  .quiet-composition {
    display: grid;
    gap: 2.5rem;
    max-width: 68rem;
    margin: 0 auto;
    padding: clamp(2rem, 8vw, 6rem) clamp(1.25rem, 5vw, 4rem);
    background: var(--surface);
    color: var(--text);
    font-family: var(--font-serif, Georgia, serif);
  }

  .quiet-header,
  .quiet-footer {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: end;
  }

  .eyebrow,
  .card-label,
  .quiet-grid span {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }

  h2,
  p {
    margin: 0;
  }

  h2 {
    font-size: clamp(3rem, 8vw, 7rem);
    font-weight: 400;
    letter-spacing: -0.1em;
    line-height: 0.82;
  }

  .quiet-mark,
  .quiet-footer {
    color: var(--text-muted);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.68rem;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }

  .quiet-hero {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: center;
    border-top: 1px solid var(--border);
    border-bottom: 1px solid var(--border);
    padding: 2rem 0;
  }

  .quiet-hero strong {
    display: block;
    color: var(--accent);
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: clamp(5rem, 13vw, 11rem);
    letter-spacing: -0.16em;
    line-height: 0.75;
  }

  .quiet-hero p {
    max-width: 24rem;
    margin-top: 0.8rem;
    color: var(--text-muted);
    font-size: 1.05rem;
    line-height: 1.6;
  }

  .quiet-bloom {
    position: relative;
    width: clamp(7rem, 20vw, 12rem);
    height: clamp(7rem, 20vw, 12rem);
  }

  .quiet-bloom span {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 48%;
    height: 82%;
    border: 1px solid var(--accent);
    border-radius: 60% 40% 60% 40%;
    transform: translate(-50%, -50%) rotate(var(--angle));
    transform-origin: 50% 92%;
    opacity: 0.65;
  }

  .quiet-bloom span:nth-child(1) {
    --angle: -28deg;
  }
  .quiet-bloom span:nth-child(2) {
    --angle: 0deg;
    border-color: var(--accent-2);
  }
  .quiet-bloom span:nth-child(3) {
    --angle: 28deg;
  }

  .quiet-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 1rem;
  }

  .quiet-grid article {
    display: grid;
    gap: 0.35rem;
    border-left: 2px solid var(--accent);
    padding: 0.4rem 0 0.4rem 1rem;
  }

  .quiet-grid span {
    color: var(--text-muted);
  }

  .quiet-grid strong {
    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: clamp(1.5rem, 3vw, 2.8rem);
    letter-spacing: -0.08em;
  }

  .quiet-grid strong small {
    color: var(--text-muted);
    font-size: 0.38em;
    letter-spacing: 0;
  }

  .quiet-grid p {
    color: var(--text-muted);
    font-size: 0.76rem;
    line-height: 1.5;
  }

  .quiet-footer {
    align-items: center;
    border-top: 1px solid var(--border);
    padding-top: 1rem;
  }

  a {
    color: var(--accent);
    text-decoration: none;
  }

  @media (max-width: 640px) {
    .quiet-header,
    .quiet-hero,
    .quiet-footer {
      align-items: flex-start;
      flex-direction: column;
    }

    .quiet-bloom {
      align-self: center;
    }

    .quiet-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
