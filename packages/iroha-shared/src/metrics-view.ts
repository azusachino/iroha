import type { Snippet } from "svelte";
import type { DesignLanguage } from "./themes";

export type MetricsThemeProps = {
  theme: DesignLanguage;
  children?: Snippet;
};
