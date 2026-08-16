<script lang="ts">
  import {
    designActivitySummary,
    designDateLabel,
    designDistance,
    designDuration,
    designPercent,
    designTimeLabel,
    type DesignCompositionProps,
  } from "../../theme/design-compositions";

  let { today, readiness, links }: DesignCompositionProps = $props();
</script>

<section class="os-composition" aria-labelledby="os-title">
  <aside class="os-sidebar">
    <span class="os-logo">i</span>
    <nav aria-label="Workspace sections">
      <a class="selected" href={links.patterns}>◈ <span>Today</span></a>
      <a href={links.patterns}>◌ <span>Patterns</span></a>
      <a href={links.motion}>↗ <span>Motion</span></a>
      <a href={links.library}>▧ <span>Library</span></a>
    </nav>
    <div class="os-sync">
      <span>Workspace health</span><strong><i></i> Synced just now</strong>
    </div>
  </aside>

  <main class="os-main">
    <header class="os-heading">
      <div>
        <p class="eyebrow">Personal OS / daily workspace</p>
        <h2 id="os-title">Good morning, Haru.</h2>
        <p class="muted">
          Everything for today, arranged as a calm working surface.
        </p>
      </div>
      <span class="os-date">{designDateLabel(today.date)}</span>
    </header>
    <div class="os-toolbar">
      <span>Today / {today.date}</span>
      <div>
        <button class="active" type="button">Board</button><button type="button"
          >List</button
        ><button type="button">Timeline</button>
      </div>
    </div>
    <div class="os-columns">
      <section class="os-column">
        <header>
          <span class="column-dot teal"></span><strong>Now</strong><small
            >2 blocks</small
          >
        </header>
        <article class="os-card os-hero">
          <span class="card-label">Readiness</span><strong
            >{readiness}<small>/100</small></strong
          >
          <p>Steady, with room to move.</p>
          <i style={`--fill:${readiness}%`}></i>
        </article>
        <article class="os-card">
          <span class="card-label">Focus note</span>
          <p>Keep the next action small enough to begin.</p>
          <span class="os-meta">Added today · personal</span>
        </article>
      </section>
      <section class="os-column">
        <header>
          <span class="column-dot pink"></span><strong>Collected</strong><small
            >{today.activities.length + 1} items</small
          >
        </header>
        {#each today.activities as activity (activity.id)}
          <a class="os-card" href={links.activity(activity.id)}
            ><span class="os-card-top"
              ><b>{activity.sport_type.slice(0, 2).toUpperCase()}</b><small
                >{designTimeLabel(activity.started_at)}</small
              ></span
            ><strong>{activity.title}</strong>
            <p>{designActivitySummary(activity)}</p></a
          >
        {:else}
          <article class="os-card">
            <span class="card-label">No movement yet</span>
            <p>Collected activity will appear here.</p>
          </article>
        {/each}
        <article class="os-card">
          <span class="card-label">Sleep</span><strong
            >{today.sleep ? designDuration(today.sleep.asleep_s) : "—"}</strong
          >
          <p>Recovery block</p>
        </article>
      </section>
      <section class="os-column">
        <header>
          <span class="column-dot amber"></span><strong>Later</strong><small
            >one idea</small
          >
        </header>
        <article class="os-card os-idea">
          <span class="card-label">Open loop</span><strong
            >Make space for a little wonder.</strong
          >
          <p>
            Media progress: {designPercent(today.media[0]?.progress_percent)}
          </p>
          <a href={links.library}>Open library →</a>
        </article>
        <article class="os-card os-facts">
          <span class="card-label">Body signal</span><strong
            >{today.daily?.steps?.toLocaleString() ?? "—"}</strong
          >
          <p>
            steps · {designDistance((today.daily?.distance_km ?? 0) * 1000)} · {today
              .daily?.resting_hr ?? "—"} bpm
          </p>
        </article>
      </section>
    </div>
  </main>
</section>

<style>
  .os-composition {
    display: grid;
    grid-template-columns: 11rem minmax(0, 1fr);
    min-height: 35rem;
    background: color-mix(in srgb, var(--surface) 95%, var(--accent));
    color: var(--text);
    font-family: var(--font-sans, system-ui, sans-serif);
  }

  .os-sidebar {
    display: grid;
    align-content: start;
    gap: 2rem;
    border-right: 1px solid var(--border);
    padding: 1rem;
    background: var(--surface-strong, var(--surface));
  }

  .os-logo {
    display: grid;
    place-items: center;
    width: 2rem;
    height: 2rem;
    border-radius: 0.6rem;
    background: var(--accent);
    color: var(--bg);
    font-family: var(--font-serif, Georgia, serif);
    font-size: 1.3rem;
    font-weight: 700;
  }

  nav {
    display: grid;
    gap: 0.3rem;
  }

  nav a {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    border-radius: 0.45rem;
    padding: 0.55rem;
    color: var(--text-muted);
    font-size: 0.78rem;
    text-decoration: none;
  }

  nav a.selected,
  nav a:hover {
    background: color-mix(in srgb, var(--accent) 12%, transparent);
    color: var(--text);
  }

  nav a.selected {
    color: var(--accent);
    font-weight: 700;
  }

  .os-sync {
    display: grid;
    gap: 0.4rem;
    margin-top: auto;
    border-top: 1px solid var(--border);
    padding-top: 0.8rem;
    color: var(--text-muted);
    font-size: 0.63rem;
  }

  .os-sync strong {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    color: var(--text);
    font-size: 0.68rem;
  }

  .os-sync i {
    width: 0.4rem;
    height: 0.4rem;
    border-radius: 50%;
    background: var(--accent);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 16%, transparent);
  }

  .os-main {
    display: grid;
    align-content: start;
    gap: 1rem;
    min-width: 0;
    padding: clamp(1rem, 3vw, 2rem);
  }

  .os-heading,
  .os-toolbar {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
  }

  .eyebrow,
  .card-label {
    margin: 0 0 0.4rem;
    color: var(--accent);
    font-size: 0.66rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  h2,
  p {
    margin: 0;
  }

  h2 {
    font-size: clamp(1.8rem, 4vw, 3.4rem);
    letter-spacing: -0.08em;
  }

  .muted,
  .os-card p {
    color: var(--text-muted);
    line-height: 1.5;
  }

  .os-date {
    color: var(--text-muted);
    font-size: 0.74rem;
  }

  .os-toolbar {
    border: 1px solid var(--border);
    border-radius: 0.55rem;
    padding: 0.35rem 0.5rem 0.35rem 0.75rem;
    color: var(--text-muted);
    font-size: 0.7rem;
  }

  .os-toolbar div {
    display: flex;
    gap: 0.2rem;
  }

  button {
    border: 0;
    border-radius: 0.35rem;
    padding: 0.35rem 0.55rem;
    background: transparent;
    color: var(--text-muted);
    font: inherit;
    font-size: 0.7rem;
  }

  button.active {
    background: var(--surface-2, var(--border));
    color: var(--text);
  }

  .os-columns {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 0.75rem;
  }

  .os-column {
    display: grid;
    align-content: start;
    gap: 0.65rem;
  }

  .os-column > header {
    display: flex;
    align-items: center;
    gap: 0.45rem;
    padding: 0.4rem 0.2rem;
    border-bottom: 1px solid var(--border);
  }

  .os-column > header small {
    margin-left: auto;
    color: var(--text-muted);
    font-size: 0.64rem;
  }

  .column-dot {
    width: 0.5rem;
    height: 0.5rem;
    border-radius: 50%;
  }

  .teal {
    background: var(--mark-teal, var(--accent));
  }
  .pink {
    background: var(--mark-magenta, var(--accent-2));
  }
  .amber {
    background: var(--mark-amber, var(--accent));
  }

  .os-card {
    display: grid;
    gap: 0.45rem;
    border: 1px solid var(--border);
    border-radius: 0.65rem;
    padding: 0.9rem;
    background: var(--surface-1);
    color: var(--text);
    text-decoration: none;
  }

  a.os-card:hover {
    border-color: var(--accent);
    transform: translateY(-1px);
  }

  .os-card strong {
    font-size: 1rem;
    letter-spacing: -0.04em;
  }

  .os-hero {
    background: color-mix(in srgb, var(--accent) 10%, var(--surface-1));
  }

  .os-hero > strong {
    color: var(--accent);
    font-size: 3.5rem;
    letter-spacing: -0.12em;
  }

  .os-hero > strong small {
    color: var(--text-muted);
    font-size: 0.3em;
    letter-spacing: 0;
  }

  .os-hero > i {
    display: block;
    height: 0.25rem;
    background: linear-gradient(
      90deg,
      var(--accent) var(--fill),
      var(--border) var(--fill)
    );
  }

  .os-meta {
    color: var(--text-muted);
    font-size: 0.65rem;
  }

  .os-card-top {
    display: flex;
    justify-content: space-between;
    gap: 0.5rem;
    color: var(--text-muted);
    font-size: 0.65rem;
  }

  .os-card-top b {
    color: var(--accent);
    font-size: 0.62rem;
  }

  .os-idea {
    background: color-mix(in srgb, var(--accent-2) 11%, var(--surface-1));
  }

  .os-idea > strong {
    font-size: 1.4rem;
    line-height: 1.1;
  }

  .os-card a {
    color: var(--accent);
    font-size: 0.72rem;
  }

  @media (max-width: 1024px) {
    .os-composition {
      grid-template-columns: 1fr;
    }

    .os-sidebar {
      grid-template-columns: auto 1fr;
      align-items: center;
      border-right: 0;
      border-bottom: 1px solid var(--border);
    }

    .os-sidebar nav {
      display: flex;
      flex-wrap: wrap;
      gap: 0.1rem;
    }

    .os-sync {
      display: none;
    }
  }

  @media (max-width: 640px) {
    .os-heading,
    .os-toolbar {
      align-items: flex-start;
      flex-direction: column;
    }

    .os-columns {
      grid-template-columns: 1fr;
    }
  }
</style>
