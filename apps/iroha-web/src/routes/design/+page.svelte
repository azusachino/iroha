<script lang="ts">
  import { replaceState } from "$app/navigation";
  import { page } from "$app/state";
  import { onMount } from "svelte";
  import type { DesignLanguage } from "@iroha/shared/themes";
  import {
    DESIGN_COMPOSITIONS,
    designComposition,
    designDuration,
    type DesignCompositionId,
    type DesignTodayData,
  } from "@iroha/shared/design-compositions";
  import DesignCompositionRenderer from "@iroha/shared/theme-ui/compositions/DesignCompositionRenderer.svelte";
  import {
    getBriefing,
    type Activity as ActivityRecord,
    type DailyRow,
    type MediaHomeEvent,
    type SleepSession,
  } from "$lib/api";
  import { useTheme } from "$lib/themes/context.svelte";
  import { THEME_DEFINITIONS } from "$lib/themes/registry";
  import { todayInTimezone } from "@iroha/shared/date";

  let variant = $state<DesignCompositionId>(
    designComposition(page.url.searchParams.get("composition")),
  );
  let today = $state<DesignTodayData>(sampleToday());
  let source = $state<"sample" | "live">("sample");
  const theme = useTheme();

  const readiness = $derived(
    today.sleep && today.daily?.resting_hr
      ? Math.round(
          Math.min(
            99,
            Math.max(
              42,
              today.sleep.efficiency * 100 -
                Math.max(0, today.daily.resting_hr - 52) * 1.4,
            ),
          ),
        )
      : 78,
  );

  const compositionLinks = {
    motion: "/motion",
    night: "/night",
    patterns: "/patterns",
    library: "/library",
    activity: (id: string) => "/motion/" + id,
  };

  onMount(() => {
    void loadToday();
  });

  async function loadToday() {
    try {
      const briefing = await getBriefing(todayInTimezone());
      const live = fromBriefing(briefing);
      if (
        live.daily ||
        live.sleep ||
        live.activities.length > 0 ||
        live.media.length > 0
      ) {
        today = { ...live, ...sampleFallback(live) };
        source = "live";
      }
    } catch {
      source = "sample";
    }
  }

  function sectionItems<T>(
    briefing: Awaited<ReturnType<typeof getBriefing>>,
    key: string,
  ): T[] {
    const section = briefing.sections.find((item) => item.key === key);
    return (section?.data as { items?: T[] } | undefined)?.items ?? [];
  }

  function fromBriefing(
    briefing: Awaited<ReturnType<typeof getBriefing>>,
  ): DesignTodayData {
    const daily = sectionItems<DailyRow>(briefing, "daily")[0];
    const sleep =
      sectionItems<SleepSession>(briefing, "sleep").find(
        (item) => item.is_main_sleep,
      ) ?? sectionItems<SleepSession>(briefing, "sleep")[0];
    return {
      date: briefing.date,
      daily: daily
        ? {
            ring: daily.ring
              ? {
                  move_kcal: daily.ring.move_kcal,
                  move_goal_kcal: daily.ring.move_goal_kcal,
                  exercise_min: daily.ring.exercise_min,
                  exercise_goal_min: daily.ring.exercise_goal_min,
                  stand_hours: daily.ring.stand_hours,
                  stand_goal_hours: daily.ring.stand_goal_hours,
                }
              : null,
            steps: daily.steps,
            distance_km: daily.distance_km,
            resting_hr: daily.resting_hr,
            hrv_sdnn: daily.hrv_sdnn,
            spo2_avg: daily.spo2_avg,
            vo2max: daily.vo2max,
          }
        : undefined,
      sleep: sleep
        ? {
            asleep_s: sleep.asleep_s,
            efficiency: sleep.efficiency,
            deep_s: sleep.deep_s,
          }
        : undefined,
      activities: sectionItems<ActivityRecord>(briefing, "activities").map(
        (activity) => ({
          id: activity.id,
          sport_type: activity.sport_type,
          title: activity.title,
          started_at: activity.started_at,
          distance_m: activity.distance_m,
          moving_time_s: activity.moving_time_s,
          duration_s: activity.duration_s,
          avg_hr: activity.avg_hr,
          avg_pace_s_per_km: activity.avg_pace_s_per_km,
        }),
      ),
      media: sectionItems<MediaHomeEvent>(briefing, "media").map((item) => ({
        id: item.id,
        title: item.title,
        progress_percent: item.progress_percent,
      })),
    };
  }

  function sampleFallback(value: DesignTodayData): DesignTodayData {
    const sample = sampleToday();
    return {
      date: value.date,
      daily: value.daily ?? sample.daily,
      sleep: value.sleep ?? sample.sleep,
      activities: value.activities.length
        ? value.activities
        : sample.activities,
      media: value.media.length ? value.media : sample.media,
    };
  }

  function sampleToday(): DesignTodayData {
    return {
      date: "2026-07-18",
      daily: {
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
        vo2max: 51.4,
      },
      sleep: { asleep_s: 25620, efficiency: 0.913, deep_s: 4860 },
      activities: [
        {
          id: "activity-demo-1",
          sport_type: "run",
          title: "Morning run",
          started_at: "2026-07-18T06:42:00Z",
          distance_m: 6420,
          duration_s: 2340,
          moving_time_s: 2290,
          avg_hr: 148,
          avg_pace_s_per_km: 357,
        },
        {
          id: "activity-demo-2",
          sport_type: "walk",
          title: "Evening walk",
          started_at: "2026-07-18T18:11:00Z",
          distance_m: 2310,
          duration_s: 1860,
          moving_time_s: 1810,
          avg_hr: 102,
          avg_pace_s_per_km: 784,
        },
      ],
      media: [
        {
          id: "media-event-demo",
          title: "A quiet chapter",
          progress_percent: 76,
        },
      ],
    };
  }

  function selectTheme(language: DesignLanguage): void {
    theme.select(language);
  }

  function selectComposition(value: DesignCompositionId): void {
    variant = value;
    const url = new URL(window.location.href);
    url.searchParams.set("composition", value);
    if (url.search !== window.location.search) replaceState(url, page.state);
  }

  const selectedComposition = $derived(
    DESIGN_COMPOSITIONS.find((composition) => composition.id === variant)!,
  );
</script>

<svelte:head><title>Design workshop · iroha</title></svelte:head>

<section
  class="design-lab"
  data-theme={theme.language()}
  data-composition={variant}
>
  <header class="lab-header">
    <div>
      <p class="eyebrow">Shared design system workshop</p>
      <h1>One payload, real compositions.</h1>
      <p class="muted">
        Package-owned compositions read the same canonical Today view model.
        Registered languages provide the surrounding visual world.
      </p>
    </div>
    <span class="source-pill" class:live={source === "live"}>
      <span class="source-dot"></span>
      {source === "live" ? "Live Today payload" : "Sample Today payload"}
    </span>
  </header>

  <section class="theme-workbench" aria-labelledby="theme-workbench-title">
    <header class="theme-workbench-heading">
      <div>
        <p class="eyebrow">Theme registry</p>
        <h2 id="theme-workbench-title">{theme.definition().identity.label}</h2>
        <p class="muted">{theme.definition().identity.description}</p>
      </div>
      <div class="theme-selection" aria-label="Registered design languages">
        {#each THEME_DEFINITIONS as definition}
          <button
            type="button"
            class:active={theme.language() === definition.identity.id}
            onclick={() => selectTheme(definition.identity.id)}
            title={definition.identity.description}
          >
            <span
              class="theme-dot"
              style={"--theme-color:" + definition.identity.swatch}
            ></span>
            {definition.identity.label.replace("Iroha ", "")}
          </button>
        {/each}
      </div>
    </header>
  </section>

  <nav class="variant-tabs" aria-label="Implemented design compositions">
    {#each DESIGN_COMPOSITIONS as composition}
      <button
        type="button"
        class:active={variant === composition.id}
        aria-pressed={variant === composition.id}
        title={composition.intent}
        onclick={() => selectComposition(composition.id)}
      >
        <span>{composition.index}</span>
        {composition.label}
      </button>
    {/each}
  </nav>
  <p class="composition-intent" aria-live="polite">
    <strong>{selectedComposition.label}</strong> · {selectedComposition.intent}
  </p>

  <section class="signal-strip" aria-label="Today at a glance">
    <div class="signal-intro">
      <span class="signal-live"></span><span>Today at a glance</span>
    </div>
    <div class="signal-metric">
      <span>Move</span><strong
        >{today.daily?.ring?.move_kcal ?? "—"}<small> kcal</small></strong
      ><i
        style={"--fill:" +
          Math.min(
            100,
            ((today.daily?.ring?.move_kcal ?? 0) /
              (today.daily?.ring?.move_goal_kcal || 1)) *
              100,
          ) +
          "%"}
      ></i>
    </div>
    <div class="signal-metric">
      <span>Sleep</span><strong
        >{today.sleep ? designDuration(today.sleep.asleep_s) : "—"}</strong
      ><i
        style={"--fill:" +
          Math.min(100, (today.sleep?.efficiency ?? 0) * 100) +
          "%"}
      ></i>
    </div>
    <div class="signal-metric">
      <span>Steps</span><strong
        >{today.daily?.steps?.toLocaleString() ?? "—"}</strong
      ><i
        style={"--fill:" +
          Math.min(100, ((today.daily?.steps ?? 0) / 10000) * 100) +
          "%"}
      ></i>
    </div>
    <div class="signal-metric">
      <span>Focus</span><strong
        >{today.activities.length} <small>sessions</small></strong
      ><i style={"--fill:" + Math.min(100, today.activities.length * 34) + "%"}
      ></i>
    </div>
  </section>

  <DesignCompositionRenderer
    composition={variant}
    {today}
    {readiness}
    links={compositionLinks}
  />
</section>

<style>
  .design-lab {
    display: grid;
    gap: 1.25rem;
    padding-bottom: 3rem;
  }
  .lab-header,
  .theme-workbench-heading {
    display: flex;
    justify-content: space-between;
    gap: 1.5rem;
    align-items: flex-start;
  }
  .lab-header h1,
  .theme-workbench-heading h2 {
    margin: 0;
    letter-spacing: -0.06em;
  }
  .lab-header h1 {
    font-size: clamp(2rem, 5vw, 3.4rem);
  }
  .eyebrow {
    margin: 0 0 0.4rem;
    color: var(--accent);
    font-size: 0.7rem;
    font-weight: 750;
    letter-spacing: 0.11em;
    text-transform: uppercase;
  }
  .muted {
    color: var(--text-muted);
    line-height: 1.55;
  }
  .lab-header .muted {
    max-width: 48rem;
    margin: 0.55rem 0 0;
  }
  .source-pill {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
    flex: 0 0 auto;
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 0.45rem 0.7rem;
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .source-dot,
  .signal-live,
  .theme-dot {
    display: block;
    flex: 0 0 auto;
    border-radius: 50%;
  }
  .source-dot {
    width: 0.45rem;
    height: 0.45rem;
    background: var(--accent-2);
  }
  .source-pill.live .source-dot {
    background: var(--accent);
    box-shadow: 0 0 10px var(--accent);
  }
  .theme-workbench {
    display: grid;
    gap: 1rem;
    border: 1px solid var(--border);
    border-radius: 1rem;
    padding: 1rem;
    background: color-mix(in srgb, var(--surface) 94%, var(--accent));
  }
  .theme-workbench-heading .muted {
    max-width: 40rem;
    margin: 0.4rem 0 0;
  }
  .theme-selection {
    display: grid;
    grid-template-columns: repeat(3, minmax(7rem, 1fr));
    gap: 0.25rem;
    min-width: min(32rem, 100%);
    border: 1px solid var(--border);
    border-radius: 0.7rem;
    padding: 0.25rem;
    background: var(--surface-strong);
  }
  .theme-selection button,
  .variant-tabs button {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    border: 0;
    border-radius: 0.45rem;
    padding: 0.5rem 0.55rem;
    background: transparent;
    color: var(--text-muted);
    font: inherit;
    font-size: 0.72rem;
    cursor: pointer;
  }
  .theme-selection button.active,
  .variant-tabs button.active {
    background: var(--surface-2);
    color: var(--text);
    box-shadow: 0 1px 5px color-mix(in srgb, var(--text) 10%, transparent);
  }
  .theme-dot {
    width: 0.55rem;
    height: 0.55rem;
    background: var(--theme-color, var(--accent));
  }
  .variant-tabs {
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem;
    width: fit-content;
    border: 1px solid var(--border);
    border-radius: 0.75rem;
    padding: 0.25rem;
    background: var(--surface);
  }
  .variant-tabs button span {
    display: grid;
    place-items: center;
    width: 1.2rem;
    height: 1.2rem;
    border: 1px solid var(--border);
    border-radius: 50%;
    font-size: 0.64rem;
  }
  .variant-tabs button.active span {
    border-color: var(--accent);
    color: var(--accent);
  }
  .composition-intent {
    margin: -0.7rem 0 0;
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .composition-intent strong {
    color: var(--text);
  }
  .signal-strip {
    display: grid;
    grid-template-columns: 1.2fr repeat(4, 1fr);
    gap: 0.7rem;
    padding: 0.65rem 0.8rem;
    border: 1px solid color-mix(in srgb, var(--accent) 24%, var(--border));
    border-radius: 0.8rem;
    background: var(--tile-surface);
    box-shadow: var(--tile-shadow);
  }
  .signal-intro,
  .signal-metric {
    display: flex;
    align-items: center;
    gap: 0.3rem;
    min-width: 0;
  }
  .signal-intro {
    gap: 0.45rem;
    color: var(--text-muted);
    font-size: 0.7rem;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  .signal-live {
    width: 0.45rem;
    height: 0.45rem;
    background: var(--accent);
    box-shadow: 0 0 0 4px color-mix(in srgb, var(--accent) 12%, transparent);
  }
  .signal-metric {
    display: grid;
    border-left: 1px solid var(--border);
    padding-left: 0.7rem;
  }
  .signal-metric span {
    color: var(--text-muted);
    font-size: 0.66rem;
  }
  .signal-metric strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 0.9rem;
  }
  .signal-metric strong small {
    color: var(--text-muted);
    font-size: 0.68rem;
    font-weight: 500;
  }
  .signal-metric i {
    display: block;
    height: 0.18rem;
    overflow: hidden;
    border-radius: 99px;
    background: linear-gradient(
      90deg,
      var(--accent) var(--fill),
      var(--border) var(--fill)
    );
  }
  @media (max-width: 1024px) {
    .lab-header,
    .theme-workbench-heading {
      flex-direction: column;
    }
    .theme-selection {
      width: 100%;
    }
    .signal-strip {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .signal-intro {
      grid-column: 1 / -1;
    }
  }
  @media (max-width: 640px) {
    .theme-selection,
    .signal-strip {
      grid-template-columns: 1fr;
    }
    .signal-intro {
      grid-column: auto;
    }
    .signal-metric {
      border-left: 0;
      border-top: 1px solid var(--border);
      padding: 0.55rem 0 0;
    }
  }
</style>
