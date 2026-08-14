import type { Snippet } from "svelte";
import type { DesignLanguage } from "./themes";

export type ShellThemeProps = {
  children: Snippet;
  theme: DesignLanguage;
};
