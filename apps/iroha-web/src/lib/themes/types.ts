import type { Component } from "svelte";

export const THEME_ROUTES = [
  "today",
  "daily",
  "activities",
  "sleep",
  "media",
  "share",
] as const;

export type ThemeRoute = (typeof THEME_ROUTES)[number];
export const THEME_IDS = [
  "atlas",
  "grapher",
  "field-journal",
  "phenology",
  "sound-map",
  "archive",
] as const;

export type DesignLanguage = (typeof THEME_IDS)[number];
export type ThemeImplementationStatus = "palette-only" | "curated";

export type ThemeComponentSet = Partial<Record<ThemeRoute, Component<any>>> & {
  shell: Component<any>;
};

export type ThemeDefinition = {
  id: DesignLanguage;
  label: string;
  hint: string;
  description: string;
  implementation: ThemeImplementationStatus;
  components?: ThemeComponentSet;
};
