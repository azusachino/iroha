import type { DesignLanguage } from "./themes";

export interface MediaRow {
  id: string;
  title: string;
  media_type: string;
  item_role: string;
  cover_image_url?: string;
  status?: string;
  position?: number;
  total?: number;
  unit?: string;
  progress_percent?: number;
  last_update_at: string;
  rating?: number;
  hidden_from_continue?: boolean;
  native_title?: string;
  episode_count?: number;
  chapter_count?: number;
  started_on?: string;
  completed_on?: string;
}

export interface MediaHomeEvent {
  id: string;
  media_id: string;
  title: string;
  native_title?: string;
  cover_image_url?: string;
  event_type: string;
  occurred_at: string;
  unit?: string;
  position?: number;
  total?: number;
  progress_percent?: number;
  rating?: number;
}

export interface MediaEvent {
  id: string;
  event_type: string;
  event_at: string;
  unit?: string;
  position?: number;
  total?: number;
  progress_percent?: number;
  rating?: number;
  note?: string;
}

export interface MediaCompletionBucket {
  year: number;
  count: number;
}

export interface MediaScoreBucket {
  score: number;
  count: number;
}

export interface MediaTypeBucket {
  type: string;
  count: number;
}

export interface MediaAggregates {
  totals: {
    item_count: number;
    completed_count: number;
    current_completed_count: number;
    this_year_completed: number;
    average_rating: number;
  };
  completions_by_year: MediaCompletionBucket[];
  score_distribution: MediaScoreBucket[];
  type_split: MediaTypeBucket[];
}

export interface MediaPage {
  items: MediaRow[];
  next_cursor: string | null;
  has_more: boolean;
  status_counts?: Record<string, number>;
  active_count?: number;
}

export interface MediaDetail {
  item: MediaRow;
  work: {
    id: string;
    work_kind: string;
    primary_title: string;
    original_title: string;
    original_language: string;
    first_release_date?: string;
    description: string;
  };
  progress?: {
    status: string;
    unit: string;
    position?: number;
    total?: number;
    progress_percent?: number;
    started_on?: string;
    last_update_at?: string;
    completed_on?: string;
    play_count: number;
  };
  creators: { id: string; name: string; role: string }[];
  relations: {
    id: string;
    relation_type: string;
    direction: string;
    related_item_id: string;
    related_title: string;
    related_type: string;
    cover_image_url?: string;
  }[];
  events: MediaEvent[];
  updates: MediaChange[];
}

export interface MediaEventInput {
  media_id: string;
  event_type: string;
  event_at: string;
  source_kind?: string;
  idempotency_key: string;
  unit?: string;
  position?: number;
  total?: number;
  progress_percent?: number;
  rating?: number;
  rating_scale?: number;
  note?: string;
}

export interface MediaChange {
  id: string;
  media_id: string;
  title: string;
  native_title?: string;
  cover_image_url?: string;
  source_kind: string;
  change_kind: string;
  time_basis: string;
  observed_at: string;
  effective_at?: string;
  effective_on?: string;
  date_precision?: "year" | "month" | "day";
  provider_recorded_at?: string;
  status?: string;
  unit?: string;
  position?: number;
  total?: number;
  progress_percent?: number;
  rating?: number;
  note?: string;
  repeat_count: number;
}

export type MediaEventPage = {
  items: MediaHomeEvent[];
  next_cursor: string | null;
  has_more: boolean;
};

export type MediaChangePage = {
  items: MediaChange[];
  next_cursor: string | null;
  has_more: boolean;
};

export interface MediaDayList<T> {
  state: "ready" | "empty";
  items: T[];
  count: number;
  has_more: boolean;
}

export interface MediaDaySection {
  sessions: MediaDayList<MediaHomeEvent>;
  dated_updates: MediaDayList<MediaChange>;
  coverage: {
    timezone: string;
    date: string;
  };
}

export interface MediaDetailThemeProps {
  detail: MediaDetail;
  progress: number;
  hasKnownTotal: boolean;
  theme: DesignLanguage;
}

export interface MediaThemeProps {
  items: MediaRow[];
  aggregates: MediaAggregates;
  family: string;
  status: string;
  completedYear: string;
  yearOptions: MediaCompletionBucket[];
  typeFamilies: { type: string; count: number }[];
  completions: MediaCompletionBucket[];
  scores: MediaScoreBucket[];
  currentCompletedCount: number;
  activeCount: number;
  theme: DesignLanguage;
  onFamily: (value: string) => void;
  onStatus: (value: string) => void;
  onYear: (value: string) => void;
  onLoadMore: () => void;
  hasMore: boolean;
  loadingMore: boolean;
}

const TYPE_LABELS: Record<string, string> = {
  anime_season: "Anime",
  movie: "Movie",
  ona: "ONA",
  ova: "OVA",
  special: "Special",
  manga: "Manga",
  one_shot: "One-shot",
  light_novel: "Light novel",
  book: "Book",
  game: "Game",
  real: "Live action",
  music: "Music",
};

export function mediaTypeLabel(type: string): string {
  return TYPE_LABELS[type] ?? type.replaceAll("_", " ");
}

export function mediaEventLabel(eventType: string): string {
  return eventType.replaceAll("_", " ");
}

export function mediaEventVerb(event: MediaHomeEvent): string {
  if (event.rating != null) return "Rated";
  if (event.progress_percent != null && event.progress_percent >= 100)
    return "Finished";
  if (event.position != null || event.progress_percent != null)
    return "Progressed";
  return "Updated library";
}

const EPISODE_COUNTED_TYPES = new Set([
  "anime_season",
  "movie",
  "ona",
  "ova",
  "special",
]);
const CHAPTER_COUNTED_TYPES = new Set([
  "manga",
  "one_shot",
  "light_novel",
  "book",
]);

export function cleanDescription(html?: string | null): string {
  if (!html) return "";
  return html
    .replace(/<br\s*\/?\s*>/gi, "\n")
    .replace(/<\/?(i|b|em|strong)>/gi, "")
    .replace(/<[^>]+>/g, "")
    .trim();
}

export function mediaWorkTotal(
  mediaType?: string | null,
  episodeCount?: number | null,
  chapterCount?: number | null,
): number | undefined {
  if (mediaType && EPISODE_COUNTED_TYPES.has(mediaType)) {
    return episodeCount ?? undefined;
  }
  if (mediaType && CHAPTER_COUNTED_TYPES.has(mediaType)) {
    return chapterCount ?? undefined;
  }
  return undefined;
}

export function mediaTypeFamily(type: string): string {
  if (["manga", "one_shot", "light_novel", "novel"].includes(type))
    return "Manga & light novels";
  if (type === "book") return "Books";
  if (["anime_season", "movie", "ona", "ova", "special"].includes(type))
    return "Anime";
  if (type === "game") return "Games";
  return "Other";
}

// Category color is semantic and intentionally shared by charts and cards so
// a media family never changes meaning between the aggregate and evidence
// layers. Text labels remain present for non-color access.
export function mediaTypeColor(type: string): string {
  const family = mediaTypeFamily(type);
  if (family === "Anime") return "var(--mark-teal)";
  if (family === "Manga & light novels") return "var(--mark-magenta)";
  if (family === "Books") return "var(--mark-violet)";
  if (family === "Games") return "var(--mark-amber)";
  return "var(--text-muted)";
}

const DASH = "—";

function effectiveTotal(
  status?: string | null,
  position?: number | null,
  total?: number | null,
  workTotal?: number | null,
): number | undefined {
  if (total != null && Number.isFinite(total) && total > 0) return total;
  if (
    status === "completed" &&
    position != null &&
    Number.isFinite(position) &&
    position > 0
  ) {
    return position;
  }
  if (workTotal != null && Number.isFinite(workTotal) && workTotal > 0) {
    return workTotal;
  }
  return undefined;
}

export function formatProgressCount(
  position?: number | null,
  total?: number | null,
  unit?: string | null,
  status?: string | null,
  workTotal?: number | null,
): string {
  if (position == null || !Number.isFinite(position)) return DASH;
  const suffix = unit ? ` ${unit}` : "";
  const knownTotal = effectiveTotal(status, position, total, workTotal);
  if (knownTotal != null) return `${position}/${knownTotal}${suffix}`;
  return `${position}${suffix}`;
}

export function progressPercent(
  status?: string | null,
  position?: number | null,
  total?: number | null,
  percent?: number | null,
  workTotal?: number | null,
): number {
  if (percent != null && Number.isFinite(percent)) {
    return Math.min(100, Math.max(0, percent));
  }
  const knownTotal = effectiveTotal(status, position, total, workTotal);
  if (position != null && Number.isFinite(position) && knownTotal != null) {
    return Math.min(100, Math.max(0, (position / knownTotal) * 100));
  }
  return 0;
}
