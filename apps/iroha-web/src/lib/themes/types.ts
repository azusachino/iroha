import type { Component } from "svelte";
import type { DesignLanguage, ThemeIdentity } from "@iroha/shared/themes";

export type { DesignLanguage } from "@iroha/shared/themes";

export const THEME_ROUTES = [
  "today",
  "dashboard",
  "daily",
  "activities",
  "activity-detail",
  "sleep",
  "media",
  "media-detail",
  "expenses",
  "reports",
] as const;

export type ThemeRoute = (typeof THEME_ROUTES)[number];
export type ThemeImplementationStatus = "palette-only" | "preview" | "curated";

export type ThemeComponentSet = Partial<Record<ThemeRoute, Component<any>>> & {
  shell: Component<any>;
};

export type ThemeDefinition = ThemeIdentity & {
  implementation: ThemeImplementationStatus;
  routes?: readonly ThemeRoute[];
  components?: ThemeComponentSet;
};
