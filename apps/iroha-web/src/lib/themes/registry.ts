import {
  THEME_ROUTES,
  type DesignLanguage,
  type ThemeDefinition,
} from "$lib/themes/types";
import GrapherActivities from "$lib/themes/grapher/Activities.svelte";
import GrapherDaily from "$lib/themes/grapher/Daily.svelte";
import GrapherShare from "$lib/themes/grapher/Share.svelte";
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
import AtlasShare from "$lib/themes/atlas/Share.svelte";
import FieldJournalShare from "$lib/themes/field-journal/Share.svelte";
import PhenologyShare from "$lib/themes/phenology/Share.svelte";
import SoundMapShare from "$lib/themes/sound-map/Share.svelte";
import ArchiveShare from "$lib/themes/archive/Share.svelte";
import AtlasMedia from "$lib/themes/atlas/Media.svelte";
import FieldJournalMedia from "$lib/themes/field-journal/Media.svelte";
import PhenologyMedia from "$lib/themes/phenology/Media.svelte";
import SoundMapMedia from "$lib/themes/sound-map/Media.svelte";
import ArchiveMedia from "$lib/themes/archive/Media.svelte";

export const THEME_DEFINITIONS = [
  {
    id: "atlas",
    label: "Iroha Atlas",
    hint: "places and routes",
    description: "A cartographic language for movement, places, and distance.",
    implementation: "preview",
    routes: ["today", "daily", "activities", "sleep", "media", "share"],
    components: {
      shell: AtlasShell,
      today: AtlasToday,
      daily: AtlasDaily,
      activities: AtlasActivities,
      sleep: AtlasSleep,
      share: AtlasShare,
      media: AtlasMedia,
    },
  },
  {
    id: "grapher",
    label: "Iroha Grapher",
    hint: "trends and comparisons",
    description: "An evidence-first language for comparison and change.",
    implementation: "curated",
    routes: ["today", "daily", "activities", "sleep", "media", "share"],
    components: {
      shell: GrapherShell,
      today: GrapherToday,
      daily: GrapherDaily,
      activities: GrapherActivities,
      sleep: GrapherSleep,
      share: GrapherShare,
    },
  },
  {
    id: "field-journal",
    label: "Iroha Field Journal",
    hint: "days and evidence",
    description: "A dated, narrative language for the shape of a day.",
    implementation: "preview",
    routes: ["today", "daily", "activities", "sleep", "share"],
    components: {
      shell: FieldJournalShell,
      today: FieldJournalToday,
      daily: FieldJournalDaily,
      activities: FieldJournalActivities,
      sleep: FieldJournalSleep,
      share: FieldJournalShare,
      media: FieldJournalMedia,
    },
  },
  {
    id: "phenology",
    label: "Iroha Phenology",
    hint: "sleep and seasons",
    description:
      "A cyclical language for recovery, rest, and unfolding patterns.",
    implementation: "preview",
    routes: ["today", "daily", "activities", "sleep", "media", "share"],
    components: {
      shell: PhenologyShell,
      today: PhenologyToday,
      daily: PhenologyDaily,
      activities: PhenologyActivities,
      sleep: PhenologySleep,
      share: PhenologyShare,
      media: PhenologyMedia,
    },
  },
  {
    id: "sound-map",
    label: "Iroha Sound Map",
    hint: "rhythm and intensity",
    description: "A rhythmic language for cadence, intensity, and flow.",
    implementation: "preview",
    routes: ["today", "daily", "activities", "sleep", "media", "share"],
    components: {
      shell: SoundMapShell,
      today: SoundMapToday,
      daily: SoundMapDaily,
      activities: SoundMapActivities,
      sleep: SoundMapSleep,
      share: SoundMapShare,
      media: SoundMapMedia,
    },
  },
  {
    id: "archive",
    label: "Iroha Archive",
    hint: "media and history",
    description: "A chronological language for collections and memory.",
    implementation: "preview",
    routes: ["today", "daily", "activities", "sleep", "media", "share"],
    components: {
      shell: ArchiveShell,
      today: ArchiveToday,
      daily: ArchiveDaily,
      activities: ArchiveActivities,
      sleep: ArchiveSleep,
      share: ArchiveShare,
      media: ArchiveMedia,
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
