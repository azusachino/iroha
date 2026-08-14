import type { Component } from "svelte";
import {
  defineThemeRegistry,
  isDesignLanguage,
  type DesignLanguage,
  type ThemeDefinition,
  type ThemeRoute,
} from "@iroha/shared/themes";
import GrapherActivities from "@iroha/shared/theme-ui/grapher/Activities.svelte";
import GrapherDaily from "@iroha/shared/theme-ui/grapher/Daily.svelte";
import GrapherShell from "@iroha/shared/theme-ui/grapher/Shell.svelte";
import GrapherSleep from "@iroha/shared/theme-ui/grapher/Sleep.svelte";
import GrapherToday from "@iroha/shared/theme-ui/grapher/Today.svelte";
import GrapherDashboard from "@iroha/shared/theme-ui/grapher/Dashboard.svelte";
import GrapherActivityDetail from "@iroha/shared/theme-ui/grapher/ActivityDetail.svelte";
import GrapherMedia from "@iroha/shared/theme-ui/grapher/Media.svelte";
import GrapherMediaDetail from "@iroha/shared/theme-ui/grapher/MediaDetail.svelte";
import FieldJournalShell from "@iroha/shared/theme-ui/field-journal/Shell.svelte";
import FieldJournalToday from "@iroha/shared/theme-ui/field-journal/Today.svelte";
import FieldJournalDaily from "@iroha/shared/theme-ui/field-journal/Daily.svelte";
import FieldJournalActivities from "@iroha/shared/theme-ui/field-journal/Activities.svelte";
import AtlasShell from "@iroha/shared/theme-ui/atlas/Shell.svelte";
import PhenologyShell from "@iroha/shared/theme-ui/phenology/Shell.svelte";
import SoundMapShell from "@iroha/shared/theme-ui/sound-map/Shell.svelte";
import ArchiveShell from "@iroha/shared/theme-ui/archive/Shell.svelte";
import AtlasToday from "@iroha/shared/theme-ui/atlas/Today.svelte";
import PhenologyToday from "@iroha/shared/theme-ui/phenology/Today.svelte";
import SoundMapToday from "@iroha/shared/theme-ui/sound-map/Today.svelte";
import ArchiveToday from "@iroha/shared/theme-ui/archive/Today.svelte";
import AtlasDaily from "@iroha/shared/theme-ui/atlas/Daily.svelte";
import PhenologyDaily from "@iroha/shared/theme-ui/phenology/Daily.svelte";
import SoundMapDaily from "@iroha/shared/theme-ui/sound-map/Daily.svelte";
import ArchiveDaily from "@iroha/shared/theme-ui/archive/Daily.svelte";
import AtlasActivities from "@iroha/shared/theme-ui/atlas/Activities.svelte";
import PhenologyActivities from "@iroha/shared/theme-ui/phenology/Activities.svelte";
import SoundMapActivities from "@iroha/shared/theme-ui/sound-map/Activities.svelte";
import ArchiveActivities from "@iroha/shared/theme-ui/archive/Activities.svelte";
import AtlasSleep from "@iroha/shared/theme-ui/atlas/Sleep.svelte";
import FieldJournalSleep from "@iroha/shared/theme-ui/field-journal/Sleep.svelte";
import PhenologySleep from "@iroha/shared/theme-ui/phenology/Sleep.svelte";
import SoundMapSleep from "@iroha/shared/theme-ui/sound-map/Sleep.svelte";
import ArchiveSleep from "@iroha/shared/theme-ui/archive/Sleep.svelte";
import AtlasMedia from "@iroha/shared/theme-ui/atlas/Media.svelte";
import FieldJournalMedia from "@iroha/shared/theme-ui/field-journal/Media.svelte";
import PhenologyMedia from "@iroha/shared/theme-ui/phenology/Media.svelte";
import SoundMapMedia from "@iroha/shared/theme-ui/sound-map/Media.svelte";
import ArchiveMedia from "@iroha/shared/theme-ui/archive/Media.svelte";
import AtlasDashboard from "@iroha/shared/theme-ui/atlas/Dashboard.svelte";
import FieldJournalDashboard from "@iroha/shared/theme-ui/field-journal/Dashboard.svelte";
import PhenologyDashboard from "@iroha/shared/theme-ui/phenology/Dashboard.svelte";
import SoundMapDashboard from "@iroha/shared/theme-ui/sound-map/Dashboard.svelte";
import ArchiveDashboard from "@iroha/shared/theme-ui/archive/Dashboard.svelte";
import AtlasActivityDetail from "@iroha/shared/theme-ui/atlas/ActivityDetail.svelte";
import FieldJournalActivityDetail from "@iroha/shared/theme-ui/field-journal/ActivityDetail.svelte";
import PhenologyActivityDetail from "@iroha/shared/theme-ui/phenology/ActivityDetail.svelte";
import SoundMapActivityDetail from "@iroha/shared/theme-ui/sound-map/ActivityDetail.svelte";
import ArchiveActivityDetail from "@iroha/shared/theme-ui/archive/ActivityDetail.svelte";
import AtlasMediaDetail from "@iroha/shared/theme-ui/atlas/MediaDetail.svelte";
import FieldJournalMediaDetail from "@iroha/shared/theme-ui/field-journal/MediaDetail.svelte";
import PhenologyMediaDetail from "@iroha/shared/theme-ui/phenology/MediaDetail.svelte";
import SoundMapMediaDetail from "@iroha/shared/theme-ui/sound-map/MediaDetail.svelte";
import ArchiveMediaDetail from "@iroha/shared/theme-ui/archive/MediaDetail.svelte";
import AtlasExpenses from "@iroha/shared/theme-ui/atlas/Expenses.svelte";
import GrapherExpenses from "@iroha/shared/theme-ui/grapher/Expenses.svelte";
import FieldJournalExpenses from "@iroha/shared/theme-ui/field-journal/Expenses.svelte";
import PhenologyExpenses from "@iroha/shared/theme-ui/phenology/Expenses.svelte";
import SoundMapExpenses from "@iroha/shared/theme-ui/sound-map/Expenses.svelte";
import ArchiveExpenses from "@iroha/shared/theme-ui/archive/Expenses.svelte";
import AtlasReports from "@iroha/shared/theme-ui/atlas/Reports.svelte";
import GrapherReports from "@iroha/shared/theme-ui/grapher/Reports.svelte";
import FieldJournalReports from "@iroha/shared/theme-ui/field-journal/Reports.svelte";
import PhenologyReports from "@iroha/shared/theme-ui/phenology/Reports.svelte";
import SoundMapReports from "@iroha/shared/theme-ui/sound-map/Reports.svelte";
import ArchiveReports from "@iroha/shared/theme-ui/archive/Reports.svelte";
import AtlasMetrics from "@iroha/shared/theme-ui/atlas/Metrics.svelte";
import GrapherMetrics from "@iroha/shared/theme-ui/grapher/Metrics.svelte";
import FieldJournalMetrics from "@iroha/shared/theme-ui/field-journal/Metrics.svelte";
import PhenologyMetrics from "@iroha/shared/theme-ui/phenology/Metrics.svelte";
import SoundMapMetrics from "@iroha/shared/theme-ui/sound-map/Metrics.svelte";
import ArchiveMetrics from "@iroha/shared/theme-ui/archive/Metrics.svelte";

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
      "activity-detail": GrapherActivityDetail,
      sleep: GrapherSleep,
      media: GrapherMedia,
      "media-detail": GrapherMediaDetail,
      dashboard: GrapherDashboard,
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
