import type { Component } from "svelte";
import {
  defineThemeRegistry,
  isDesignLanguage,
  type DesignLanguage,
  type ThemeDefinition,
  type ThemeRoute,
} from "../themes";
import GrapherActivities from "./grapher/Activities.svelte";
import GrapherDaily from "./grapher/Daily.svelte";
import GrapherShell from "./grapher/Shell.svelte";
import GrapherSleep from "./grapher/Sleep.svelte";
import GrapherToday from "./grapher/Today.svelte";
import GrapherDashboard from "./grapher/Dashboard.svelte";
import GrapherActivityDetail from "./grapher/ActivityDetail.svelte";
import GrapherMedia from "./grapher/Media.svelte";
import GrapherMediaDetail from "./grapher/MediaDetail.svelte";
import FieldJournalShell from "./field-journal/Shell.svelte";
import FieldJournalToday from "./field-journal/Today.svelte";
import FieldJournalDaily from "./field-journal/Daily.svelte";
import FieldJournalActivities from "./field-journal/Activities.svelte";
import AtlasShell from "./atlas/Shell.svelte";
import PhenologyShell from "./phenology/Shell.svelte";
import SoundMapShell from "./sound-map/Shell.svelte";
import ArchiveShell from "./archive/Shell.svelte";
import AtlasToday from "./atlas/Today.svelte";
import PhenologyToday from "./phenology/Today.svelte";
import SoundMapToday from "./sound-map/Today.svelte";
import ArchiveToday from "./archive/Today.svelte";
import AtlasDaily from "./atlas/Daily.svelte";
import PhenologyDaily from "./phenology/Daily.svelte";
import SoundMapDaily from "./sound-map/Daily.svelte";
import ArchiveDaily from "./archive/Daily.svelte";
import AtlasActivities from "./atlas/Activities.svelte";
import PhenologyActivities from "./phenology/Activities.svelte";
import SoundMapActivities from "./sound-map/Activities.svelte";
import ArchiveActivities from "./archive/Activities.svelte";
import AtlasSleep from "./atlas/Sleep.svelte";
import FieldJournalSleep from "./field-journal/Sleep.svelte";
import PhenologySleep from "./phenology/Sleep.svelte";
import SoundMapSleep from "./sound-map/Sleep.svelte";
import ArchiveSleep from "./archive/Sleep.svelte";
import AtlasMedia from "./atlas/Media.svelte";
import FieldJournalMedia from "./field-journal/Media.svelte";
import PhenologyMedia from "./phenology/Media.svelte";
import SoundMapMedia from "./sound-map/Media.svelte";
import ArchiveMedia from "./archive/Media.svelte";
import AtlasDashboard from "./atlas/Dashboard.svelte";
import FieldJournalDashboard from "./field-journal/Dashboard.svelte";
import PhenologyDashboard from "./phenology/Dashboard.svelte";
import SoundMapDashboard from "./sound-map/Dashboard.svelte";
import ArchiveDashboard from "./archive/Dashboard.svelte";
import AtlasActivityDetail from "./atlas/ActivityDetail.svelte";
import FieldJournalActivityDetail from "./field-journal/ActivityDetail.svelte";
import PhenologyActivityDetail from "./phenology/ActivityDetail.svelte";
import SoundMapActivityDetail from "./sound-map/ActivityDetail.svelte";
import ArchiveActivityDetail from "./archive/ActivityDetail.svelte";
import AtlasMediaDetail from "./atlas/MediaDetail.svelte";
import FieldJournalMediaDetail from "./field-journal/MediaDetail.svelte";
import PhenologyMediaDetail from "./phenology/MediaDetail.svelte";
import SoundMapMediaDetail from "./sound-map/MediaDetail.svelte";
import ArchiveMediaDetail from "./archive/MediaDetail.svelte";
import AtlasExpenses from "./atlas/Expenses.svelte";
import GrapherExpenses from "./grapher/Expenses.svelte";
import FieldJournalExpenses from "./field-journal/Expenses.svelte";
import PhenologyExpenses from "./phenology/Expenses.svelte";
import SoundMapExpenses from "./sound-map/Expenses.svelte";
import ArchiveExpenses from "./archive/Expenses.svelte";
import AtlasReports from "./atlas/Reports.svelte";
import GrapherReports from "./grapher/Reports.svelte";
import FieldJournalReports from "./field-journal/Reports.svelte";
import PhenologyReports from "./phenology/Reports.svelte";
import SoundMapReports from "./sound-map/Reports.svelte";
import ArchiveReports from "./archive/Reports.svelte";
import AtlasMetrics from "./atlas/Metrics.svelte";
import GrapherMetrics from "./grapher/Metrics.svelte";
import FieldJournalMetrics from "./field-journal/Metrics.svelte";
import PhenologyMetrics from "./phenology/Metrics.svelte";
import SoundMapMetrics from "./sound-map/Metrics.svelte";
import ArchiveMetrics from "./archive/Metrics.svelte";

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
