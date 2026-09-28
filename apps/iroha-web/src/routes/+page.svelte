<script lang="ts">
  import { goto } from "$app/navigation";
  import { Check, ListTodo } from "@lucide/svelte";
  import RingGauge from "@iroha/shared/theme-ui/components/RingGauge.svelte";
  import SportBadge from "@iroha/shared/components/SportBadge.svelte";
  import DayPicker from "$lib/components/DayPicker.svelte";
  import { useTheme } from "$lib/themes/context.svelte";
  import ThemeRouteRenderer from "@iroha/shared/theme-ui/ThemeRouteRenderer.svelte";
  import { hasThemeRoute } from "$lib/themes/registry";
  import {
    formatDistance,
    formatDuration,
    formatPace,
    formatHr,
    mediaEventVerb,
  } from "$lib/format";
  import EmptyState from "@iroha/shared/theme-ui/components/EmptyState.svelte";
  import TodaySkeleton from "$lib/components/TodaySkeleton.svelte";
  import MediaUpdateList from "@iroha/shared/theme-ui/components/MediaUpdateList.svelte";
  import { createTodayState } from "./today-state.svelte";

  const theme = useTheme();
  const t = createTodayState();
</script>

<svelte:head>
  <title>Today · iroha</title>
</svelte:head>

<svelte:window onkeydown={t.onKey} />

<section class="cockpit">
  {#if !t.briefing || !t.dayHasData}
    <h1 class="visually-hidden">Today, {t.dayLabel}</h1>
  {/if}
  <div class="scrubber tile glow">
    <button class="nav" aria-label="Previous day" onclick={() => t.shift(-1)}
      >‹</button
    >
    <button
      class="scrub-center"
      aria-haspopup="dialog"
      aria-expanded={t.pickerOpen}
      onclick={() => (t.pickerOpen = !t.pickerOpen)}
    >
      <span class="day-main">{t.dayLabel}</span>
      <span class="day-hint">pick a day · ← → to move</span>
    </button>
    <button
      class="nav"
      aria-label="Next day"
      disabled={!t.canMoveNext}
      onclick={() => t.shift(1)}>›</button
    >
    {#if t.pickerOpen}
      <button
        class="pk-backdrop"
        aria-label="Close picker"
        onclick={() => (t.pickerOpen = false)}
      ></button>
      <div class="pk-pop tile" role="dialog" aria-label="Pick a day">
        <DayPicker
          value={t.day}
          days={t.daysSet}
          max={t.today}
          onselect={(d) => {
            t.day = d;
            t.pickerOpen = false;
          }}
        />
        <button
          class="today-link"
          type="button"
          onclick={() => {
            t.day = t.today;
            t.pickerOpen = false;
          }}
        >
          Return to today
        </button>
      </div>
    {/if}

    {#if !t.taskError}
      <section class="to-go-strip tile" aria-label="Daily to-go">
        <div class="to-go-heading">
          <span class="to-go-icon" aria-hidden="true"
            ><ListTodo size={17} /></span
          >
          <div>
            <p class="eyebrow">Daily to-go</p>
            <p class="to-go-title">
              {t.toGoTasks.length
                ? `${t.toGoTasks.length} things to carry`
                : "A clear next step"}
            </p>
          </div>
        </div>
        <div class="to-go-items">
          {#if t.toGoTasks.length}
            {#each t.toGoTasks as task (task.id)}
              <div class="to-go-task">
                <button
                  type="button"
                  aria-label={`Complete ${task.title}`}
                  onclick={() => t.finishTask(task)}><Check size={14} /></button
                >
                <span>{task.title}</span>
              </div>
            {/each}
          {:else}
            <span class="to-go-empty">No open tasks for this day.</span>
          {/if}
        </div>
        <a class="to-go-link" href="/to-go">Open control room →</a>
      </section>
    {/if}
  </div>

  {#if !t.briefing && t.loading}
    <TodaySkeleton label={`Loading ${t.dayLabel}…`} />
  {:else if !t.briefing && t.error}
    <p class="error status" role="alert">Could not load data: {t.error}</p>
  {:else if t.briefing}
    <div class="briefing-surface" aria-busy={t.loading}>
      {#if t.loading}
        <p class="briefing-update" role="status" aria-live="polite">
          Updating {t.dayLabel}…
        </p>
      {/if}
      {#if t.error}
        <p class="error update-error" role="alert">
          Could not update {t.dayLabel}; showing {t.dataDayLabel}.
        </p>
      {/if}
      <div class="briefing-content">
        {#if !t.dayHasData}
          <EmptyState
            eyebrow="Quiet day"
            title={`No records for ${t.dataDayLabel}.`}
            description="The canonical cockpit has no imported movement, recovery, health, media, or task records for this date."
            actionHref={!t.canJumpToLatestDay && t.dataDay !== t.today
              ? "/"
              : undefined}
            actionLabel={t.canJumpToLatestDay
              ? "Jump to latest recorded day"
              : "Return to today"}
            actionOnclick={t.canJumpToLatestDay ? t.jumpToLatestDay : undefined}
          />
        {:else if hasThemeRoute(theme.definition(), "today")}
          <ThemeRouteRenderer
            route="today"
            props={{
              dayLabel: t.dataDayLabel,
              day: t.dataDay,
              dRow: t.dRow,
              mainNight: t.mainNight,
              acts: t.acts,
              mediaEvents: t.mediaEvents,
              mediaUpdates: t.mediaUpdates,
              onOpenActivity: (id: string) => void goto(`/motion/${id}`),
              onOpenMedia: (id: string) => void goto(`/library/${id}`),
            }}
          />
        {:else}
          <header class="command-heading tile hero-surface">
            <div>
              <p class="eyebrow">Private command center / {t.dataDayLabel}</p>
              <h1>Today, in one view.</h1>
              <p class="heading-copy">
                A calm operating view of movement, recovery, and the things you
                touched today.
              </p>
              <p class="data-note">
                Imported snapshot · {t.dataDay === t.today
                  ? "latest available day"
                  : "historical day"}
              </p>
            </div>
            <div class="day-orbit" aria-label="Day signal">
              <div class="orbit orbit-one"></div>
              <div class="orbit orbit-two"></div>
              <div class="orbit-core">
                <strong>{t.daySignal.value}</strong>
                <span>{t.daySignal.label}</span>
              </div>
              <i class="orbit-star star-one"></i>
              <i class="orbit-star star-two"></i>
            </div>
          </header>
          <div class="home-kpis" aria-label="Today summary">
            <div class="home-kpi tile">
              <span>Move</span>
              <strong
                >{t.num(t.dailyRing?.move_kcal, 0)}<small> kcal</small></strong
              >
              <i
                style={"--fill:" +
                  Math.min(
                    100,
                    ((t.dailyRing?.move_kcal ?? 0) /
                      (t.dailyRing?.move_goal_kcal || 1)) *
                      100,
                  ) +
                  "%"}
              ></i>
            </div>
            <div class="home-kpi tile">
              <span>Exercise</span>
              <strong
                >{t.num(t.dailyRing?.exercise_min, 0)}<small>
                  min</small
                ></strong
              >
              <i
                style={"--fill:" +
                  Math.min(
                    100,
                    ((t.dailyRing?.exercise_min ?? 0) /
                      (t.dailyRing?.exercise_goal_min || 1)) *
                      100,
                  ) +
                  "%"}
              ></i>
            </div>
            <div class="home-kpi tile">
              <span>Steps</span>
              <strong>{t.num(t.dRow?.steps, 0)}</strong>
              <small class="kpi-note"
                >{t.num(t.dRow?.distance_km, 1)} km walked</small
              >
            </div>
            <div class="home-kpi tile">
              <span>Recovery</span>
              <strong
                >{t.mainNight
                  ? Math.round(t.mainNight.efficiency * 100) + "%"
                  : "—"}</strong
              >
              <small class="kpi-note">sleep efficiency</small>
            </div>
          </div>
          <div class="bento signal-layout">
            <!-- Rings -->
            <a class="card tile feature-card" href="/patterns">
              <header><span class="ic">◎</span> Activity rings</header>
              {#if t.hasRing}
                <RingGauge rings={t.ringData} size={116} />
                <div class="mini-stats">
                  <span>{t.num(t.dRow?.steps, 0)} steps</span>
                  <span>{t.num(t.dRow?.distance_km, 1)} km</span>
                  <span>{t.num(t.dRow?.flights, 0)} flights</span>
                </div>
              {:else}
                <p class="empty">No rings this day</p>
              {/if}
            </a>

            <!-- Vitals -->
            <a class="card tile vitals-card" href="/patterns">
              <header><span class="ic">♥</span> Body vitals</header>
              {#if t.vitals.length}
                <dl class="vitals">
                  {#each t.vitals as m}
                    <div>
                      <dt>{m.l}</dt>
                      <dd>{t.num(m.v, m.d)}<span class="u">{m.u}</span></dd>
                    </div>
                  {/each}
                </dl>
              {:else}
                <p class="empty">No vitals this day</p>
              {/if}
            </a>

            <!-- Sleep -->
            <a class="card tile sleep-card" href="/night">
              <header><span class="ic">☾</span> Sleep</header>
              {#if t.mainNight}
                <div class="sleep-hero">
                  {formatDuration(t.mainNight.asleep_s)}
                </div>
                <div class="mini-stats">
                  <span
                    >{Math.round(t.mainNight.efficiency * 100)}% efficiency</span
                  >
                  <span>{formatDuration(t.mainNight.time_in_bed_s)} in bed</span
                  >
                  <span>{t.mainNight.is_main_sleep ? "Main sleep" : "Nap"}</span
                  >
                </div>
              {:else}
                <p class="empty">No sleep recorded</p>
              {/if}
            </a>

            <!-- Activities: each row links to its own detail page. -->
            <div class="card tile wide activity-card">
              <header>
                <span class="ic">⚡</span>
                <a class="hdr-link" href="/motion">Motion</a>
              </header>
              {#if t.acts.length}
                <ul class="acts">
                  {#each t.acts as a}
                    <li>
                      <a class="act-row" href={`/motion/${a.id}`}>
                        <SportBadge sport={a.sport_type} />
                        <span class="a-title">{a.title || "Untitled"}</span>
                        <span class="a-metrics">
                          {#if a.distance_m}{formatDistance(a.distance_m)} ·
                          {/if}{formatDuration(
                            a.duration_s ?? a.moving_time_s,
                          )}{#if a.avg_pace_s_per_km}
                            · {formatPace(
                              a.avg_pace_s_per_km,
                            )}{/if}{#if a.avg_hr}
                            · {formatHr(a.avg_hr)}{/if}
                        </span>
                      </a>
                    </li>
                  {/each}
                </ul>
              {:else}
                <p class="empty">No activities this day</p>
              {/if}
            </div>

            <!-- Media events: the selected-day slice of the media history. -->
            <div class="card tile wide media-card">
              <header>
                <span class="ic">▤</span>
                <a class="hdr-link" href="/library">Library</a>
              </header>
              {#if t.mediaEvents.length}
                <ul class="media-events">
                  {#each t.mediaEvents as event (event.id)}
                    <li>
                      <a
                        class="media-event-row"
                        href={`/library/${event.media_id}`}
                      >
                        {#if event.cover_image_url}
                          <img
                            src={event.cover_image_url}
                            alt=""
                            loading="lazy"
                          />
                        {:else}
                          <span class="media-thumb" aria-hidden="true"
                            >{(event.native_title || event.title).slice(
                              0,
                              1,
                            )}</span
                          >
                        {/if}
                        <span class="media-event-copy">
                          <strong>{event.native_title || event.title}</strong>
                          <span>{mediaEventVerb(event)}</span>
                        </span>
                        {#if event.rating != null}<span class="media-score"
                            >{event.rating.toFixed(1)}</span
                          >{/if}
                      </a>
                    </li>
                  {/each}
                </ul>
              {:else}
                <p class="empty">No exact media sessions this day</p>
              {/if}
            </div>
            {#if t.mediaUpdates.length}
              <div class="card tile wide media-card media-updates-card">
                <header>
                  <span class="ic">↻</span>
                  <div>
                    <span class="hdr-link">Library updates</span>
                    <small>dated provider facts</small>
                  </div>
                  <span>{t.mediaUpdates.length} updates</span>
                </header>
                <MediaUpdateList
                  updates={t.mediaUpdates}
                  onOpenMedia={(id) => void goto(`/library/${id}`)}
                />
              </div>
            {/if}
          </div>
        {/if}
      </div>
    </div>
  {/if}
</section>

<style>
  .cockpit {
    display: grid;
    gap: 1.25rem;
  }
  .to-go-strip {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr) auto;
    gap: 1rem;
    align-items: center;
    padding: 0.8rem 1rem;
    background: color-mix(in srgb, var(--surface) 92%, var(--accent));
  }
  .to-go-heading {
    display: flex;
    align-items: center;
    gap: 0.65rem;
  }
  .to-go-icon {
    display: grid;
    width: 2.2rem;
    height: 2.2rem;
    place-items: center;
    border: 1px solid color-mix(in srgb, var(--accent) 40%, var(--border));
    border-radius: 50%;
    color: var(--accent);
  }
  .to-go-heading .eyebrow {
    margin-bottom: 0.1rem;
  }
  .to-go-title {
    margin: 0;
    font-size: 0.98rem;
    font-weight: 700;
  }
  .to-go-items {
    display: flex;
    flex-wrap: wrap;
    gap: 0.35rem 0.8rem;
    min-width: 0;
  }
  .to-go-task {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    min-width: 0;
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .to-go-task span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .to-go-task button {
    display: grid;
    flex: 0 0 auto;
    width: 1.35rem;
    height: 1.35rem;
    place-items: center;
    border: 1px solid var(--border);
    border-radius: 50%;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
  }
  .to-go-task button:hover {
    border-color: var(--accent);
    color: var(--accent);
  }
  .to-go-empty {
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .to-go-link {
    color: var(--accent);
    font-size: 0.75rem;
    white-space: nowrap;
  }
  .status {
    padding: 2rem 0;
  }
  .error {
    color: var(--danger);
  }
  .briefing-surface {
    position: relative;
    min-width: 0;
  }
  .briefing-content {
    min-width: 0;
  }
  .briefing-update,
  .update-error {
    margin: 0 0 0.65rem;
    font-size: 0.75rem;
  }
  .briefing-update {
    /* Keep the live announcement out of layout so date scrubbing cannot
       move the whole data surface while the next snapshot is loading. */
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    overflow: hidden;
    clip: rect(0 0 0 0);
    white-space: nowrap;
    border: 0;
    margin: -1px;
  }
  .update-error {
    color: var(--danger);
  }
  .command-heading {
    position: relative;
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 1.5rem;
    min-height: 22rem;
    padding: 2.5rem 2.5rem 2rem;
    overflow: hidden;
  }
  .hero-surface {
    background:
      radial-gradient(
        circle at 78% 45%,
        color-mix(in srgb, var(--accent) 18%, transparent),
        transparent 19rem
      ),
      linear-gradient(
        135deg,
        color-mix(in srgb, var(--surface) 92%, var(--accent)),
        var(--surface)
      );
  }
  .hero-surface::before {
    position: absolute;
    inset: 0;
    background: linear-gradient(
      115deg,
      transparent 0 48%,
      color-mix(in srgb, var(--accent) 8%, transparent) 48.2% 48.5%,
      transparent 48.7%
    );
    content: "";
    pointer-events: none;
  }
  .command-heading > div:first-child {
    position: relative;
    z-index: 1;
  }
  .eyebrow {
    margin: 0 0 0.45rem;
    color: var(--accent);
    font-size: 0.7rem;
    font-weight: 750;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .command-heading h1 {
    margin: 0;
    max-width: 9ch;
    font-size: clamp(3.2rem, 8vw, 7rem);
    letter-spacing: -0.1em;
    line-height: 0.84;
  }
  .heading-copy {
    max-width: 34rem;
    margin: 0.7rem 0 0;
    color: var(--text-muted);
    line-height: 1.5;
  }
  .data-note {
    margin: 0.75rem 0 0;
    color: var(--text-muted);
    font-size: 0.72rem;
    letter-spacing: 0.02em;
  }
  .day-orbit {
    position: relative;
    display: grid;
    width: min(32rem, 44vw);
    min-width: 18rem;
    height: 18rem;
    place-items: center;
    overflow: hidden;
  }
  .orbit {
    position: absolute;
    width: 13rem;
    height: 13rem;
    border: 1px solid color-mix(in srgb, var(--accent) 42%, transparent);
    border-radius: 50%;
    transform: rotate(30deg) scaleX(1.55);
  }
  .orbit-two {
    width: 9rem;
    height: 9rem;
    border-color: color-mix(in srgb, var(--accent-2) 48%, transparent);
    transform: rotate(-38deg) scaleX(1.7);
  }
  .orbit-core {
    position: relative;
    z-index: 1;
    display: grid;
    width: 8.5rem;
    height: 8.5rem;
    place-items: center;
    align-content: center;
    border: 1px solid color-mix(in srgb, var(--accent) 62%, transparent);
    border-radius: 50%;
    background: radial-gradient(
      circle at 35% 25%,
      color-mix(in srgb, var(--accent) 30%, transparent),
      var(--surface-2) 72%
    );
    box-shadow: 0 0 60px color-mix(in srgb, var(--accent) 22%, transparent);
  }
  .orbit-core strong {
    font-size: 4rem;
    letter-spacing: -0.12em;
    line-height: 0.75;
  }
  .orbit-core span {
    margin-top: 0.55rem;
    color: var(--text-muted);
    font-size: 0.62rem;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .orbit-star {
    position: absolute;
    width: 0.45rem;
    height: 0.45rem;
    border-radius: 50%;
    background: var(--accent);
    box-shadow: 0 0 16px var(--accent);
  }
  .star-one {
    top: 22%;
    left: 20%;
  }
  .star-two {
    right: 16%;
    bottom: 22%;
    background: var(--accent-2);
    box-shadow: 0 0 16px var(--accent-2);
  }
  .home-kpis {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 0.75rem;
    position: relative;
    z-index: 1;
  }
  .home-kpi {
    display: grid;
    gap: 0.35rem;
    padding: 0.9rem;
    border-radius: calc(var(--radius) - 4px);
    background: color-mix(in srgb, var(--surface) 78%, transparent);
  }
  .home-kpi > span {
    color: var(--text-muted);
    font-size: 0.67rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .home-kpi strong {
    font-size: 1.45rem;
    letter-spacing: -0.06em;
  }
  .home-kpi strong small {
    color: var(--text-muted);
    font-size: 0.45em;
    font-weight: 500;
    letter-spacing: 0;
  }
  .home-kpi > i {
    display: block;
    width: 100%;
    height: 0.2rem;
    border-radius: 99px;
    background: linear-gradient(
      90deg,
      var(--accent) var(--fill, 0%),
      var(--surface-2) var(--fill, 0%)
    );
  }
  .home-kpi:nth-child(2) > i {
    background: linear-gradient(
      90deg,
      var(--ring-exercise) var(--fill, 0%),
      var(--surface-2) var(--fill, 0%)
    );
  }
  .kpi-note {
    color: var(--text-muted);
    font-size: 0.7rem;
  }

  /* Date scrubber — the spine control, gets the neon chrome glow. */
  .scrubber {
    position: relative;
    display: flex;
    align-items: center;
    gap: 0.75rem;
    min-width: 0;
    padding: 0.85rem 1rem;
  }
  .glow {
    border: 1px solid color-mix(in srgb, var(--accent) 34%, var(--border));
    box-shadow: var(--accent-glow), var(--tile-shadow);
  }
  .scrub-center {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.1rem;
    appearance: none;
    border: 0;
    background: transparent;
    cursor: pointer;
    border-radius: var(--radius);
    padding: 0.25rem;
  }
  .scrub-center:hover {
    background: color-mix(in srgb, var(--accent) 8%, transparent);
  }
  .day-main {
    color: var(--accent);
    font-size: 1.1rem;
    font-weight: 700;
    text-shadow: 0 0 16px color-mix(in srgb, var(--accent) 28%, transparent);
  }
  .day-hint {
    font-size: 0.72rem;
    color: var(--text-muted);
  }
  .pk-backdrop {
    position: fixed;
    inset: 0;
    z-index: 20;
    border: 0;
    background: transparent;
    cursor: default;
  }
  .pk-pop {
    position: absolute;
    z-index: 21;
    top: calc(100% + 0.5rem);
    left: 50%;
    transform: translateX(-50%);
    box-shadow: var(--accent-glow), var(--tile-shadow);
    border: 1px solid color-mix(in srgb, var(--accent) 30%, var(--border));
  }
  .nav {
    appearance: none;
    border: 1px solid var(--border);
    background: var(--surface);
    color: var(--text);
    width: 2.2rem;
    height: 2.2rem;
    border-radius: 50%;
    font-size: 1.2rem;
    line-height: 1;
    cursor: pointer;
  }
  .nav:hover {
    border-color: var(--accent);
    color: var(--accent);
  }
  .nav:disabled {
    cursor: default;
    opacity: 0.35;
  }
  .today-link {
    display: block;
    width: calc(100% - 1.5rem);
    margin: 0 auto 0.7rem;
    padding: 0.45rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface-2);
    color: var(--text-muted);
    font: inherit;
    font-size: 0.75rem;
    cursor: pointer;
  }
  .today-link:hover {
    border-color: var(--accent);
    color: var(--text);
  }
  .bento {
    display: grid;
    grid-template-columns: repeat(12, minmax(0, 1fr));
    gap: 1rem;
  }
  .card {
    padding: 1.1rem;
    display: flex;
    flex-direction: column;
    gap: 0.85rem;
    color: var(--text);
    min-height: 11rem;
    overflow: hidden;
    border-radius: calc(var(--radius) + 2px);
    background:
      linear-gradient(
        145deg,
        color-mix(in srgb, var(--surface) 96%, var(--accent)),
        var(--surface)
      ),
      var(--surface);
  }
  .card:hover {
    text-decoration: none;
    border-color: color-mix(in srgb, var(--accent) 45%, var(--border));
  }
  .card.wide {
    grid-column: span 7;
  }
  .feature-card {
    grid-column: span 5;
    grid-row: span 2;
    min-height: 24rem;
    justify-content: space-between;
    background:
      radial-gradient(
        circle at 90% 10%,
        color-mix(in srgb, var(--accent-2) 17%, transparent),
        transparent 14rem
      ),
      linear-gradient(
        145deg,
        color-mix(in srgb, var(--surface) 86%, var(--accent)),
        var(--surface)
      );
  }
  .vitals-card {
    grid-column: span 7;
    min-height: 11.5rem;
  }
  .sleep-card {
    grid-column: span 7;
    min-height: 11.5rem;
  }
  .activity-card {
    min-height: 16rem;
  }
  .media-card {
    min-height: 16rem;
  }
  .card header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.82rem;
    font-weight: 650;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    padding-bottom: 0.75rem;
    border-bottom: 1px solid color-mix(in srgb, var(--border) 70%, transparent);
  }
  .ic {
    color: var(--accent);
    font-size: 1rem;
  }
  .empty {
    color: var(--text-muted);
    font-size: 0.88rem;
    margin: auto 0;
  }

  .mini-stats {
    display: flex;
    flex-wrap: wrap;
    gap: 0.35rem 1rem;
    color: var(--text-muted);
    font-size: 0.82rem;
  }
  .sleep-hero {
    font-size: 1.9rem;
    font-weight: 700;
    color: var(--accent);
    text-shadow: 0 0 18px color-mix(in srgb, var(--accent) 30%, transparent);
  }

  .vitals {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.55rem 1rem;
    margin: 0;
  }
  .vitals div {
    display: flex;
    flex-direction: column;
  }
  .vitals dt {
    font-size: 0.72rem;
    color: var(--text-muted);
  }
  .vitals dd {
    margin: 0;
    font-size: 1.05rem;
    font-weight: 650;
  }
  .vitals .u {
    font-size: 0.72rem;
    font-weight: 400;
    color: var(--text-muted);
    margin-left: 0.15rem;
  }

  .acts {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 0.6rem;
  }
  .acts li {
    min-width: 0;
  }
  .act-row {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr);
    align-items: center;
    gap: 0.3rem 0.6rem;
    min-width: 0;
    color: var(--text);
    padding: 0.2rem 0;
  }
  .act-row:hover {
    color: var(--accent);
    text-decoration: none;
  }
  .hdr-link {
    color: inherit;
  }
  .hdr-link:hover {
    color: var(--accent);
  }
  .a-title {
    min-width: 0;
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .a-metrics {
    grid-column: 2;
    min-width: 0;
    color: var(--text-muted);
    font-size: 0.8rem;
    overflow-wrap: anywhere;
  }
  .media-events {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 0.55rem;
  }
  .media-event-row {
    display: grid;
    grid-template-columns: 2.2rem minmax(0, 1fr) auto;
    align-items: center;
    gap: 0.65rem;
    min-width: 0;
    color: var(--text);
  }
  .media-event-row:hover {
    color: var(--accent);
    text-decoration: none;
  }
  .media-event-row img,
  .media-thumb {
    width: 2.2rem;
    height: 2.2rem;
    object-fit: cover;
    border-radius: 4px;
  }
  .media-thumb {
    display: grid;
    place-items: center;
    background: var(--surface-2);
    color: var(--accent);
    font-weight: 800;
  }
  .media-event-copy {
    min-width: 0;
    display: grid;
    gap: 0.15rem;
  }
  .media-event-copy strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 0.82rem;
  }
  .media-event-copy span {
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .media-score {
    color: var(--accent);
    font-size: 0.78rem;
    font-weight: 750;
  }

  @media (max-width: 1024px) {
    .to-go-strip {
      grid-template-columns: 1fr auto;
    }
    .to-go-items {
      grid-column: 1 / -1;
      grid-row: 2;
    }
    .command-heading {
      align-items: flex-start;
      flex-direction: column;
      padding: 1.5rem;
    }
    .command-heading {
      min-height: 0;
    }
    .day-orbit {
      width: 100%;
      min-width: 0;
      height: 15rem;
    }
    .home-kpis {
      grid-template-columns: repeat(2, 1fr);
    }
    .bento {
      grid-template-columns: 1fr;
    }
    .card.wide,
    .feature-card,
    .vitals-card,
    .sleep-card {
      grid-column: span 1;
      grid-row: auto;
    }
  }

  @media (max-width: 640px) {
    .scrubber {
      flex-wrap: wrap;
      gap: 0.45rem;
    }

    .scrub-center {
      min-width: 0;
    }

    .day-hint {
      max-width: 100%;
      overflow-wrap: anywhere;
      text-align: center;
    }

    .to-go-strip {
      width: 100%;
      grid-template-columns: 1fr;
      gap: 0.55rem;
    }

    .to-go-items {
      grid-column: auto;
      grid-row: auto;
    }

    .to-go-link {
      justify-self: start;
    }
  }
</style>
