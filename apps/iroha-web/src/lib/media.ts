// The API returns raw media_type values (anime_season, manga, one_shot,
// light_novel, game, movie, ova, ona, special, book, real, music, …).
// Collapse those into display families for charts/filters, and give each
// raw type a readable label for per-item badges.

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

// Small family-colored dot so anime / manga-books / games read apart at a
// glance; the text label still carries the meaning (color is not the only cue).
export function mediaTypeColor(type: string): string {
  const family = mediaTypeFamily(type);
  if (family === "Anime") return "var(--mark-teal)";
  if (family === "Manga & books") return "var(--mark-magenta)";
  if (family === "Games") return "var(--mark-amber)";
  return "var(--text-muted)";
}
