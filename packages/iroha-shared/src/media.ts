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
  activeCount: number;
  theme: DesignLanguage;
  onFamily: (value: string) => void;
  onStatus: () => void;
  onYear: () => void;
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

export function mediaTypeFamily(type: string): string {
  if (["manga", "one_shot", "light_novel", "book", "novel"].includes(type))
    return "Manga & books";
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
  if (family === "Manga & books") return "var(--mark-magenta)";
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
  if (status === "completed" && position != null && Number.isFinite(position)) {
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
