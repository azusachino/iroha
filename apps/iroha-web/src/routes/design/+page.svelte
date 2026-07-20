<script lang="ts">
  import { onMount } from "svelte";
  import {
    Activity,
    ArrowRight,
    BookOpen,
    Moon,
    Sparkles,
  } from "@lucide/svelte";
  import {
    getBriefing,
    type Activity as ActivityRecord,
    type DailyRow,
    type MediaHomeEvent,
    type SleepSession,
  } from "$lib/api";
  import RingGauge, { type Ring } from "$lib/components/RingGauge.svelte";
  import SportBadge from "$lib/components/SportBadge.svelte";
  import {
    formatDistance,
    formatDuration,
    formatHr,
    formatPace,
    formatSport,
  } from "$lib/format";

  type Variant =
    | "editorial"
    | "command"
    | "chronicle"
    | "cover"
    | "workspace"
    | "journal"
    | "quiet";
  type TodayData = {
    date: string;
    daily?: DailyRow;
    sleep?: SleepSession;
    activities: ActivityRecord[];
    media: MediaHomeEvent[];
  };

  let variant = $state<Variant>("editorial");
  let today = $state<TodayData>(sampleToday());
  let source = $state<"sample" | "live">("sample");

  const rings = $derived<Ring[]>(
    today.daily
      ? [
          {
            label: "Move",
            value: today.daily.move_kcal,
            goal: today.daily.move_goal_kcal,
            unit: "kcal",
            color: "var(--ring-move)",
          },
          {
            label: "Exercise",
            value: today.daily.exercise_min,
            goal: today.daily.exercise_goal_min,
            unit: "min",
            color: "var(--ring-exercise)",
          },
          {
            label: "Stand",
            value: today.daily.stand_hours,
            goal: today.daily.stand_goal_hours,
            unit: "h",
            color: "var(--ring-stand)",
          },
        ]
      : [],
  );

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

  onMount(() => {
    void loadToday();
  });

  async function loadToday() {
    const date = new Date().toISOString().slice(0, 10);
    try {
      const briefing = await getBriefing(date);
      today = fromBriefing(briefing);
      source = "live";
    } catch {
      // The playground remains useful without a running server.
      source = "sample";
    }
  }

  function fromBriefing(
    briefing: Awaited<ReturnType<typeof getBriefing>>,
  ): TodayData {
    function items<T>(key: string): T[] {
      const section = briefing.sections.find((item) => item.key === key);
      return (section?.data as { items?: T[] } | undefined)?.items ?? [];
    }

    return {
      date: briefing.date,
      daily: items<DailyRow>("daily")[0],
      sleep:
        items<SleepSession>("sleep").find((item) => item.is_main_sleep) ??
        items<SleepSession>("sleep")[0],
      activities: items<ActivityRecord>("activities"),
      media: items<MediaHomeEvent>("media"),
    };
  }

  function sampleToday(): TodayData {
    return {
      date: new Date().toISOString().slice(0, 10),
      daily: {
        id: "daily-demo",
        day: "2026-07-18",
        move_kcal: 486,
        move_goal_kcal: 650,
        exercise_min: 34,
        exercise_goal_min: 30,
        stand_hours: 9,
        stand_goal_hours: 12,
        steps: 8420,
        distance_km: 6.8,
        flights: 7,
        resting_hr: 51,
        hrv_sdnn: 58,
        spo2_avg: 98,
        respiratory_rate: 14.2,
        vo2max: 51.4,
        body_mass_kg: 64.8,
        source: "Apple Health",
        first_raw_file_id: "raw-demo",
        created_at: "2026-07-18T08:00:00Z",
        updated_at: "2026-07-18T08:00:00Z",
      },
      sleep: {
        id: "sleep-demo",
        wake_date: "2026-07-18",
        started_at: "2026-07-17T23:16:00Z",
        ended_at: "2026-07-18T07:04:00Z",
        time_in_bed_s: 28080,
        asleep_s: 25620,
        efficiency: 0.913,
        is_main_sleep: true,
        core_s: 12420,
        deep_s: 4860,
        rem_s: 8340,
        awake_s: 2460,
        unspecified_s: 0,
        source: "Apple Health",
        first_raw_file_id: "raw-demo",
        created_at: "2026-07-18T08:00:00Z",
        updated_at: "2026-07-18T08:00:00Z",
      },
      activities: [
        {
          id: "activity-demo-1",
          sport_type: "run",
          title: "Morning run",
          started_at: "2026-07-18T06:42:00Z",
          ended_at: "2026-07-18T07:21:00Z",
          timezone: "Asia/Tokyo",
          distance_m: 6420,
          duration_s: 2340,
          moving_time_s: 2290,
          elevation_gain_m: 74,
          avg_hr: 148,
          max_hr: 171,
          avg_pace_s_per_km: 357,
          source_kind: "apple_health_export",
          first_raw_file_id: "raw-demo",
          created_at: "2026-07-18T08:00:00Z",
          updated_at: "2026-07-18T08:00:00Z",
        },
        {
          id: "activity-demo-2",
          sport_type: "walk",
          title: "Evening walk",
          started_at: "2026-07-18T18:11:00Z",
          ended_at: "2026-07-18T18:42:00Z",
          timezone: "Asia/Tokyo",
          distance_m: 2310,
          duration_s: 1860,
          moving_time_s: 1810,
          elevation_gain_m: 18,
          avg_hr: 102,
          max_hr: 119,
          avg_pace_s_per_km: 784,
          source_kind: "apple_health_export",
          first_raw_file_id: "raw-demo",
          created_at: "2026-07-18T08:00:00Z",
          updated_at: "2026-07-18T08:00:00Z",
        },
      ],
      media: [
        {
          id: "media-event-demo",
          media_id: "media-demo",
          title: "A quiet chapter",
          event_type: "progress",
          occurred_at: "2026-07-18T21:05:00Z",
          position: 182,
          total: 240,
          progress_percent: 76,
        },
      ],
    };
  }

  function activityMetric(activity: ActivityRecord): string {
    return `${formatDistance(activity.distance_m)} · ${formatDuration(activity.moving_time_s ?? activity.duration_s)} · ${formatPace(activity.avg_pace_s_per_km)}`;
  }

  function dateLabel(date: string): string {
    return new Date(`${date}T00:00:00Z`).toLocaleDateString(undefined, {
      weekday: "long",
      month: "short",
      day: "numeric",
      timeZone: "UTC",
    });
  }
</script>

<svelte:head>
  <title>Today design lab · iroha</title>
</svelte:head>

<section class="design-lab">
  <header class="lab-header">
    <div>
      <p class="eyebrow">Local design lab</p>
      <h1>Today, three ways</h1>
      <p class="muted">
        Same Today payload. Different hierarchy, rhythm, and density.
      </p>
    </div>
    <span class="source-pill" class:live={source === "live"}>
      <span class="source-dot"></span>
      {source === "live" ? "Live Today payload" : "Sample Today payload"}
    </span>
  </header>

  <nav class="variant-tabs" aria-label="Design variants">
    <button
      class:active={variant === "editorial"}
      onclick={() => (variant = "editorial")}
    >
      <span>A</span> Editorial
    </button>
    <button
      class:active={variant === "command"}
      onclick={() => (variant = "command")}
    >
      <span>B</span> Command center
    </button>
    <button
      class:active={variant === "chronicle"}
      onclick={() => (variant = "chronicle")}
    >
      <span>C</span> Chronicle
    </button>
    <button
      class:active={variant === "cover"}
      onclick={() => (variant = "cover")}
    >
      <span>D</span> Cover page
    </button>
    <button
      class:active={variant === "workspace"}
      onclick={() => (variant = "workspace")}
    >
      <span>E</span> Personal OS
    </button>
    <button
      class:active={variant === "journal"}
      onclick={() => (variant = "journal")}
    >
      <span>F</span> Field journal
    </button>
    <button
      class:active={variant === "quiet"}
      onclick={() => (variant = "quiet")}
    >
      <span>G</span> Quiet
    </button>
  </nav>

  <section class="signal-strip" aria-label="Today at a glance">
    <div class="signal-intro">
      <span class="signal-live"></span>
      <span>Today at a glance</span>
    </div>
    <div class="signal-metric">
      <span>Move</span>
      <strong>{today.daily?.move_kcal ?? "—"}<small> kcal</small></strong>
      <i
        style={"--fill:" +
          Math.min(
            100,
            ((today.daily?.move_kcal ?? 0) /
              (today.daily?.move_goal_kcal || 1)) *
              100,
          ) +
          "%"}
      ></i>
    </div>
    <div class="signal-metric">
      <span>Sleep</span>
      <strong>{today.sleep ? formatDuration(today.sleep.asleep_s) : "—"}</strong
      >
      <i
        style={"--fill:" +
          Math.min(100, (today.sleep?.efficiency ?? 0) * 100) +
          "%"}
      ></i>
    </div>
    <div class="signal-metric">
      <span>Steps</span>
      <strong>{today.daily?.steps?.toLocaleString() ?? "—"}</strong>
      <i
        style={"--fill:" +
          Math.min(100, ((today.daily?.steps ?? 0) / 10000) * 100) +
          "%"}
      ></i>
    </div>
    <div class="signal-metric signal-focus">
      <span>Focus</span>
      <strong>{today.activities.length} <small>sessions</small></strong>
      <i style={"--fill:" + Math.min(100, today.activities.length * 34) + "%"}
      ></i>
    </div>
  </section>

  {#if variant === "editorial"}
    <section class="demo editorial" aria-label="Editorial Today design">
      <header class="demo-heading">
        <div>
          <p class="eyebrow">{dateLabel(today.date)}</p>
          <h2>A good day has a shape.</h2>
          <p class="muted">
            A compact read of movement, recovery, and the things you touched
            today.
          </p>
        </div>
        <button class="date-button">Today <ArrowRight size={15} /></button>
      </header>

      <div class="editorial-hero tile">
        <div class="hero-copy">
          <span class="hero-kicker"><Sparkles size={15} /> Daily signal</span>
          <strong>{readiness}<small>/100</small></strong>
          <h3>Steady, with room to move.</h3>
          <p class="muted">
            Sleep was strong and you already cleared your exercise target. A
            short walk would close the day gently.
          </p>
        </div>
        <div class="hero-rings"><RingGauge {rings} size={168} /></div>
      </div>

      <div class="editorial-grid">
        <a class="feature tile" href="/sleep">
          <span class="card-kicker"><Moon size={15} /> Recovery</span>
          <strong
            >{today.sleep ? formatDuration(today.sleep.asleep_s) : "—"}</strong
          >
          <p class="muted">
            {today.sleep
              ? `${Math.round(today.sleep.efficiency * 100)}% sleep efficiency`
              : "No sleep recorded"}
          </p>
        </a>
        <a class="feature tile" href="/activities">
          <span class="card-kicker"><Activity size={15} /> Training</span>
          <strong>{today.activities.length} <small>sessions</small></strong>
          <p class="muted">
            {today.activities[0]
              ? activityMetric(today.activities[0])
              : "No activity recorded"}
          </p>
        </a>
        <section class="feature tile">
          <span class="card-kicker"><BookOpen size={15} /> In progress</span>
          <strong
            >{today.media[0]?.progress_percent ?? 0}<small>%</small></strong
          >
          <p class="muted">{today.media[0]?.title ?? "Nothing logged today"}</p>
        </section>
      </div>
    </section>
  {:else if variant === "command"}
    <section
      class="demo command-center"
      aria-label="Command center Today design"
    >
      <aside class="command-rail tile">
        <div class="rail-date">
          <span>Today</span><strong>{dateLabel(today.date)}</strong>
        </div>
        <div class="rail-score">
          <span>Readiness</span><strong>{readiness}</strong><small
            >balanced</small
          >
        </div>
        <div class="rail-facts">
          <div>
            <span>Steps</span><strong
              >{today.daily?.steps?.toLocaleString() ?? "—"}</strong
            >
          </div>
          <div>
            <span>Resting HR</span><strong
              >{formatHr(today.daily?.resting_hr)}</strong
            >
          </div>
          <div>
            <span>Distance</span><strong
              >{formatDistance((today.daily?.distance_km ?? 0) * 1000)}</strong
            >
          </div>
        </div>
        <p class="rail-hint">⌘K <span>jump anywhere</span></p>
      </aside>

      <div class="command-main">
        <header class="demo-heading compact">
          <div>
            <p class="eyebrow">Overview</p>
            <h2>Keep the signal visible.</h2>
          </div>
          <button class="date-button"
            >{today.date} <ArrowRight size={15} /></button
          >
        </header>
        <div class="command-kpis">
          <article class="command-kpi tile">
            <span>Readiness</span>
            <strong>{readiness}<small>/100</small></strong>
            <i style={`--fill:${readiness}%`}></i>
          </article>
          <article class="command-kpi tile">
            <span>Move target</span>
            <strong
              >{today.daily?.move_kcal ?? "—"}<small>
                / {today.daily?.move_goal_kcal ?? "—"} kcal</small
              ></strong
            >
            <i
              style={`--fill:${Math.min(100, ((today.daily?.move_kcal ?? 0) / (today.daily?.move_goal_kcal || 1)) * 100)}%`}
            ></i>
          </article>
          <article class="command-kpi tile">
            <span>Resting heart</span>
            <strong>{formatHr(today.daily?.resting_hr)}</strong>
            <small class="kpi-note">{today.daily?.hrv_sdnn ?? "—"} ms HRV</small
            >
          </article>
          <article class="command-kpi tile">
            <span>Open threads</span>
            <strong>{today.activities.length + today.media.length}</strong>
            <small class="kpi-note">activity + media</small>
          </article>
        </div>
        <div class="command-grid">
          <section class="command-rings tile">
            <div class="section-title">
              <span>Activity rings</span><a href="/daily"
                >Open daily <ArrowRight size={14} /></a
              >
            </div>
            <RingGauge {rings} size={150} />
          </section>
          <section class="command-sleep tile">
            <div class="section-title">
              <span>Sleep</span><a href="/sleep"
                >Details <ArrowRight size={14} /></a
              >
            </div>
            <strong
              >{today.sleep
                ? formatDuration(today.sleep.asleep_s)
                : "—"}</strong
            >
            <p class="muted">
              {today.sleep
                ? `${Math.round(today.sleep.efficiency * 100)}% efficiency · ${formatDuration(today.sleep.deep_s)} deep`
                : "No record"}
            </p>
            <div class="sleep-bar">
              <span
                style={`width:${Math.min(100, (today.sleep?.asleep_s ?? 0) / 288)}%`}
              ></span>
            </div>
          </section>
          <section class="command-activities tile">
            <div class="section-title">
              <span>Activity stream</span><a href="/activities"
                >View all <ArrowRight size={14} /></a
              >
            </div>
            <div class="activity-stream">
              {#each today.activities as activity (activity.id)}<a
                  href={`/activities/${activity.id}`}
                  class="stream-row"
                  ><SportBadge sport={activity.sport_type} /><span
                    ><strong>{activity.title}</strong><small
                      >{activityMetric(activity)}</small
                    ></span
                  ><ArrowRight size={14} /></a
                >{/each}
            </div>
          </section>
          <section class="command-vitals tile">
            <div class="section-title">
              <span>Vitals</span><a href="/daily"
                >Trends <ArrowRight size={14} /></a
              >
            </div>
            <div class="vital-list">
              {#each [{ label: "HRV", value: `${today.daily?.hrv_sdnn ?? "—"} ms` }, { label: "SpO₂", value: `${today.daily?.spo2_avg ?? "—"}%` }, { label: "VO₂max", value: today.daily?.vo2max?.toFixed(1) ?? "—" }] as vital}<div
                >
                  <span>{vital.label}</span><strong>{vital.value}</strong>
                </div>{/each}
            </div>
          </section>
          <section class="command-next tile">
            <div class="section-title">
              <span>Next move</span><span class="status-label">low effort</span>
            </div>
            <strong
              >{today.activities.length
                ? "Close the loop gently."
                : "Make the first mark."}</strong
            >
            <p class="muted">
              {today.daily?.exercise_min &&
              today.daily.exercise_min >= (today.daily.exercise_goal_min ?? 0)
                ? "Exercise is covered. A short walk keeps the evening open."
                : "A small block of movement will change the shape of the day."}
            </p>
            <a href="/activities">Plan an activity <ArrowRight size={14} /></a>
          </section>
          <section class="command-media tile">
            <div class="section-title">
              <span>Recently touched</span><a href="/media"
                >Open library <ArrowRight size={14} /></a
              >
            </div>
            {#if today.media[0]}
              <strong>{today.media[0].title}</strong>
              <p class="muted">
                {today.media[0].progress_percent ?? 0}% through · last touched
                tonight
              </p>
              <div class="media-progress">
                <span style={`width:${today.media[0].progress_percent ?? 0}%`}
                ></span>
              </div>
            {:else}
              <strong>No media signal yet.</strong>
              <p class="muted">
                Your library will appear here when something changes.
              </p>
            {/if}
          </section>
        </div>
      </div>
    </section>
  {:else if variant === "chronicle"}
    <section class="demo chronicle" aria-label="Chronicle Today design">
      <header class="chronicle-heading">
        <div>
          <p class="eyebrow">{dateLabel(today.date)}</p>
          <h2>Saturday, in sequence.</h2>
          <p class="muted">
            A chronological record that makes the day feel lived in, not
            measured.
          </p>
        </div>
        <div class="chronicle-total">
          <strong
            >{today.activities.length +
              (today.sleep ? 1 : 0) +
              today.media.length}</strong
          ><span>moments</span>
        </div>
      </header>
      <div class="chronicle-body">
        <div class="timeline">
          {#if today.sleep}<article class="moment">
              <time>07:04</time>
              <div class="moment-pin sleep-pin"><Moon size={14} /></div>
              <div class="moment-card tile">
                <span class="card-kicker">Recovery</span>
                <h3>Sleep completed</h3>
                <p>
                  {formatDuration(today.sleep.asleep_s)} asleep · {Math.round(
                    today.sleep.efficiency * 100,
                  )}% efficiency
                </p>
                <div class="micro-bar">
                  <span style={`width:${today.sleep.efficiency * 100}%`}></span>
                </div>
              </div>
            </article>{/if}
          {#each today.activities as activity (activity.id)}<article
              class="moment"
            >
              <time>{activity.started_at.slice(11, 16)}</time>
              <div class="moment-pin activity-pin"><Activity size={14} /></div>
              <a class="moment-card tile" href={`/activities/${activity.id}`}
                ><span class="card-kicker"
                  >{formatSport(activity.sport_type)}</span
                >
                <h3>{activity.title}</h3>
                <p>{activityMetric(activity)} · {formatHr(activity.avg_hr)}</p>
                <span class="moment-link"
                  >Open activity <ArrowRight size={14} /></span
                ></a
              >
            </article>{/each}
          {#if today.media[0]}<article class="moment">
              <time>21:05</time>
              <div class="moment-pin media-pin"><BookOpen size={14} /></div>
              <div class="moment-card tile">
                <span class="card-kicker">Media</span>
                <h3>{today.media[0].title}</h3>
                <p>{today.media[0].progress_percent}% complete</p>
                <div class="micro-bar">
                  <span style={`width:${today.media[0].progress_percent ?? 0}%`}
                  ></span>
                </div>
              </div>
            </article>{/if}
        </div>
        <aside class="chronicle-aside tile">
          <span class="card-kicker">Day summary</span><strong
            >{readiness}<small>/100</small></strong
          >
          <p class="muted">
            Your day has enough movement, enough rest, and one more small story.
          </p>
          <dl>
            <div>
              <dt>Steps</dt>
              <dd>{today.daily?.steps?.toLocaleString() ?? "—"}</dd>
            </div>
            <div>
              <dt>Exercise</dt>
              <dd>{today.daily?.exercise_min ?? "—"} min</dd>
            </div>
            <div>
              <dt>Distance</dt>
              <dd>{formatDistance((today.daily?.distance_km ?? 0) * 1000)}</dd>
            </div>
          </dl>
        </aside>
      </div>
    </section>
  {:else if variant === "cover"}
    <section class="demo cover-page" aria-label="Cover page Today design">
      <header class="cover-masthead">
        <div>
          <p class="eyebrow">Iroha / personal field notes</p>
          <h2>Saturday<br /><em>in motion.</em></h2>
        </div>
        <div class="cover-index">
          <span>{dateLabel(today.date)}</span>
          <strong>07—18</strong>
          <span>Tokyo · private edition</span>
        </div>
      </header>
      <div class="cover-grid">
        <article class="cover-art tile">
          <div class="cover-orbit orbit-one"></div>
          <div class="cover-orbit orbit-two"></div>
          <div class="cover-sun">I</div>
          <span class="cover-art-label">Daily composition / 01</span>
          <strong>{readiness}<small> readiness</small></strong>
          <p>
            Sleep gave the day a soft start. Movement is already in the frame;
            leave some space for whatever comes next.
          </p>
        </article>
        <div class="cover-notes">
          <article class="cover-note tile">
            <span class="card-kicker">The body</span>
            <strong>{today.daily?.steps?.toLocaleString() ?? "—"}</strong>
            <p>
              steps across {formatDistance(
                (today.daily?.distance_km ?? 0) * 1000,
              )}
            </p>
          </article>
          <article class="cover-note tile">
            <span class="card-kicker">The night</span>
            <strong
              >{today.sleep
                ? formatDuration(today.sleep.asleep_s)
                : "—"}</strong
            >
            <p>
              {today.sleep
                ? Math.round(today.sleep.efficiency * 100) + "% efficient"
                : "No sleep record"}
            </p>
          </article>
          <article class="cover-note cover-note-wide tile">
            <span class="card-kicker">The next small thing</span>
            <strong
              >{today.activities.length
                ? "Take the long way home."
                : "Make the first mark."}</strong
            >
            <p>One gentle choice is enough to give the day a shape.</p>
          </article>
        </div>
      </div>
    </section>
  {:else if variant === "workspace"}
    <section class="demo workspace" aria-label="Personal OS Today design">
      <header class="workspace-header">
        <div>
          <p class="eyebrow">Personal OS / daily workspace</p>
          <h2>Good morning, Haru.</h2>
          <p class="muted">
            Everything you need for today, arranged as a calm working surface.
          </p>
        </div>
        <button class="workspace-add" type="button"
          ><Sparkles size={15} /> Add block</button
        >
      </header>
      <div class="workspace-layout">
        <aside class="workspace-sidebar tile">
          <span class="workspace-logo">i</span>
          <nav aria-label="Workspace sections">
            <a class="selected" href="/"><span>◈</span> Today</a>
            <a href="/daily"><span>◌</span> Patterns</a>
            <a href="/activities"><span>↗</span> Activities</a>
            <a href="/media"><span>▧</span> Library</a>
          </nav>
          <div class="workspace-sidebar-foot">
            <span>Workspace health</span>
            <strong><i></i> Synced just now</strong>
          </div>
        </aside>
        <div class="workspace-canvas">
          <div class="workspace-toolbar">
            <span>Today / {today.date}</span>
            <div>
              <button class="view-active" type="button">Board</button><button
                type="button">List</button
              ><button type="button">Timeline</button>
            </div>
          </div>
          <div class="workspace-columns">
            <section class="workspace-column">
              <header>
                <span class="column-dot teal"></span><strong>Now</strong><small
                  >2 blocks</small
                >
              </header>
              <article class="workspace-card hero-block tile">
                <span class="card-kicker">Readiness</span>
                <strong>{readiness}<small>/100</small></strong>
                <p>Steady, with room to move.</p>
                <div class="workspace-progress">
                  <i style={"--fill:" + readiness + "%"}></i>
                </div>
              </article>
              <article class="workspace-card tile">
                <span class="card-kicker">Focus note</span>
                <p>Keep the next action small enough to begin.</p>
                <span class="workspace-meta">Added today · personal</span>
              </article>
            </section>
            <section class="workspace-column">
              <header>
                <span class="column-dot pink"></span><strong>Collected</strong
                ><small>{today.activities.length + 1} items</small>
              </header>
              {#each today.activities as activity (activity.id)}
                <a
                  class="workspace-card tile"
                  href={"/activities/" + activity.id}
                >
                  <span class="workspace-card-top"
                    ><SportBadge sport={activity.sport_type} /><small
                      >{activity.started_at.slice(11, 16)}</small
                    ></span
                  >
                  <strong>{activity.title}</strong>
                  <p>{activityMetric(activity)}</p>
                </a>
              {/each}
              <article class="workspace-card tile">
                <span class="card-kicker">Sleep</span>
                <strong
                  >{today.sleep
                    ? formatDuration(today.sleep.asleep_s)
                    : "—"}</strong
                >
                <p>Recovery block</p>
              </article>
            </section>
            <section class="workspace-column">
              <header>
                <span class="column-dot amber"></span><strong>Later</strong
                ><small>one idea</small>
              </header>
              <article class="workspace-card idea-card tile">
                <span class="card-kicker">Open loop</span>
                <strong>Make space for a little wonder.</strong>
                <p>Media progress: {today.media[0]?.progress_percent ?? 0}%</p>
              </article>
            </section>
          </div>
        </div>
      </div>
    </section>
  {:else if variant === "journal"}
    <section class="demo field-journal" aria-label="Field journal Today design">
      <header class="journal-header">
        <div class="journal-number">No. 018</div>
        <div>
          <p class="eyebrow">Field journal / {dateLabel(today.date)}</p>
          <h2>Notes from a body in motion.</h2>
        </div>
        <span class="journal-weather"
          >private · {today.activities.length} traces</span
        >
      </header>
      <div class="journal-layout">
        <div class="journal-paper">
          <div class="journal-rule"></div>
          <article class="journal-entry">
            <time>07:04</time>
            <div>
              <span class="journal-tag">Recovery</span>
              <h3>The night held.</h3>
              <p>
                {today.sleep
                  ? formatDuration(today.sleep.asleep_s) +
                    " asleep, " +
                    Math.round(today.sleep.efficiency * 100) +
                    "% efficient."
                  : "No sleep record for this morning."} The day began with enough
                reserve.
              </p>
            </div>
          </article>
          {#each today.activities as activity (activity.id)}
            <article class="journal-entry">
              <time>{activity.started_at.slice(11, 16)}</time>
              <div>
                <span class="journal-tag"
                  >{formatSport(activity.sport_type)}</span
                >
                <h3>{activity.title}.</h3>
                <p>{activityMetric(activity)}. A trace worth keeping.</p>
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
        <aside class="journal-index tile">
          <span class="card-kicker">Today's index</span>
          <strong>{readiness}<small>/100</small></strong>
          <div class="journal-stats">
            <div>
              <span>movement</span><b>{today.daily?.exercise_min ?? "—"} min</b>
            </div>
            <div>
              <span>distance</span><b
                >{formatDistance((today.daily?.distance_km ?? 0) * 1000)}</b
              >
            </div>
            <div>
              <span>heartbeat</span><b>{formatHr(today.daily?.resting_hr)}</b>
            </div>
          </div>
          <p>Collected from Apple Health and your own small observations.</p>
        </aside>
      </div>
    </section>
  {:else}
    <section
      class="demo quiet-dashboard"
      aria-label="Quiet dashboard Today design"
    >
      <header class="quiet-header">
        <div>
          <p class="eyebrow">Saturday · {dateLabel(today.date)}</p>
          <h2>A quieter way to notice.</h2>
        </div>
        <span class="quiet-mark">iroha / 01</span>
      </header>
      <div class="quiet-hero">
        <div>
          <span class="card-kicker">Your pace today</span>
          <strong>{readiness}</strong>
          <p>There is energy here, but nothing needs to be forced.</p>
        </div>
        <div class="quiet-bloom" aria-hidden="true">
          <span></span><span></span><span></span>
        </div>
      </div>
      <div class="quiet-grid">
        <article class="quiet-stat">
          <span>sleep</span>
          <strong
            >{today.sleep ? formatDuration(today.sleep.asleep_s) : "—"}</strong
          >
          <p>
            {today.sleep
              ? Math.round(today.sleep.efficiency * 100) + "% efficiency"
              : "awaiting a record"}
          </p>
        </article>
        <article class="quiet-stat">
          <span>movement</span>
          <strong>{today.daily?.exercise_min ?? "—"} <small>min</small></strong>
          <p>
            {today.daily?.move_kcal ?? "—"} kcal · {today.daily?.steps?.toLocaleString() ??
              "—"} steps
          </p>
        </article>
        <article class="quiet-stat">
          <span>body signal</span>
          <strong>{formatHr(today.daily?.resting_hr)}</strong>
          <p>resting heart rate · {today.daily?.hrv_sdnn ?? "—"} ms HRV</p>
        </article>
      </div>
      <footer class="quiet-footer">
        <span>Nothing to optimize right now.</span>
        <a href="/daily">See the longer pattern <ArrowRight size={14} /></a>
      </footer>
    </section>
  {/if}
</section>

<style>
  .design-lab {
    position: relative;
    display: grid;
    gap: 1.25rem;
    padding-bottom: 3rem;
  }
  .design-lab::before,
  .design-lab::after {
    position: fixed;
    z-index: -1;
    width: 28rem;
    height: 28rem;
    border-radius: 50%;
    content: "";
    filter: blur(80px);
    opacity: 0.14;
    pointer-events: none;
  }
  .design-lab::before {
    top: 4rem;
    right: -16rem;
    background: var(--accent);
  }
  .design-lab::after {
    bottom: -12rem;
    left: -18rem;
    background: var(--accent-2);
  }
  .lab-header,
  .demo-heading,
  .chronicle-heading {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1.5rem;
  }
  .lab-header h1,
  .demo-heading h2,
  .chronicle-heading h2 {
    margin: 0;
    letter-spacing: -0.035em;
  }
  .lab-header h1 {
    font-size: clamp(2rem, 5vw, 3.25rem);
  }
  .demo-heading h2,
  .chronicle-heading h2 {
    font-size: clamp(1.7rem, 3vw, 2.45rem);
  }
  .demo-heading .muted,
  .chronicle-heading .muted {
    max-width: 42rem;
    margin: 0.45rem 0 0;
  }
  .eyebrow {
    margin: 0 0 0.4rem;
    color: var(--accent);
    font-size: 0.72rem;
    font-weight: 750;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .source-pill {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
    flex: 0 0 auto;
    padding: 0.45rem 0.7rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    color: var(--text-muted);
    font-size: 0.76rem;
  }
  .source-dot {
    width: 0.45rem;
    height: 0.45rem;
    border-radius: 50%;
    background: var(--accent-2);
  }
  .source-pill.live .source-dot {
    background: var(--accent);
    box-shadow: 0 0 10px var(--accent);
  }
  .variant-tabs {
    display: flex;
    gap: 0.35rem;
    padding: 0.25rem;
    border: 1px solid var(--border);
    border-radius: 0.8rem;
    background: var(--surface);
    width: fit-content;
  }
  .variant-tabs button {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    border: 0;
    border-radius: 0.55rem;
    padding: 0.55rem 0.8rem;
    background: transparent;
    color: var(--text-muted);
    font: inherit;
    font-size: 0.82rem;
    cursor: pointer;
  }
  .variant-tabs button span {
    display: grid;
    place-items: center;
    width: 1.25rem;
    height: 1.25rem;
    border: 1px solid var(--border);
    border-radius: 50%;
    font-size: 0.68rem;
  }
  .variant-tabs button.active {
    background: var(--surface-2);
    color: var(--text);
    box-shadow: 0 1px 0 rgb(255 255 255 / 0.05) inset;
  }
  .variant-tabs button.active span {
    border-color: var(--accent);
    color: var(--accent);
  }
  .signal-strip {
    display: grid;
    grid-template-columns: 1.25fr repeat(4, 1fr);
    align-items: center;
    gap: 0.75rem;
    padding: 0.65rem 0.8rem;
    border: 1px solid color-mix(in srgb, var(--accent) 20%, var(--border));
    border-radius: 0.9rem;
    background:
      linear-gradient(
        90deg,
        color-mix(in srgb, var(--accent) 7%, transparent),
        transparent 52%
      ),
      var(--tile-surface);
    box-shadow: var(--tile-shadow);
  }
  .signal-intro,
  .signal-metric {
    display: grid;
    gap: 0.2rem;
    min-width: 0;
  }
  .signal-intro {
    display: flex;
    align-items: center;
    gap: 0.45rem;
    color: var(--text-muted);
    font-size: 0.72rem;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  .signal-live {
    width: 0.45rem;
    height: 0.45rem;
    border-radius: 50%;
    background: var(--accent);
    box-shadow:
      0 0 0 4px color-mix(in srgb, var(--accent) 12%, transparent),
      0 0 14px var(--accent);
  }
  .signal-metric {
    padding-left: 0.8rem;
    border-left: 1px solid var(--border);
  }
  .signal-metric span {
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  .signal-metric strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 0.92rem;
    letter-spacing: -0.03em;
  }
  .signal-metric strong small {
    color: var(--text-muted);
    font-size: 0.68rem;
    font-weight: 500;
  }
  .signal-metric i {
    display: block;
    width: 100%;
    height: 0.18rem;
    overflow: hidden;
    border-radius: 99px;
    background: linear-gradient(
      90deg,
      var(--accent) var(--fill),
      var(--surface-2) var(--fill)
    );
  }
  .demo {
    display: grid;
    gap: 1.25rem;
  }
  .tile {
    border: 1px solid var(--border);
    background: var(--tile-surface);
    box-shadow: var(--tile-shadow);
    border-radius: 1rem;
  }
  .tile:hover {
    border-color: color-mix(in srgb, var(--accent) 40%, var(--border));
  }
  .date-button {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    flex: 0 0 auto;
    padding: 0.6rem 0.8rem;
    border: 1px solid var(--border);
    border-radius: 0.65rem;
    background: var(--surface);
    color: var(--text);
    font: inherit;
    font-size: 0.8rem;
    cursor: pointer;
  }
  .date-button:hover {
    border-color: var(--accent);
  }
  .editorial-hero {
    display: grid;
    grid-template-columns: 1.3fr 1fr;
    align-items: center;
    min-height: 20rem;
    padding: clamp(1.5rem, 5vw, 3.5rem);
    overflow: hidden;
    background:
      radial-gradient(
        circle at 88% 42%,
        color-mix(in srgb, var(--accent) 17%, transparent),
        transparent 28%
      ),
      linear-gradient(
        115deg,
        color-mix(in srgb, var(--accent) 7%, transparent),
        transparent 58%
      ),
      var(--tile-surface);
    border-color: color-mix(in srgb, var(--accent) 28%, var(--border));
    box-shadow: var(--accent-glow), var(--tile-shadow);
  }
  .hero-copy {
    max-width: 34rem;
  }
  .hero-kicker,
  .card-kicker {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    color: var(--accent);
    font-size: 0.72rem;
    font-weight: 750;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .hero-copy > strong {
    display: block;
    margin: 1.5rem 0 0.4rem;
    color: var(--text);
    font-size: clamp(4rem, 12vw, 8rem);
    letter-spacing: -0.09em;
    line-height: 0.8;
  }
  .hero-copy > strong small,
  .feature strong small,
  .rail-score small,
  .chronicle-aside strong small {
    margin-left: 0.35rem;
    color: var(--text-muted);
    font-size: 0.25em;
    font-weight: 500;
    letter-spacing: -0.02em;
  }
  .hero-copy h3 {
    margin: 0;
    font-size: clamp(1.25rem, 2.5vw, 2rem);
    letter-spacing: -0.04em;
  }
  .hero-copy p {
    max-width: 30rem;
    margin: 0.65rem 0 0;
    line-height: 1.6;
  }
  .hero-rings {
    justify-self: end;
    padding-right: 1rem;
    filter: drop-shadow(
      0 0 22px color-mix(in srgb, var(--accent) 25%, transparent)
    );
  }
  .editorial-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1rem;
  }
  .feature {
    display: grid;
    align-content: start;
    gap: 0.85rem;
    min-height: 11rem;
    padding: 1.25rem;
    color: var(--text);
    text-decoration: none;
  }
  .feature:hover {
    border-color: var(--accent);
    text-decoration: none;
    transform: translateY(-2px);
  }
  .feature {
    transition:
      border-color 180ms ease,
      transform 180ms ease,
      box-shadow 180ms ease;
  }
  .feature strong {
    font-size: clamp(2rem, 5vw, 3.6rem);
    letter-spacing: -0.07em;
    line-height: 0.95;
  }
  .feature strong small {
    font-size: 0.32em;
  }
  .feature p {
    margin: 0;
    font-size: 0.85rem;
  }
  .command-center {
    grid-template-columns: 13rem minmax(0, 1fr);
    align-items: start;
    gap: 1.25rem;
  }
  .command-rail {
    position: sticky;
    top: 5rem;
    display: grid;
    gap: 1.5rem;
    padding: 1rem;
  }
  .rail-date {
    display: grid;
    gap: 0.25rem;
  }
  .rail-date span,
  .rail-facts span,
  .rail-hint {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .rail-date strong {
    font-size: 0.9rem;
  }
  .rail-score {
    display: grid;
    gap: 0.2rem;
    padding: 1rem 0;
    border-top: 1px solid var(--border);
    border-bottom: 1px solid var(--border);
  }
  .rail-score span,
  .rail-score small {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .rail-score strong {
    color: var(--accent);
    font-size: 3.5rem;
    letter-spacing: -0.08em;
    line-height: 0.9;
  }
  .rail-facts {
    display: grid;
    gap: 0.85rem;
  }
  .rail-facts div {
    display: flex;
    justify-content: space-between;
    gap: 0.5rem;
  }
  .rail-facts strong {
    font-size: 0.8rem;
  }
  .rail-hint {
    margin: 0;
  }
  .rail-hint span {
    margin-left: 0.35rem;
  }
  .command-main {
    display: grid;
    gap: 1rem;
    min-width: 0;
  }
  .command-kpis {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 0.7rem;
  }
  .command-kpi {
    display: grid;
    gap: 0.35rem;
    min-width: 0;
    padding: 0.85rem;
  }
  .command-kpi > span {
    overflow: hidden;
    color: var(--text-muted);
    font-size: 0.66rem;
    text-overflow: ellipsis;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    white-space: nowrap;
  }
  .command-kpi strong {
    overflow: hidden;
    font-size: 1.45rem;
    letter-spacing: -0.06em;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .command-kpi strong small {
    color: var(--text-muted);
    font-size: 0.42em;
    font-weight: 500;
    letter-spacing: 0;
  }
  .command-kpi > i,
  .media-progress {
    display: block;
    height: 0.2rem;
    overflow: hidden;
    border-radius: 99px;
    background: linear-gradient(
      90deg,
      var(--accent) var(--fill, 35%),
      var(--surface-2) var(--fill, 35%)
    );
  }
  .command-kpi:nth-child(2) > i {
    background: linear-gradient(
      90deg,
      var(--ring-exercise) var(--fill, 35%),
      var(--surface-2) var(--fill, 35%)
    );
  }
  .kpi-note {
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  .demo-heading.compact {
    align-items: center;
  }
  .command-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
  }
  .command-grid > section {
    min-width: 0;
    padding: 1.1rem;
  }
  .section-title {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 1rem;
    margin-bottom: 1rem;
    color: var(--text-muted);
    font-size: 0.78rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.07em;
  }
  .section-title a {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    color: var(--accent);
    font-size: 0.72rem;
    font-weight: 500;
    text-transform: none;
    letter-spacing: 0;
  }
  .command-sleep > strong {
    display: block;
    color: var(--text);
    font-size: 3.3rem;
    letter-spacing: -0.08em;
    line-height: 1;
  }
  .command-sleep p {
    margin: 0.5rem 0 1.2rem;
  }
  .sleep-bar,
  .micro-bar {
    height: 0.35rem;
    overflow: hidden;
    border-radius: 99px;
    background: var(--surface-2);
  }
  .sleep-bar span,
  .micro-bar span {
    display: block;
    height: 100%;
    border-radius: inherit;
    background: linear-gradient(90deg, var(--accent), var(--accent-2));
  }
  .command-activities {
    grid-column: span 2;
  }
  .activity-stream {
    display: grid;
    gap: 0.45rem;
  }
  .stream-row {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr) auto;
    align-items: center;
    gap: 0.65rem;
    padding: 0.55rem 0;
    color: var(--text);
    border-top: 1px solid var(--border);
    text-decoration: none;
  }
  .stream-row:first-child {
    border-top: 0;
  }
  .stream-row span {
    min-width: 0;
    display: grid;
    gap: 0.2rem;
  }
  .stream-row strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 0.86rem;
  }
  .stream-row small {
    color: var(--text-muted);
    font-size: 0.76rem;
  }
  .stream-row:hover {
    color: var(--accent);
  }
  .vital-list {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 0.6rem;
  }
  .vital-list div {
    display: grid;
    gap: 0.25rem;
  }
  .vital-list span {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .vital-list strong {
    font-size: 1rem;
  }
  .command-next,
  .command-media {
    display: grid;
    align-content: start;
    gap: 0.55rem;
  }
  .command-next > strong,
  .command-media > strong {
    font-size: 1.35rem;
    letter-spacing: -0.045em;
  }
  .command-next p,
  .command-media p {
    margin: 0;
    line-height: 1.5;
  }
  .command-next a,
  .command-media a {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    margin-top: 0.35rem;
    font-size: 0.75rem;
  }
  .status-label {
    color: var(--accent);
    font-size: 0.68rem;
    font-weight: 500;
    text-transform: none;
    letter-spacing: 0;
  }
  .media-progress {
    margin-top: 0.35rem;
    background: var(--surface-2);
  }
  .media-progress span {
    display: block;
    width: var(--fill, 0%);
    height: 100%;
    border-radius: inherit;
    background: var(--accent-2);
  }
  .chronicle {
    gap: 2rem;
  }
  .chronicle-total {
    display: grid;
    justify-items: end;
    align-content: start;
  }
  .chronicle-total strong {
    color: var(--accent);
    font-size: 3.5rem;
    letter-spacing: -0.09em;
    line-height: 0.85;
  }
  .chronicle-total span {
    color: var(--text-muted);
    font-size: 0.76rem;
  }
  .chronicle-body {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 15rem;
    align-items: start;
    gap: 2rem;
  }
  .timeline {
    position: relative;
    display: grid;
    gap: 1rem;
  }
  .timeline::before {
    position: absolute;
    top: 1rem;
    bottom: 1rem;
    left: 5.25rem;
    width: 1px;
    background: var(--border);
    content: "";
  }
  .moment {
    display: grid;
    grid-template-columns: 4rem 2.5rem minmax(0, 1fr);
    align-items: start;
    gap: 0.75rem;
  }
  .moment time {
    padding-top: 0.9rem;
    color: var(--text-muted);
    font-size: 0.75rem;
    text-align: right;
  }
  .moment-pin {
    z-index: 1;
    display: grid;
    place-items: center;
    width: 2.5rem;
    height: 2.5rem;
    border: 1px solid var(--border);
    border-radius: 50%;
    background: var(--surface);
    color: var(--accent);
  }
  .sleep-pin {
    color: var(--accent-2);
  }
  .media-pin {
    color: var(--ring-exercise);
  }
  .moment-card {
    display: grid;
    gap: 0.45rem;
    padding: 1rem;
    color: var(--text);
    text-decoration: none;
  }
  .moment-card:hover {
    border-color: var(--accent);
    text-decoration: none;
  }
  .moment-card h3 {
    margin: 0;
    font-size: 1rem;
  }
  .moment-card p {
    margin: 0;
    color: var(--text-muted);
    font-size: 0.82rem;
  }
  .moment-link {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    color: var(--accent);
    font-size: 0.76rem;
  }
  .micro-bar {
    margin-top: 0.2rem;
  }
  .chronicle-aside {
    position: sticky;
    top: 5rem;
    display: grid;
    gap: 0.8rem;
    padding: 1.2rem;
  }
  .chronicle-aside > strong {
    color: var(--accent);
    font-size: 4.5rem;
    letter-spacing: -0.1em;
    line-height: 0.85;
  }
  .chronicle-aside p {
    margin: 0;
    line-height: 1.5;
  }
  .chronicle-aside dl {
    display: grid;
    gap: 0.65rem;
    margin: 0.5rem 0 0;
    padding-top: 0.8rem;
    border-top: 1px solid var(--border);
  }
  .chronicle-aside dl div {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .chronicle-aside dt {
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .chronicle-aside dd {
    margin: 0;
    font-size: 0.8rem;
    font-weight: 700;
  }
  .cover-page,
  .workspace,
  .field-journal,
  .quiet-dashboard {
    gap: 1.5rem;
  }
  .cover-masthead,
  .workspace-header,
  .journal-header,
  .quiet-header {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 1.5rem;
  }
  .cover-masthead h2,
  .workspace-header h2,
  .journal-header h2,
  .quiet-header h2 {
    margin: 0;
    font-size: clamp(2rem, 5vw, 4.8rem);
    letter-spacing: -0.08em;
    line-height: 0.88;
  }
  .cover-masthead h2 em,
  .journal-header h2,
  .quiet-header h2 {
    font-family: Georgia, serif;
    font-weight: 400;
  }
  .cover-masthead h2 em {
    color: var(--accent);
  }
  .cover-index,
  .journal-header,
  .quiet-mark {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .cover-index {
    display: grid;
    gap: 0.25rem;
    justify-items: end;
    text-align: right;
  }
  .cover-index strong {
    color: var(--text);
    font-size: 2rem;
    letter-spacing: -0.08em;
  }
  .cover-grid {
    display: grid;
    grid-template-columns: 1.3fr 0.7fr;
    gap: 1rem;
  }
  .cover-art {
    position: relative;
    display: grid;
    align-content: end;
    min-height: 31rem;
    padding: 1.5rem;
    overflow: hidden;
    background:
      radial-gradient(
        circle at 50% 36%,
        color-mix(in srgb, var(--accent-2) 28%, transparent),
        transparent 18%
      ),
      linear-gradient(
        145deg,
        color-mix(in srgb, var(--accent) 16%, var(--surface)),
        var(--surface)
      );
    border-color: color-mix(in srgb, var(--accent) 30%, var(--border));
  }
  .cover-sun {
    position: absolute;
    top: 29%;
    left: 50%;
    display: grid;
    place-items: center;
    width: 8rem;
    height: 8rem;
    border-radius: 50%;
    background: linear-gradient(135deg, var(--accent), var(--accent-2));
    color: var(--bg);
    font-size: 5rem;
    font-weight: 800;
    transform: translate(-50%, -50%);
    box-shadow: 0 0 60px color-mix(in srgb, var(--accent) 38%, transparent);
  }
  .cover-orbit {
    position: absolute;
    top: 29%;
    left: 50%;
    width: 20rem;
    height: 7rem;
    border: 1px solid color-mix(in srgb, var(--accent) 52%, transparent);
    border-radius: 50%;
    transform: translate(-50%, -50%) rotate(-24deg);
  }
  .orbit-two {
    width: 25rem;
    height: 10rem;
    border-color: color-mix(in srgb, var(--accent-2) 32%, transparent);
    transform: translate(-50%, -50%) rotate(31deg);
  }
  .cover-art-label {
    position: absolute;
    top: 1.1rem;
    left: 1.2rem;
    color: var(--text-muted);
    font-size: 0.68rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .cover-art > strong {
    z-index: 1;
    font-size: clamp(4rem, 10vw, 7rem);
    letter-spacing: -0.1em;
    line-height: 0.8;
  }
  .cover-art > strong small,
  .journal-index > strong small {
    margin-left: 0.3rem;
    color: var(--text-muted);
    font-size: 0.2em;
    font-weight: 500;
    letter-spacing: 0;
  }
  .cover-art p {
    z-index: 1;
    max-width: 26rem;
    margin: 0.8rem 0 0;
    color: var(--text-muted);
    line-height: 1.55;
  }
  .cover-notes {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
  }
  .cover-note {
    display: grid;
    align-content: start;
    gap: 0.7rem;
    min-height: 12rem;
    padding: 1.1rem;
  }
  .cover-note strong {
    font-size: clamp(1.9rem, 4vw, 3.2rem);
    letter-spacing: -0.08em;
    line-height: 0.92;
  }
  .cover-note p,
  .workspace-card p,
  .journal-index > p,
  .quiet-stat p {
    margin: 0;
    color: var(--text-muted);
    font-size: 0.78rem;
    line-height: 1.5;
  }
  .cover-note-wide {
    grid-column: span 2;
    min-height: 0;
    background: linear-gradient(
      135deg,
      color-mix(in srgb, var(--accent-2) 12%, var(--surface)),
      var(--surface)
    );
  }
  .cover-note-wide strong,
  .idea-card > strong {
    font-family: Georgia, serif;
    font-size: 1.35rem;
    font-weight: 400;
    letter-spacing: -0.03em;
    line-height: 1.1;
  }
  .workspace-header h2 {
    font-size: clamp(2rem, 5vw, 3.7rem);
  }
  .workspace-header .muted {
    max-width: 36rem;
    margin: 0.6rem 0 0;
  }
  .workspace-add {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
    padding: 0.65rem 0.85rem;
    border: 1px solid var(--accent);
    border-radius: 0.6rem;
    background: var(--accent);
    color: var(--bg);
    font: inherit;
    font-size: 0.78rem;
    font-weight: 700;
  }
  .workspace-layout {
    display: grid;
    grid-template-columns: 10.5rem minmax(0, 1fr);
    gap: 1rem;
  }
  .workspace-sidebar {
    display: flex;
    flex-direction: column;
    min-height: 31rem;
    padding: 0.8rem;
  }
  .workspace-logo {
    display: grid;
    place-items: center;
    width: 2.2rem;
    height: 2.2rem;
    margin-bottom: 2rem;
    border-radius: 0.6rem;
    background: var(--accent);
    color: var(--bg);
    font-size: 1.4rem;
    font-weight: 800;
  }
  .workspace-sidebar nav {
    display: grid;
    gap: 0.3rem;
  }
  .workspace-sidebar nav a {
    display: flex;
    align-items: center;
    gap: 0.55rem;
    padding: 0.55rem 0.45rem;
    border-radius: 0.45rem;
    color: var(--text-muted);
    font-size: 0.76rem;
  }
  .workspace-sidebar nav a:hover,
  .workspace-sidebar nav a.selected {
    background: color-mix(in srgb, var(--accent) 12%, transparent);
    color: var(--text);
    text-decoration: none;
  }
  .workspace-sidebar nav a.selected {
    color: var(--accent);
    font-weight: 700;
  }
  .workspace-sidebar-foot {
    display: grid;
    gap: 0.35rem;
    margin-top: auto;
    padding-top: 0.8rem;
    border-top: 1px solid var(--border);
    color: var(--text-muted);
    font-size: 0.65rem;
  }
  .workspace-sidebar-foot strong {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    color: var(--accent);
    font-size: 0.66rem;
  }
  .workspace-sidebar-foot i {
    width: 0.4rem;
    height: 0.4rem;
    border-radius: 50%;
    background: var(--accent);
  }
  .workspace-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 1rem;
    color: var(--text-muted);
    font-size: 0.74rem;
  }
  .workspace-toolbar div {
    display: flex;
    gap: 0.2rem;
    padding: 0.2rem;
    border: 1px solid var(--border);
    border-radius: 0.5rem;
    background: var(--surface);
  }
  .workspace-toolbar button {
    padding: 0.35rem 0.5rem;
    border: 0;
    border-radius: 0.3rem;
    background: transparent;
    color: var(--text-muted);
    font: inherit;
    font-size: 0.7rem;
  }
  .workspace-toolbar button.view-active {
    background: var(--surface-2);
    color: var(--text);
  }
  .workspace-columns,
  .quiet-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 0.7rem;
  }
  .workspace-column {
    display: grid;
    align-content: start;
    gap: 0.7rem;
  }
  .workspace-column > header {
    display: flex;
    align-items: center;
    gap: 0.45rem;
    padding: 0 0.2rem 0.3rem;
    border-bottom: 1px solid var(--border);
  }
  .workspace-column > header small {
    margin-left: auto;
    color: var(--text-muted);
    font-size: 0.65rem;
  }
  .column-dot {
    width: 0.45rem;
    height: 0.45rem;
    border-radius: 50%;
  }
  .column-dot.teal {
    background: var(--accent);
  }
  .column-dot.pink {
    background: var(--accent-2);
  }
  .column-dot.amber {
    background: var(--ring-exercise);
  }
  .workspace-card {
    display: grid;
    gap: 0.55rem;
    padding: 0.9rem;
    color: var(--text);
    text-decoration: none;
  }
  .workspace-card:hover {
    text-decoration: none;
    transform: translateY(-2px);
  }
  .workspace-card > strong {
    font-size: 0.9rem;
    letter-spacing: -0.02em;
  }
  .workspace-card-top {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .workspace-card-top small,
  .workspace-meta {
    color: var(--text-muted);
    font-size: 0.65rem;
  }
  .hero-block > strong {
    font-size: 3.6rem;
    letter-spacing: -0.1em;
    line-height: 0.8;
  }
  .hero-block > strong small {
    margin-left: 0.3rem;
    color: var(--text-muted);
    font-size: 0.3em;
    font-weight: 500;
  }
  .workspace-progress {
    height: 0.25rem;
    overflow: hidden;
    border-radius: 99px;
    background: var(--surface-2);
  }
  .workspace-progress i {
    display: block;
    width: var(--fill);
    height: 100%;
    background: var(--accent);
  }
  .idea-card {
    min-height: 11rem;
    background: linear-gradient(
      145deg,
      color-mix(in srgb, var(--ring-exercise) 13%, var(--surface)),
      var(--surface)
    );
  }
  .field-journal {
    gap: 1.8rem;
  }
  .journal-header {
    align-items: center;
  }
  .journal-number {
    color: var(--accent);
    font-size: 0.75rem;
    font-weight: 800;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .journal-header h2 {
    font-size: clamp(2rem, 5vw, 3.7rem);
  }
  .journal-weather {
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  .journal-layout {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 15rem;
    align-items: start;
    gap: 2rem;
  }
  .journal-paper {
    position: relative;
    display: grid;
    padding: 0 1rem 0 5.5rem;
  }
  .journal-rule {
    position: absolute;
    top: 0;
    bottom: 0;
    left: 4.2rem;
    width: 1px;
    background: color-mix(in srgb, var(--accent-2) 45%, var(--border));
  }
  .journal-entry {
    display: grid;
    grid-template-columns: 4.5rem minmax(0, 1fr);
    gap: 1rem;
    padding: 1.2rem 0;
    border-bottom: 1px solid var(--border);
  }
  .journal-entry time {
    color: var(--text-muted);
    font-size: 0.7rem;
    text-align: right;
  }
  .journal-entry h3 {
    margin: 0.45rem 0;
    font-family: Georgia, serif;
    font-size: 1.45rem;
    font-weight: 400;
  }
  .journal-entry p {
    max-width: 33rem;
    margin: 0;
    color: var(--text-muted);
    font-size: 0.85rem;
    line-height: 1.6;
  }
  .journal-tag {
    color: var(--accent);
    font-size: 0.65rem;
    font-weight: 750;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .journal-index {
    position: sticky;
    top: 5rem;
    display: grid;
    gap: 0.8rem;
    padding: 1.2rem;
  }
  .journal-index > strong {
    color: var(--accent);
    font-size: 4rem;
    letter-spacing: -0.1em;
    line-height: 0.85;
  }
  .journal-stats {
    display: grid;
    gap: 0.55rem;
    padding-top: 0.7rem;
    border-top: 1px solid var(--border);
  }
  .journal-stats div {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .journal-stats span,
  .journal-index > p {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .journal-stats b {
    font-size: 0.75rem;
  }
  .quiet-dashboard {
    padding: clamp(1rem, 5vw, 3rem);
    border: 1px solid
      color-mix(in srgb, var(--ring-exercise) 26%, var(--border));
    border-radius: 1.25rem;
    background:
      radial-gradient(
        circle at 85% 20%,
        color-mix(in srgb, var(--accent-2) 13%, transparent),
        transparent 22%
      ),
      linear-gradient(
        145deg,
        color-mix(in srgb, var(--ring-exercise) 7%, var(--surface)),
        var(--surface)
      );
    box-shadow: var(--tile-shadow);
  }
  .quiet-header h2 {
    font-size: clamp(2rem, 5vw, 3.8rem);
  }
  .quiet-hero {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 2rem;
    min-height: 20rem;
    padding: clamp(1.2rem, 5vw, 3rem) 0;
    border-top: 1px solid var(--border);
    border-bottom: 1px solid var(--border);
  }
  .quiet-hero > div:first-child {
    display: grid;
    gap: 0.8rem;
  }
  .quiet-hero strong {
    font-size: clamp(6rem, 17vw, 12rem);
    font-weight: 300;
    letter-spacing: -0.14em;
    line-height: 0.7;
  }
  .quiet-hero p {
    max-width: 22rem;
    margin: 0;
    color: var(--text-muted);
    font-family: Georgia, serif;
    font-size: 1.15rem;
    line-height: 1.4;
  }
  .quiet-bloom {
    position: relative;
    width: 14rem;
    height: 14rem;
    margin-right: 3rem;
  }
  .quiet-bloom span {
    position: absolute;
    inset: 20%;
    border: 1px solid color-mix(in srgb, var(--accent) 55%, transparent);
    border-radius: 50% 50% 45% 55%;
    transform: rotate(30deg);
  }
  .quiet-bloom span:nth-child(2) {
    inset: 8%;
    border-color: color-mix(in srgb, var(--accent-2) 48%, transparent);
    transform: rotate(120deg);
  }
  .quiet-bloom span:nth-child(3) {
    inset: 32%;
    background: color-mix(in srgb, var(--ring-exercise) 22%, transparent);
    border-color: var(--ring-exercise);
    transform: rotate(210deg);
  }
  .quiet-stat {
    display: grid;
    gap: 0.45rem;
  }
  .quiet-stat > span {
    color: var(--text-muted);
    font-size: 0.68rem;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .quiet-stat strong {
    font-size: 2.2rem;
    font-weight: 400;
    letter-spacing: -0.07em;
  }
  .quiet-stat strong small {
    color: var(--text-muted);
    font-size: 0.35em;
  }
  .quiet-footer {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    padding-top: 0.8rem;
    border-top: 1px solid var(--border);
    color: var(--text-muted);
    font-family: Georgia, serif;
    font-size: 0.9rem;
  }
  .quiet-footer a {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    font-family: system-ui, sans-serif;
    font-size: 0.74rem;
  }

  @media (max-width: 760px) {
    .lab-header,
    .demo-heading,
    .chronicle-heading {
      display: grid;
    }
    .source-pill,
    .date-button {
      width: fit-content;
    }
    .editorial-hero {
      grid-template-columns: 1fr;
    }
    .hero-rings {
      justify-self: start;
      padding: 1rem 0 0;
    }
    .editorial-grid,
    .command-grid,
    .chronicle-body {
      grid-template-columns: 1fr;
    }
    .command-center {
      grid-template-columns: 1fr;
    }
    .command-rail,
    .chronicle-aside {
      position: static;
    }
    .command-activities {
      grid-column: span 1;
    }
    .command-kpis {
      grid-template-columns: repeat(2, 1fr);
    }
    .vital-list {
      grid-template-columns: repeat(3, 1fr);
    }
    .variant-tabs {
      width: 100%;
      overflow-x: auto;
    }
    .variant-tabs button {
      flex: 1 0 auto;
    }
    .signal-strip {
      grid-template-columns: repeat(2, 1fr);
    }
    .signal-intro {
      grid-column: span 2;
    }
    .signal-metric:nth-child(2n + 3) {
      padding-left: 0;
      border-left: 0;
    }
    .cover-masthead,
    .workspace-header,
    .journal-header,
    .quiet-header {
      align-items: flex-start;
      flex-direction: column;
    }
    .cover-index {
      justify-items: start;
      text-align: left;
    }
    .cover-grid,
    .workspace-layout,
    .journal-layout,
    .workspace-columns,
    .quiet-grid {
      grid-template-columns: 1fr;
    }
    .cover-art {
      min-height: 25rem;
    }
    .workspace-sidebar {
      min-height: 0;
    }
    .workspace-sidebar nav {
      grid-template-columns: repeat(4, 1fr);
    }
    .workspace-sidebar-foot,
    .journal-index {
      display: none;
    }
    .quiet-hero {
      min-height: 16rem;
    }
    .quiet-bloom {
      width: 8rem;
      height: 8rem;
      margin-right: 0;
    }
    .quiet-footer {
      align-items: flex-start;
      flex-direction: column;
    }
  }
</style>
