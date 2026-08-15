import { onMount } from "svelte";
import { replaceState } from "$app/navigation";
import { page } from "$app/state";
import {
  getBriefing,
  getDailyDates,
  listTasks,
  updateTask,
  type DailyRow,
  type SleepSession,
  type Activity,
  type MediaDaySection,
  type BriefingResponse,
  type Task,
} from "$lib/api";
import type { Ring } from "@iroha/shared/theme-ui/components/RingGauge.svelte";
import { todayInTimezone } from "@iroha/shared/date";
import { IROHA_TIMEZONE } from "$lib/config";
import { formatDateOnly } from "$lib/format";

// All state, derivations, and data loading for the Today route, kept out of
// the .svelte file so the template isn't interleaved with ~300 lines of
// business logic. `theme` (a Svelte context lookup) stays in the component
// itself -- it's a rendering-position concern, not Today-specific state.
export function createTodayState() {
  let briefing = $state<BriefingResponse | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let toGoTasks = $state<Task[]>([]);
  let taskError = $state<string | null>(null);
  let briefingRequestVersion = 0;
  let taskRequestVersion = 0;

  const today = todayInTimezone(new Date(), IROHA_TIMEZONE);
  const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;

  // The selected day — the spine everything on this page snapshots to.
  // Seeded from ?date= so a refresh or shared link lands back on the same
  // day instead of resetting to today; a future or malformed date falls
  // back to today rather than showing an empty briefing.
  function dayFromUrl(): string {
    const requested = page.url.searchParams.get("date");
    if (requested && DATE_RE.test(requested) && requested <= today) {
      return requested;
    }
    return today;
  }

  let day = $state<string>(dayFromUrl());
  let pickerOpen = $state(false);
  let availableDays = $state<Set<string>>(new Set());

  type BriefingList<T> = { items: T[]; has_more: boolean };
  function sectionData<T>(key: string): BriefingList<T> {
    const section = briefing?.sections.find((item) => item.key === key);
    return (
      (section?.data as BriefingList<T> | undefined) ?? {
        items: [],
        has_more: false,
      }
    );
  }

  function mediaSection(): MediaDaySection {
    const section = briefing?.sections.find((item) => item.key === "media");
    const data = section?.data as Partial<MediaDaySection> | undefined;
    return {
      sessions: data?.sessions ?? {
        state: "empty",
        items: [],
        count: 0,
        has_more: false,
      },
      dated_updates: data?.dated_updates ?? {
        state: "empty",
        items: [],
        count: 0,
        has_more: false,
      },
      coverage: data?.coverage ?? {
        timezone: IROHA_TIMEZONE,
        date: briefing?.date ?? day,
      },
    };
  }

  const daily = $derived(sectionData<DailyRow>("daily"));
  const sleep = $derived(sectionData<SleepSession>("sleep"));
  const activities = $derived(sectionData<Activity>("activities"));
  const media = $derived(mediaSection());
  const dRow = $derived(daily.items[0]);
  const nights = $derived(sleep.items);
  const mainNight = $derived(nights.find((n) => n.is_main_sleep) ?? nights[0]);
  const acts = $derived(activities.items);
  const mediaEvents = $derived(media.sessions.items);
  const mediaUpdates = $derived(media.dated_updates.items);
  const dailyRing = $derived(dRow?.ring);
  const hasRing = $derived(!!dailyRing && dailyRing.move_goal_kcal > 0);
  const ringData = $derived<Ring[]>(
    hasRing && dailyRing
      ? [
          {
            label: "Move",
            value: dailyRing.move_kcal,
            goal: dailyRing.move_goal_kcal,
            unit: "kcal",
            color: "var(--ring-move)",
          },
          {
            label: "Exercise",
            value: dailyRing.exercise_min,
            goal: dailyRing.exercise_goal_min,
            unit: "min",
            color: "var(--ring-exercise)",
          },
          {
            label: "Stand",
            value: dailyRing.stand_hours,
            goal: dailyRing.stand_goal_hours,
            unit: "h",
            color: "var(--ring-stand)",
          },
        ]
      : [],
  );

  const vitals = $derived(
    dRow
      ? [
          { l: "Resting HR", v: dRow.resting_hr, u: "bpm", d: 0 },
          { l: "HRV", v: dRow.hrv_sdnn, u: "ms", d: 0 },
          { l: "SpO₂", v: dRow.spo2_avg, u: "%", d: 1 },
          { l: "Respiratory", v: dRow.respiratory_rate, u: "/min", d: 1 },
          { l: "VO₂max", v: dRow.vo2max, u: "", d: 1 },
          { l: "Body mass", v: dRow.body_mass_kg, u: "kg", d: 1 },
        ].filter((m) => typeof m.v === "number")
      : [],
  );

  const dayLabel = $derived(formatDateOnly(day));
  // During a date change, keep the last committed snapshot mounted. Its
  // canonical date keeps the old values from being presented as the new day
  // while the next request is in flight.
  const dataDay = $derived(briefing?.date ?? day);
  const dataDayLabel = $derived(formatDateOnly(dataDay));
  const dayHasData = $derived(
    briefing?.sections.some((section) => {
      if (section.state !== "ready") return false;
      const data = section.data as {
        items?: unknown[];
        sessions?: { items?: unknown[] };
        dated_updates?: { items?: unknown[] };
      };
      return section.key === "media"
        ? Boolean(
            data.sessions?.items?.length || data.dated_updates?.items?.length,
          )
        : Boolean(data.items?.length);
    }) ?? false,
  );
  const daysSet = $derived(
    availableDays.size > 0 ? availableDays : new Set([day]),
  );
  const canMoveNext = $derived(day < today);
  const daySignal = $derived(
    mainNight
      ? {
          value: `${Math.round(mainNight.efficiency * 100)}%`,
          label: "sleep efficiency",
        }
      : dailyRing && dailyRing.move_goal_kcal > 0
        ? {
            value: `${Math.round((dailyRing.move_kcal / dailyRing.move_goal_kcal) * 100)}%`,
            label: "move goal",
          }
        : { value: "—", label: "no baseline" },
  );

  function shift(delta: number) {
    if (delta > 0 && day >= today) return;
    // All in UTC: parsing local midnight then emitting toISOString (UTC)
    // silently dropped a day in +hh zones — hence "left = two days back".
    const d = new Date(day + "T00:00:00Z");
    d.setUTCDate(d.getUTCDate() + delta);
    const next = d.toISOString().slice(0, 10);
    if (next <= today && next !== day) day = next;
  }
  // Arrow keys scrub days (ignored while typing in a field); Escape closes the picker.
  function onKey(e: KeyboardEvent) {
    const t = e.target as HTMLElement | null;
    if (
      t &&
      (t.tagName === "INPUT" ||
        t.tagName === "TEXTAREA" ||
        t.tagName === "SELECT" ||
        t.isContentEditable)
    )
      return;
    if (e.key === "ArrowLeft" || e.key === "ArrowRight") {
      e.preventDefault();
      shift(e.key === "ArrowLeft" ? -1 : 1);
    } else if (e.key === "Escape") pickerOpen = false;
  }
  function num(v: number | null | undefined, digits: number): string {
    if (typeof v !== "number" || !Number.isFinite(v)) return "—";
    return v.toLocaleString(undefined, {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
    });
  }

  async function loadBriefing(selectedDay: string) {
    const requestVersion = ++briefingRequestVersion;
    loading = true;
    error = null;
    try {
      const next = await getBriefing(selectedDay);
      if (requestVersion === briefingRequestVersion) briefing = next;
    } catch (e) {
      if (requestVersion === briefingRequestVersion) {
        error = e instanceof Error ? e.message : String(e);
      }
    } finally {
      if (requestVersion === briefingRequestVersion) loading = false;
    }
  }

  async function loadAvailableDays() {
    try {
      availableDays = new Set(await getDailyDates());
    } catch {
      // The briefing remains useful even when the calendar index is unavailable.
    }
  }

  $effect(() => {
    void loadBriefing(day);
    void loadTasks(day);
  });

  // Keep ?date= in sync with the selected day -- replaceState rather than
  // goto so scrubbing days doesn't spam browser history, just the current
  // entry. Omitted entirely for today so the common-case URL stays plain "/".
  $effect(() => {
    const url = new URL(window.location.href);
    if (day === today) {
      url.searchParams.delete("date");
    } else {
      url.searchParams.set("date", day);
    }
    if (url.search !== window.location.search) {
      replaceState(url, page.state);
    }
  });

  onMount(() => {
    void loadAvailableDays();
  });

  async function loadTasks(selectedDay: string) {
    const requestVersion = ++taskRequestVersion;
    taskError = null;
    try {
      const next = await listTasks({
        status: "open",
        due: selectedDay,
        limit: 5,
      });
      if (requestVersion === taskRequestVersion) toGoTasks = next;
    } catch (cause) {
      if (requestVersion === taskRequestVersion) {
        taskError = cause instanceof Error ? cause.message : String(cause);
      }
    }
  }

  async function finishTask(task: Task) {
    try {
      await updateTask(task.id, "completed");
      toGoTasks = toGoTasks.filter((item) => item.id !== task.id);
    } catch (cause) {
      taskError = cause instanceof Error ? cause.message : String(cause);
    }
  }

  return {
    today,
    get briefing() {
      return briefing;
    },
    get loading() {
      return loading;
    },
    get error() {
      return error;
    },
    get toGoTasks() {
      return toGoTasks;
    },
    get taskError() {
      return taskError;
    },
    get day() {
      return day;
    },
    set day(value: string) {
      day = value;
    },
    get pickerOpen() {
      return pickerOpen;
    },
    set pickerOpen(value: boolean) {
      pickerOpen = value;
    },
    get dRow() {
      return dRow;
    },
    get mainNight() {
      return mainNight;
    },
    get dailyRing() {
      return dailyRing;
    },
    get hasRing() {
      return hasRing;
    },
    get acts() {
      return acts;
    },
    get mediaEvents() {
      return mediaEvents;
    },
    get mediaUpdates() {
      return mediaUpdates;
    },
    get ringData() {
      return ringData;
    },
    get vitals() {
      return vitals;
    },
    get dayLabel() {
      return dayLabel;
    },
    get dataDay() {
      return dataDay;
    },
    get dataDayLabel() {
      return dataDayLabel;
    },
    get dayHasData() {
      return dayHasData;
    },
    get daysSet() {
      return daysSet;
    },
    get canMoveNext() {
      return canMoveNext;
    },
    get daySignal() {
      return daySignal;
    },
    shift,
    onKey,
    num,
    finishTask,
  };
}
