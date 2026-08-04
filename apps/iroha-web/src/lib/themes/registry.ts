import {
  THEME_ROUTES,
  type DesignLanguage,
  type ThemeDefinition,
  type ThemeImplementationStatus,
} from "$lib/themes/types";
import GrapherActivities from "$lib/themes/grapher/Activities.svelte";
import GrapherDaily from "$lib/themes/grapher/Daily.svelte";
import GrapherShell from "$lib/themes/grapher/Shell.svelte";
import GrapherSleep from "$lib/themes/grapher/Sleep.svelte";
import GrapherToday from "$lib/themes/grapher/Today.svelte";
import FieldJournalShell from "$lib/themes/field-journal/Shell.svelte";
import FieldJournalToday from "$lib/themes/field-journal/Today.svelte";
import FieldJournalDaily from "$lib/themes/field-journal/Daily.svelte";
import FieldJournalActivities from "$lib/themes/field-journal/Activities.svelte";
import AtlasShell from "$lib/themes/atlas/Shell.svelte";
import PhenologyShell from "$lib/themes/phenology/Shell.svelte";
import SoundMapShell from "$lib/themes/sound-map/Shell.svelte";
import ArchiveShell from "$lib/themes/archive/Shell.svelte";
import AtlasToday from "$lib/themes/atlas/Today.svelte";
import PhenologyToday from "$lib/themes/phenology/Today.svelte";
import SoundMapToday from "$lib/themes/sound-map/Today.svelte";
import ArchiveToday from "$lib/themes/archive/Today.svelte";
import AtlasDaily from "$lib/themes/atlas/Daily.svelte";
import PhenologyDaily from "$lib/themes/phenology/Daily.svelte";
import SoundMapDaily from "$lib/themes/sound-map/Daily.svelte";
import ArchiveDaily from "$lib/themes/archive/Daily.svelte";
import AtlasActivities from "$lib/themes/atlas/Activities.svelte";
import PhenologyActivities from "$lib/themes/phenology/Activities.svelte";
import SoundMapActivities from "$lib/themes/sound-map/Activities.svelte";
import ArchiveActivities from "$lib/themes/archive/Activities.svelte";
import AtlasSleep from "$lib/themes/atlas/Sleep.svelte";
import FieldJournalSleep from "$lib/themes/field-journal/Sleep.svelte";
import PhenologySleep from "$lib/themes/phenology/Sleep.svelte";
import SoundMapSleep from "$lib/themes/sound-map/Sleep.svelte";
import ArchiveSleep from "$lib/themes/archive/Sleep.svelte";
import AtlasMedia from "$lib/themes/atlas/Media.svelte";
import FieldJournalMedia from "$lib/themes/field-journal/Media.svelte";
import PhenologyMedia from "$lib/themes/phenology/Media.svelte";
import SoundMapMedia from "$lib/themes/sound-map/Media.svelte";
import ArchiveMedia from "$lib/themes/archive/Media.svelte";
import AtlasDashboard from "$lib/themes/atlas/Dashboard.svelte";
import FieldJournalDashboard from "$lib/themes/field-journal/Dashboard.svelte";
import PhenologyDashboard from "$lib/themes/phenology/Dashboard.svelte";
import SoundMapDashboard from "$lib/themes/sound-map/Dashboard.svelte";
import ArchiveDashboard from "$lib/themes/archive/Dashboard.svelte";
import AtlasActivityDetail from "$lib/themes/atlas/ActivityDetail.svelte";
import FieldJournalActivityDetail from "$lib/themes/field-journal/ActivityDetail.svelte";
import PhenologyActivityDetail from "$lib/themes/phenology/ActivityDetail.svelte";
import SoundMapActivityDetail from "$lib/themes/sound-map/ActivityDetail.svelte";
import ArchiveActivityDetail from "$lib/themes/archive/ActivityDetail.svelte";
import AtlasMediaDetail from "$lib/themes/atlas/MediaDetail.svelte";
import FieldJournalMediaDetail from "$lib/themes/field-journal/MediaDetail.svelte";
import PhenologyMediaDetail from "$lib/themes/phenology/MediaDetail.svelte";
import SoundMapMediaDetail from "$lib/themes/sound-map/MediaDetail.svelte";
import ArchiveMediaDetail from "$lib/themes/archive/MediaDetail.svelte";

export const THEME_DEFINITIONS = [
  {
    id: "atlas",
    label: "Iroha Atlas",
    hint: "places and routes",
    description: "A cartographic language for movement, places, and distance.",
    implementation: "curated",
    routes: [
      "today",
      "dashboard",
      "daily",
      "activities",
      "activity-detail",
      "sleep",
      "media",
      "media-detail",
    ],
    components: {
      shell: AtlasShell,
      today: AtlasToday,
      daily: AtlasDaily,
      activities: AtlasActivities,
      "activity-detail": AtlasActivityDetail,
      sleep: AtlasSleep,
      media: AtlasMedia,
      "media-detail": AtlasMediaDetail,
      dashboard: AtlasDashboard,
    },
  },
  {
    id: "grapher",
    label: "Iroha Grapher",
    hint: "trends and comparisons",
    description: "An evidence-first language for comparison and change.",
    implementation: "curated",
    routes: ["today", "daily", "activities", "sleep"],
    components: {
      shell: GrapherShell,
      today: GrapherToday,
      daily: GrapherDaily,
      activities: GrapherActivities,
      sleep: GrapherSleep,
    },
  },
  {
    id: "field-journal",
    label: "Iroha Field Journal",
    hint: "days and evidence",
    description:
      "A dated, observational language for entries, continuity, and the shape of a day.",
    implementation: "curated",
    routes: [
      "today",
      "dashboard",
      "daily",
      "activities",
      "activity-detail",
      "sleep",
      "media",
      "media-detail",
    ],
    components: {
      shell: FieldJournalShell,
      today: FieldJournalToday,
      daily: FieldJournalDaily,
      activities: FieldJournalActivities,
      "activity-detail": FieldJournalActivityDetail,
      sleep: FieldJournalSleep,
      media: FieldJournalMedia,
      "media-detail": FieldJournalMediaDetail,
      dashboard: FieldJournalDashboard,
    },
  },
  {
    id: "phenology",
    label: "Iroha Phenology",
    hint: "sleep and seasons",
    description:
      "A cyclical language for recovery, rest, and unfolding patterns.",
    implementation: "curated",
    routes: [
      "today",
      "dashboard",
      "daily",
      "activities",
      "activity-detail",
      "sleep",
      "media",
      "media-detail",
    ],
    components: {
      shell: PhenologyShell,
      today: PhenologyToday,
      daily: PhenologyDaily,
      activities: PhenologyActivities,
      "activity-detail": PhenologyActivityDetail,
      sleep: PhenologySleep,
      media: PhenologyMedia,
      "media-detail": PhenologyMediaDetail,
      dashboard: PhenologyDashboard,
    },
  },
  {
    id: "sound-map",
    label: "Iroha Sound Map",
    hint: "rhythm and intensity",
    description: "A rhythmic language for cadence, intensity, and flow.",
    implementation: "curated",
    routes: [
      "today",
      "dashboard",
      "daily",
      "activities",
      "activity-detail",
      "sleep",
      "media",
      "media-detail",
    ],
    components: {
      shell: SoundMapShell,
      today: SoundMapToday,
      daily: SoundMapDaily,
      activities: SoundMapActivities,
      "activity-detail": SoundMapActivityDetail,
      sleep: SoundMapSleep,
      media: SoundMapMedia,
      "media-detail": SoundMapMediaDetail,
      dashboard: SoundMapDashboard,
    },
  },
  {
    id: "archive",
    label: "Iroha Archive",
    hint: "media and history",
    description: "A chronological language for collections and memory.",
    // Cast (rather than a bare literal) so `as const` below doesn't narrow
    // every entry's `implementation` to the literal "curated" -- that would
    // make `=== "preview"` checks elsewhere (registry.test.ts,
    // DesignLanguagePicker) unreachable at the type level now that no theme
    // is still in preview.
    implementation: "curated" as ThemeImplementationStatus,
    routes: [
      "today",
      "dashboard",
      "daily",
      "activities",
      "activity-detail",
      "sleep",
      "media",
      "media-detail",
    ],
    components: {
      shell: ArchiveShell,
      today: ArchiveToday,
      daily: ArchiveDaily,
      activities: ArchiveActivities,
      "activity-detail": ArchiveActivityDetail,
      sleep: ArchiveSleep,
      media: ArchiveMedia,
      "media-detail": ArchiveMediaDetail,
      dashboard: ArchiveDashboard,
    },
  },
] as const satisfies readonly ThemeDefinition[];

export function isDesignLanguage(
  value: string | null | undefined,
): value is DesignLanguage {
  return THEME_DEFINITIONS.some((theme) => theme.id === value);
}

export function getThemeDefinition(language: DesignLanguage): ThemeDefinition {
  const theme = THEME_DEFINITIONS.find((item) => item.id === language);
  if (!theme) throw new Error(`Unknown Iroha design language: ${language}`);
  return theme;
}

export function hasThemeRoute(
  theme: ThemeDefinition,
  route: (typeof THEME_ROUTES)[number],
): boolean {
  return theme.routes?.includes(route) ?? false;
}
