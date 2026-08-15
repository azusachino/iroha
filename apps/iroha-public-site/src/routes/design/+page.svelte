<script lang="ts">
  import { goto } from "$app/navigation";
  import { base } from "$app/paths";
  import type { Activity } from "@iroha/shared/activity";
  import {
    DESIGN_COMPOSITIONS,
    designComposition,
    type DesignCompositionId,
    type DesignCompositionProps,
    type DesignTodayData,
  } from "@iroha/shared/design-compositions";
  import type { DailyRow } from "@iroha/shared/daily";
  import type { MediaChange, MediaHomeEvent } from "@iroha/shared/media";
  import type { SleepSession } from "@iroha/shared/sleep";
  import type { DesignLanguage } from "@iroha/shared/themes";
  import {
    THEME_DEFINITIONS,
    getThemeDefinition,
  } from "@iroha/shared/theme-ui/registry";
  import ThemeProvider from "@iroha/shared/theme-ui/ThemeProvider.svelte";
  import ThemeRouteRenderer from "@iroha/shared/theme-ui/ThemeRouteRenderer.svelte";
  import DesignCompositionRenderer from "@iroha/shared/theme-ui/compositions/DesignCompositionRenderer.svelte";
  import ThemeToggle from "$lib/components/ThemeToggle.svelte";

  const fixtureDate = "2026-08-14";
  let language = $state<DesignLanguage>("grapher");
  let composition = $state<DesignCompositionId>("editorial");

  const fixtureActivities: Activity[] = [
    {
      id: "public-design-run",
      sport_type: "run",
      title: "Morning run",
      started_at: `${fixtureDate}T06:42:00Z`,
      timezone: "",
      distance_m: 6420,
      duration_s: 2340,
      moving_time_s: 2290,
      avg_hr: 148,
      avg_pace_s_per_km: 357,
      source_kind: "design-fixture",
      first_raw_file_id: "design-fixture",
      created_at: `${fixtureDate}T07:30:00Z`,
      updated_at: `${fixtureDate}T07:30:00Z`,
    },
    {
      id: "public-design-walk",
      sport_type: "walk",
      title: "Evening walk",
      started_at: `${fixtureDate}T18:11:00Z`,
      timezone: "",
      distance_m: 2310,
      duration_s: 1860,
      moving_time_s: 1810,
      avg_hr: 102,
      avg_pace_s_per_km: 784,
      source_kind: "design-fixture",
      first_raw_file_id: "design-fixture",
      created_at: `${fixtureDate}T19:00:00Z`,
      updated_at: `${fixtureDate}T19:00:00Z`,
    },
  ];

  const fixtureDaily: DailyRow = {
    id: "public-design-daily",
    day: fixtureDate,
    ring: {
      move_kcal: 486,
      move_goal_kcal: 650,
      exercise_min: 34,
      exercise_goal_min: 30,
      stand_hours: 9,
      stand_goal_hours: 12,
    },
    steps: 8420,
    distance_km: 6.8,
    resting_hr: 51,
    hrv_sdnn: 58,
    spo2_avg: 98,
    respiratory_rate: 14.2,
    vo2max: 51.4,
    source: "design-fixture",
    first_raw_file_id: "design-fixture",
    created_at: `${fixtureDate}T23:59:00Z`,
    updated_at: `${fixtureDate}T23:59:00Z`,
  };

  const fixtureSleep: SleepSession = {
    id: "public-design-sleep",
    wake_date: fixtureDate,
    started_at: "2026-08-13T22:21:00Z",
    ended_at: `${fixtureDate}T06:05:00Z`,
    time_in_bed_s: 27960,
    asleep_s: 25620,
    efficiency: 0.913,
    is_main_sleep: true,
    core_s: 12300,
    deep_s: 4860,
    rem_s: 8460,
    awake_s: 2340,
    unspecified_s: 0,
    source: "design-fixture",
    first_raw_file_id: "design-fixture",
    created_at: `${fixtureDate}T06:05:00Z`,
    updated_at: `${fixtureDate}T06:05:00Z`,
  };

  const fixtureMedia: MediaHomeEvent[] = [
    {
      id: "public-design-media-event",
      media_id: "public-design-media",
      title: "A quiet chapter",
      event_type: "progress",
      occurred_at: `${fixtureDate}T21:05:00Z`,
      position: 76,
      total: 100,
      progress_percent: 76,
    },
  ];

  const fixtureMediaUpdates: MediaChange[] = [
    {
      id: "public-design-media-update",
      media_id: "public-design-media",
      title: "A quiet chapter",
      source_kind: "anilist",
      change_kind: "changed",
      time_basis: "source_date",
      observed_at: `${fixtureDate}T23:00:00Z`,
      effective_on: fixtureDate,
      date_precision: "day",
      status: "in_progress",
      unit: "chapters",
      position: 76,
      total: 100,
      progress_percent: 76,
      repeat_count: 0,
    },
  ];

  const todayProps: Omit<
    DesignCompositionProps,
    "today" | "readiness" | "links"
  > & {
    dayLabel: string;
    day: string;
    dRow: DailyRow;
    mainNight: SleepSession;
    acts: Activity[];
    mediaEvents: MediaHomeEvent[];
    mediaUpdates: MediaChange[];
    onOpenActivity: (id: string) => void;
    onOpenMedia: (id: string) => void;
  } = {
    dayLabel: fixtureDate,
    day: fixtureDate,
    dRow: fixtureDaily,
    mainNight: fixtureSleep,
    acts: fixtureActivities,
    mediaEvents: fixtureMedia,
    mediaUpdates: fixtureMediaUpdates,
    onOpenActivity: (id) =>
      void goto(`${base}/?activity=${encodeURIComponent(id)}`),
    onOpenMedia: () => void goto(`${base}/`),
  };

  const designToday: DesignTodayData = {
    date: fixtureDate,
    daily: {
      ring: fixtureDaily.ring,
      steps: fixtureDaily.steps,
      distance_km: fixtureDaily.distance_km,
      resting_hr: fixtureDaily.resting_hr,
      hrv_sdnn: fixtureDaily.hrv_sdnn,
      spo2_avg: fixtureDaily.spo2_avg,
      vo2max: fixtureDaily.vo2max,
    },
    sleep: {
      asleep_s: fixtureSleep.asleep_s,
      efficiency: fixtureSleep.efficiency,
      deep_s: fixtureSleep.deep_s,
    },
    activities: fixtureActivities.map((activity) => ({
      id: activity.id,
      sport_type: activity.sport_type,
      title: activity.title,
      started_at: activity.started_at,
      distance_m: activity.distance_m,
      moving_time_s: activity.moving_time_s,
      duration_s: activity.duration_s,
      avg_hr: activity.avg_hr,
      avg_pace_s_per_km: activity.avg_pace_s_per_km,
    })),
    media: fixtureMedia.map((event) => ({
      id: event.id,
      title: event.title,
      progress_percent: event.progress_percent,
    })),
  };

  const readiness = 87;
  const compositionLinks = {
    motion: `${base}/`,
    night: `${base}/design`,
    patterns: `${base}/design`,
    library: `${base}/`,
    activity: (id: string) => `${base}/?activity=${encodeURIComponent(id)}`,
  };

  const selectedLanguage = $derived(getThemeDefinition(language));
  const selectedComposition = $derived(
    DESIGN_COMPOSITIONS.find((item) => item.id === composition) ??
      DESIGN_COMPOSITIONS[0],
  );

  function selectLanguage(next: DesignLanguage): void {
    language = next;
  }

  function selectComposition(next: DesignCompositionId): void {
    composition = designComposition(next);
  }
</script>

<svelte:head><title>Design workshop · {"harus tracks"}</title></svelte:head>

<ThemeProvider {language} onSelect={selectLanguage}>
  <main class="design-lab" data-language={language}>
    <header class="lab-header tile">
      <div>
        <a class="back-link" href={`${base}/`}>← Public archive</a>
        <p class="eyebrow">Shared design system workbench</p>
        <h1>One payload. Many ways to see it.</h1>
        <p class="lead">
          This page is a public fixture consumer of the same package-owned
          compositions used by iroha-web. The fixture is illustrative; it is not
          private health data.
        </p>
      </div>
      <ThemeToggle />
    </header>

    <section class="control-panel tile" aria-labelledby="language-title">
      <div class="control-heading">
        <div>
          <p class="eyebrow">Registered production language</p>
          <h2 id="language-title">{selectedLanguage.identity.label}</h2>
          <p class="muted">{selectedLanguage.identity.description}</p>
        </div>
        <span class="fixture-pill">Representative fixture · {fixtureDate}</span>
      </div>
      <div class="language-grid" aria-label="Registered design languages">
        {#each THEME_DEFINITIONS as definition}
          <button
            type="button"
            class:active={language === definition.identity.id}
            aria-pressed={language === definition.identity.id}
            onclick={() => selectLanguage(definition.identity.id)}
          >
            <span
              class="language-swatch"
              style={`--swatch:${definition.identity.swatch}`}
            ></span>
            <span>
              <strong>{definition.identity.label.replace("Iroha ", "")}</strong>
              <small>{definition.identity.hint}</small>
            </span>
          </button>
        {/each}
      </div>
    </section>

    <section class="production-specimen" aria-labelledby="production-title">
      <header class="section-heading">
        <div>
          <p class="eyebrow">Registered route specimen</p>
          <h2 id="production-title">
            Today · {selectedLanguage.identity.label}
          </h2>
        </div>
        <span class="route-note">Shared Today view contract</span>
      </header>
      <div class="render-frame" data-theme={language}>
        <ThemeRouteRenderer route="today" props={todayProps} />
      </div>
    </section>

    <section class="composition-specimen" aria-labelledby="composition-title">
      <header class="control-heading">
        <div>
          <p class="eyebrow">Adopted composition</p>
          <h2 id="composition-title">{selectedComposition.label}</h2>
          <p class="muted">{selectedComposition.intent}</p>
        </div>
      </header>
      <nav class="composition-tabs" aria-label="Adopted compositions">
        {#each DESIGN_COMPOSITIONS as item}
          <button
            type="button"
            class:active={composition === item.id}
            aria-pressed={composition === item.id}
            onclick={() => selectComposition(item.id)}
          >
            <span>{item.index}</span>{item.label}
          </button>
        {/each}
      </nav>
      <div class="render-frame" data-composition={composition}>
        <DesignCompositionRenderer
          {composition}
          today={designToday}
          {readiness}
          links={compositionLinks}
        />
      </div>
    </section>
  </main>
</ThemeProvider>

<style>
  .design-lab {
    display: grid;
    gap: 1.25rem;
    padding-bottom: 2rem;
  }

  .lab-header,
  .control-panel,
  .production-specimen,
  .composition-specimen {
    padding: clamp(1rem, 3vw, 1.5rem);
  }

  .lab-header,
  .control-heading,
  .section-heading {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1.25rem;
  }

  .back-link {
    display: inline-block;
    margin-bottom: 1rem;
    font-size: 0.8rem;
    font-weight: 700;
  }

  .eyebrow {
    margin: 0 0 0.4rem;
    color: var(--accent);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  h1,
  h2,
  p {
    margin-top: 0;
  }

  h1 {
    max-width: 13ch;
    margin-bottom: 0.7rem;
    font-size: clamp(2rem, 6vw, 4.6rem);
    letter-spacing: -0.07em;
    line-height: 0.94;
  }

  h2 {
    margin-bottom: 0.35rem;
    letter-spacing: -0.04em;
  }

  .lead,
  .muted {
    color: var(--text-muted);
    line-height: 1.55;
  }

  .lead {
    max-width: 42rem;
    margin-bottom: 0;
  }

  .fixture-pill,
  .route-note {
    flex: 0 0 auto;
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 0.4rem 0.65rem;
    color: var(--text-muted);
    font-size: 0.72rem;
  }

  .control-panel,
  .production-specimen,
  .composition-specimen {
    display: grid;
    gap: 1rem;
  }

  .control-heading .muted {
    margin-bottom: 0;
  }

  .language-grid {
    display: grid;
    grid-template-columns: repeat(6, minmax(0, 1fr));
    gap: 0.5rem;
  }

  .language-grid button,
  .composition-tabs button {
    display: flex;
    align-items: center;
    gap: 0.55rem;
    min-width: 0;
    border: 1px solid var(--border);
    border-radius: 0.65rem;
    padding: 0.65rem;
    background: var(--surface-2);
    color: var(--text-muted);
    cursor: pointer;
    text-align: left;
  }

  .language-grid button.active,
  .composition-tabs button.active {
    border-color: var(--accent);
    color: var(--text);
    box-shadow: 0 0 0 1px color-mix(in srgb, var(--accent) 30%, transparent);
  }

  .language-grid strong,
  .language-grid small {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .language-grid strong {
    font-size: 0.78rem;
  }

  .language-grid small {
    margin-top: 0.2rem;
    color: var(--text-muted);
    font-size: 0.66rem;
  }

  .language-swatch {
    width: 0.65rem;
    height: 0.65rem;
    flex: 0 0 auto;
    border-radius: 50%;
    background: var(--swatch);
    box-shadow: 0 0 0 4px color-mix(in srgb, var(--swatch) 16%, transparent);
  }

  .section-heading {
    align-items: end;
    padding: 0 0.15rem;
  }

  .section-heading h2 {
    margin-bottom: 0;
  }

  .render-frame {
    overflow: hidden;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }

  .composition-tabs {
    display: flex;
    flex-wrap: wrap;
    gap: 0.45rem;
  }

  .composition-tabs button {
    flex: 0 0 auto;
    border-radius: 999px;
    padding: 0.45rem 0.7rem;
    font-size: 0.74rem;
  }

  .composition-tabs button span {
    color: var(--accent);
    font-weight: 800;
  }

  @media (max-width: 1024px) {
    .language-grid {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
  }

  @media (max-width: 640px) {
    .lab-header,
    .control-heading,
    .section-heading {
      display: grid;
    }

    .language-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }
</style>
