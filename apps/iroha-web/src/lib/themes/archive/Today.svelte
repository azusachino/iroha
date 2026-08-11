<script lang="ts">
  import type {
    Activity,
    DailyRow,
    MediaHomeEvent,
    SleepSession,
  } from "$lib/api";
  import {
    formatDistance,
    formatDuration,
    formatPace,
    mediaEventVerb,
  } from "$lib/format";

  let {
    dayLabel,
    day,
    dRow,
    mainNight,
    acts,
    mediaEvents,
  }: {
    dayLabel: string;
    day: string;
    dRow: DailyRow | undefined;
    mainNight: SleepSession | undefined;
    acts: Activity[];
    mediaEvents: MediaHomeEvent[];
  } = $props();

  function number(value: number | null | undefined, digits = 0): string {
    if (typeof value !== "number" || !Number.isFinite(value)) return "—";
    return value.toLocaleString(undefined, {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
    });
  }

  // The catalog number is derived from the date itself -- an honest label,
  // not a random identifier -- following the accession-number convention a
  // collections registrar assigns an object entering the record.
  const accession = $derived(`ARC · ${day.replaceAll("-", ".")}`);

  const moveProgress = $derived(
    dRow && dRow.move_goal_kcal > 0
      ? Math.min(100, (dRow.move_kcal / dRow.move_goal_kcal) * 100)
      : 0,
  );
  const sleepProgress = $derived(
    mainNight ? Math.round(mainNight.efficiency * 100) : null,
  );

  // Duotone reading between the two catalog inks: cool violet at the low
  // end, warm ochre at the high end. Every proportion in this language
  // reads through the same two-color scale.
  function tone(pct: number | null | undefined): string {
    if (pct == null || !Number.isFinite(pct))
      return "color-mix(in srgb, var(--border) 55%, var(--surface))";
    const clamped = Math.max(0, Math.min(100, pct));
    return `color-mix(in srgb, var(--accent-2) ${clamped}%, var(--accent) ${100 - clamped}%)`;
  }

  // The day's sessions become a single core sample: one stratum per
  // recorded activity, thickness set by real distance (or duration when no
  // distance was logged), tone set by relative pace across today's
  // sessions -- an honest transform of the day's own evidence, not a
  // decorative gauge.
  const paceRange = $derived.by(() => {
    const paces = acts
      .map((a) => a.avg_pace_s_per_km)
      .filter((p): p is number => typeof p === "number" && Number.isFinite(p));
    if (paces.length === 0) return null;
    const min = Math.min(...paces);
    const max = Math.max(...paces);
    return { min, max: min === max ? min + 1 : max };
  });

  const sessionCore = $derived(
    acts.map((activity, index) => {
      const magnitude =
        activity.distance_m && activity.distance_m > 0
          ? activity.distance_m
          : (activity.duration_s ?? activity.moving_time_s ?? 60);
      const pace = activity.avg_pace_s_per_km;
      const pct =
        pace != null && paceRange
          ? ((pace - paceRange.min) / (paceRange.max - paceRange.min)) * 100
          : null;
      return {
        key: activity.id,
        index,
        magnitude,
        pct,
        label: activity.title || activity.sport_type,
        value: `${formatDistance(activity.distance_m)} · ${formatDuration(activity.duration_s ?? activity.moving_time_s)}`,
      };
    }),
  );
  const maxSessionMagnitude = $derived(
    Math.max(1, ...sessionCore.map((row) => row.magnitude)),
  );
</script>

<section class="folio-today" aria-labelledby="folio-today-title">
  <header class="folio-head">
    <div>
      <p class="folio-kicker">Daily record / accession</p>
      <h1 id="folio-today-title">Filed under {day}.</h1>
      <p>
        An indexed record of the body, its routes, and the hours between --
        catalogued, not scored.
      </p>
      <small>{dayLabel} · imported snapshot</small>
    </div>
    <div class="accession-tag" aria-label={`Accession number ${accession}`}>
      {accession}
    </div>
  </header>

  <section class="folio-focus catalog-card">
    <div>
      <p class="folio-kicker">Primary observation</p>
      <strong>{number(dRow?.steps)} <small>steps</small></strong>
      <p>
        {number(dRow?.distance_km, 1)} km traveled · {number(
          dRow?.exercise_min,
        )} min exercise
      </p>
    </div>
    <div
      class="core-gauge"
      aria-label={`${Math.round(moveProgress)} percent move goal`}
    >
      <div class="core-gauge-well">
        <i
          style={`height: ${moveProgress}%; background: ${tone(moveProgress)};`}
        ></i>
      </div>
      <span>{Math.round(moveProgress)}%<br /><small>move goal</small></span>
    </div>
  </section>

  <div class="folio-grid">
    <section class="catalog-card">
      <p class="folio-kicker">Body signals</p>
      <h2>{number(dRow?.resting_hr)} <small>bpm resting</small></h2>
      <dl>
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

    <section class="catalog-card">
      <p class="folio-kicker">Recovery signal</p>
      {#if mainNight}
        <div class="recovery-row">
          <div class="core-gauge-well small">
            <i
              style={`height: ${sleepProgress}%; background: ${tone(sleepProgress)};`}
            ></i>
          </div>
          <div>
            <h2>{sleepProgress}% <small>efficiency</small></h2>
            <p>
              {formatDuration(mainNight.asleep_s)} asleep · {formatDuration(
                mainNight.time_in_bed_s,
              )} in bed
            </p>
          </div>
        </div>
      {:else}
        <h2>— <small>no session</small></h2>
        <p>No sleep session was recorded for this date.</p>
      {/if}
    </section>
  </div>

  <section class="folio-sessions">
    <header>
      <div>
        <p class="folio-kicker">Recorded movement</p>
        <h2>Today's core sample</h2>
      </div>
      <span>{acts.length} entries</span>
    </header>
    {#if sessionCore.length}
      <div
        class="core-log"
        role="img"
        aria-label="Today's sessions by distance"
      >
        <div class="core-strip">
          {#each sessionCore as row (row.key)}
            <div
              class="core-band"
              style={`flex-grow: ${Math.max(row.magnitude / maxSessionMagnitude, 0.08)}; background: ${tone(row.pct)};`}
            ></div>
          {/each}
        </div>
        <div class="core-legend">
          {#each sessionCore as row (row.key)}
            <div
              class="core-row"
              style={`flex-grow: ${Math.max(row.magnitude / maxSessionMagnitude, 0.08)};`}
            >
              <b>{String(row.index + 1).padStart(2, "0")}</b>
              <strong>{row.label}</strong>
              <span>{row.value}</span>
            </div>
          {/each}
        </div>
      </div>
    {:else}
      <p class="folio-empty">
        No movement sessions were recorded for this date.
      </p>
    {/if}
  </section>

  <section class="folio-sessions">
    <header>
      <div>
        <p class="folio-kicker">Media touched today</p>
        <h2>Today's accessions</h2>
      </div>
      <span>{mediaEvents.length} entries</span>
    </header>
    {#if mediaEvents.length}
      <ul class="folio-media-list">
        {#each mediaEvents as event (event.id)}
          <li>
            <a class="folio-media-row" href={`/library/${event.media_id}`}>
              {#if event.cover_image_url}
                <img src={event.cover_image_url} alt="" loading="lazy" />
              {:else}
                <span class="folio-media-thumb" aria-hidden="true"
                  >{(event.native_title || event.title).slice(0, 1)}</span
                >
              {/if}
              <span class="folio-media-copy">
                <strong>{event.native_title || event.title}</strong>
                <span>{mediaEventVerb(event)}</span>
              </span>
              {#if event.rating != null}<span class="folio-media-score"
                  >{event.rating.toFixed(1)}</span
                >{/if}
            </a>
          </li>
        {/each}
      </ul>
    {:else}
      <p class="folio-empty">No media events were recorded for this date.</p>
    {/if}
  </section>

  <footer class="folio-source">
    Source: imported raw evidence · no derived readiness score
  </footer>
</section>

<style>
  .folio-today {
    display: grid;
    gap: 1.3rem;
  }
  .folio-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.64rem;
    letter-spacing: 0.15em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    font-family: var(--font-serif);
    font-weight: 700;
    letter-spacing: -0.01em;
  }
  h1 {
    max-width: 13ch;
    font-size: clamp(2.5rem, 6.5vw, 5.6rem);
    line-height: 0.95;
  }
  h2 {
    font-size: 1.55rem;
  }
  .folio-head {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: start;
    padding-bottom: 1.75rem;
    border-bottom: 1px solid var(--border);
  }
  .folio-head p:not(.folio-kicker) {
    max-width: 40rem;
    margin: 1rem 0 0;
    color: var(--text-muted);
    line-height: 1.6;
  }
  .folio-head small {
    display: block;
    margin-top: 1rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.7rem;
  }
  .accession-tag {
    display: inline-flex;
    align-items: center;
    flex-shrink: 0;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    padding: 0.45rem 0.75rem;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.78rem;
    letter-spacing: 0.04em;
    white-space: nowrap;
  }
  .catalog-card {
    position: relative;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
    padding: 1.5rem 1.5rem 1.5rem 1.7rem;
  }
  .catalog-card::before {
    content: "";
    position: absolute;
    left: -1px;
    top: 1.15rem;
    width: 4px;
    height: 2.3rem;
    background: var(--accent-2);
    border-radius: 0 2px 2px 0;
  }
  .folio-focus {
    display: grid;
    grid-template-columns: 1fr minmax(10rem, 0.7fr);
    gap: 2rem;
    align-items: center;
  }
  .folio-focus strong {
    display: block;
    font-family: var(--font-serif);
    font-size: clamp(2.4rem, 6.5vw, 4.6rem);
    font-weight: 700;
    letter-spacing: -0.02em;
  }
  .folio-focus strong small,
  .catalog-card h2 small {
    color: var(--text-muted);
    font-family: inherit;
    font-size: 0.78rem;
    letter-spacing: 0;
    text-transform: none;
    font-weight: 400;
  }
  .folio-focus div > p:last-child,
  .catalog-card > p:last-child,
  .folio-sessions > p {
    color: var(--text-muted);
  }
  .core-gauge {
    display: flex;
    align-items: end;
    justify-self: end;
    gap: 0.7rem;
    color: var(--text-muted);
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  .core-gauge span small {
    font-size: 0.62rem;
  }
  .core-gauge-well {
    position: relative;
    width: 1.6rem;
    height: 4.4rem;
    overflow: hidden;
    border: 1px solid var(--border);
    border-radius: 2px;
    background: color-mix(in srgb, var(--surface) 94%, transparent);
  }
  .core-gauge-well.small {
    width: 1.15rem;
    height: 3.2rem;
  }
  .core-gauge-well i {
    position: absolute;
    right: 0;
    bottom: 0;
    left: 0;
    display: block;
  }
  .recovery-row {
    display: flex;
    align-items: center;
    gap: 1.1rem;
    margin-top: 0.5rem;
  }
  .recovery-row p {
    margin: 0.5rem 0 0;
    color: var(--text-muted);
  }
  .folio-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 1.25rem;
  }
  .catalog-card h2 {
    margin-top: 0.9rem;
  }
  .catalog-card > p {
    line-height: 1.6;
  }
  dl {
    display: grid;
    gap: 0.7rem;
    margin: 1.4rem 0 0;
  }
  dl div {
    display: flex;
    justify-content: space-between;
    border-bottom: 1px solid var(--border);
    padding-bottom: 0.5rem;
  }
  dt {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  dd {
    margin: 0;
    font-family: var(--font-mono);
  }
  .folio-sessions {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
    padding: 1.5rem;
  }
  .folio-sessions header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: end;
  }
  .folio-sessions header > span {
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  .folio-empty {
    margin-top: 1rem;
    color: var(--text-muted);
  }
  .folio-media-list {
    display: grid;
    gap: 0.6rem;
    margin: 1.25rem 0 0;
    padding: 0;
    list-style: none;
  }
  .folio-media-row {
    display: grid;
    grid-template-columns: 2.4rem minmax(0, 1fr) auto;
    align-items: center;
    gap: 0.7rem;
    min-width: 0;
    color: var(--text);
  }
  .folio-media-row:hover {
    color: var(--accent);
    text-decoration: none;
  }
  .folio-media-row img,
  .folio-media-thumb {
    width: 2.4rem;
    height: 2.4rem;
    border-radius: 4px;
    object-fit: cover;
  }
  .folio-media-thumb {
    display: grid;
    place-items: center;
    background: var(--surface-2);
    color: var(--accent);
    font-weight: 800;
  }
  .folio-media-copy {
    display: grid;
    gap: 0.15rem;
    min-width: 0;
  }
  .folio-media-copy strong {
    overflow: hidden;
    font-family: var(--font-serif);
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .folio-media-copy span {
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .folio-media-score {
    color: var(--accent);
    font-weight: 700;
  }
  .core-log {
    display: flex;
    gap: 0.9rem;
    height: 14rem;
    margin-top: 1.25rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    overflow: hidden;
  }
  .core-strip {
    display: flex;
    flex-direction: column;
    width: 1.4rem;
    flex-shrink: 0;
  }
  .core-band {
    flex-shrink: 0;
    border-top: 1px solid var(--bg);
  }
  .core-band:first-child {
    border-top: 0;
  }
  .core-legend {
    display: flex;
    flex: 1;
    min-width: 0;
    flex-direction: column;
  }
  .core-row {
    display: grid;
    grid-template-columns: 1.6rem minmax(0, 1fr) auto;
    align-items: center;
    gap: 0.85rem;
    min-height: 1.9rem;
    overflow: hidden;
    border-top: 1px solid var(--border);
    padding: 0 0.25rem;
    font-size: 0.78rem;
  }
  .core-row:first-child {
    border-top: 0;
  }
  .core-row b {
    color: var(--accent);
    font-family: var(--font-mono);
    font-weight: 400;
  }
  .core-row strong {
    overflow: hidden;
    color: var(--text);
    font-family: var(--font-serif);
    font-size: 0.95rem;
    font-weight: 400;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .core-row span {
    overflow: hidden;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.7rem;
    text-align: right;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .folio-source {
    border-top: 1px solid var(--border);
    padding-top: 0.9rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.66rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  @media (max-width: 680px) {
    .folio-head,
    .folio-focus,
    .folio-grid {
      display: block;
    }
    .core-gauge {
      justify-self: start;
      margin-top: 1.5rem;
    }
    .folio-grid .catalog-card + .catalog-card {
      margin-top: 1.25rem;
    }
    .core-row {
      grid-template-columns: 1.6rem minmax(0, 1fr);
    }
    .core-row span {
      grid-column: 2;
      text-align: left;
    }
  }
</style>
