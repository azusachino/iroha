import type { Component } from "svelte";
import {
  defineThemeRegistry,
  isDesignLanguage,
  type DesignLanguage,
  type ThemeDefinition,
  type ThemeRoute,
} from "@iroha/shared/themes";
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
import AtlasExpenses from "$lib/themes/atlas/Expenses.svelte";
import GrapherExpenses from "$lib/themes/grapher/Expenses.svelte";
import FieldJournalExpenses from "$lib/themes/field-journal/Expenses.svelte";
import PhenologyExpenses from "$lib/themes/phenology/Expenses.svelte";
import SoundMapExpenses from "$lib/themes/sound-map/Expenses.svelte";
import ArchiveExpenses from "$lib/themes/archive/Expenses.svelte";
import AtlasReports from "$lib/themes/atlas/Reports.svelte";
import GrapherReports from "$lib/themes/grapher/Reports.svelte";
import FieldJournalReports from "$lib/themes/field-journal/Reports.svelte";
import PhenologyReports from "$lib/themes/phenology/Reports.svelte";
import SoundMapReports from "$lib/themes/sound-map/Reports.svelte";
import ArchiveReports from "$lib/themes/archive/Reports.svelte";
import AtlasMetrics from "$lib/themes/atlas/Metrics.svelte";
import GrapherMetrics from "$lib/themes/grapher/Metrics.svelte";
import FieldJournalMetrics from "$lib/themes/field-journal/Metrics.svelte";
import PhenologyMetrics from "$lib/themes/phenology/Metrics.svelte";
import SoundMapMetrics from "$lib/themes/sound-map/Metrics.svelte";
import ArchiveMetrics from "$lib/themes/archive/Metrics.svelte";

const registry = defineThemeRegistry<Component<any>>({
  atlas: {
    implementation: "curated",
    primitives: { periodControl: { appearance: "atlas" } },
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
      expenses: AtlasExpenses,
      reports: AtlasReports,
      metrics: AtlasMetrics,
    },
  },
  grapher: {
    implementation: "curated",
    primitives: { periodControl: { appearance: "grapher" } },
    components: {
      shell: GrapherShell,
      today: GrapherToday,
      daily: GrapherDaily,
      activities: GrapherActivities,
      sleep: GrapherSleep,
      expenses: GrapherExpenses,
      reports: GrapherReports,
      metrics: GrapherMetrics,
    },
  },
  "field-journal": {
    implementation: "curated",
    primitives: { periodControl: { appearance: "field-journal" } },
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
      expenses: FieldJournalExpenses,
      reports: FieldJournalReports,
      metrics: FieldJournalMetrics,
    },
  },
  phenology: {
    implementation: "curated",
    primitives: { periodControl: { appearance: "phenology" } },
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
      expenses: PhenologyExpenses,
      reports: PhenologyReports,
      metrics: PhenologyMetrics,
    },
  },
  "sound-map": {
    implementation: "curated",
    primitives: { periodControl: { appearance: "sound-map" } },
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
      expenses: SoundMapExpenses,
      reports: SoundMapReports,
      metrics: SoundMapMetrics,
    },
  },
  archive: {
    implementation: "curated",
    primitives: { periodControl: { appearance: "archive" } },
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
      expenses: ArchiveExpenses,
      reports: ArchiveReports,
      metrics: ArchiveMetrics,
    },
  },
});

export const THEME_DEFINITIONS = registry.definitions;

export { isDesignLanguage };

export function getThemeDefinition(
  language: DesignLanguage,
): ThemeDefinition<Component<any>> {
  return registry.get(language);
}

export function hasThemeRoute(
  theme: ThemeDefinition<Component<any>>,
  route: ThemeRoute,
): boolean {
  return registry.hasRoute(theme, route);
}
