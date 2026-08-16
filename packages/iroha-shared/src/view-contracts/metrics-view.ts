import type { Snippet } from "svelte";
import type { DesignLanguage } from "../theme/themes";

export type MetricsThemeProps = {
  theme: DesignLanguage;
  children?: Snippet;
};
