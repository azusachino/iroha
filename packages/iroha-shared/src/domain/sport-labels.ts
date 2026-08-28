// Import sources disagree on sport_type casing/wording ("Swimming" from
// Apple Health, "run" from GPX/fixtures) -- domain/activity.ts's sport_type
// is a bare string, not a strict enum, so this normalizes best-effort rather
// than assuming one canonical spelling. --sport-run/walk/hike/ride/swim/other
// (apps/iroha-web/src/routes/app.css) are the only six real color slots;
// anything unrecognized becomes "other" rather than inventing a seventh.
const SPORT_ALIASES: Record<string, string> = {
  run: "run",
  running: "run",
  walk: "walk",
  walking: "walk",
  hike: "hike",
  hiking: "hike",
  ride: "ride",
  riding: "ride",
  cycle: "ride",
  cycling: "ride",
  bike: "ride",
  biking: "ride",
  swim: "swim",
  swimming: "swim",
};

export function canonicalSport(sport: string): string {
  return SPORT_ALIASES[sport.trim().toLowerCase()] ?? "other";
}

const SPORT_LABELS: Record<string, string> = {
  run: "Run",
  walk: "Walk",
  hike: "Hike",
  ride: "Ride",
  swim: "Swim",
  other: "Other",
};

export function sportLabel(sport: string): string {
  return SPORT_LABELS[canonicalSport(sport)] ?? sport;
}

export function sportColorVar(sport: string): string {
  return `--sport-${canonicalSport(sport)}`;
}
