import type { Snippet } from "svelte";
import type { DesignLanguage } from "../theme/themes";

export type ShellThemeProps = {
  children: Snippet;
  theme: DesignLanguage;
};
