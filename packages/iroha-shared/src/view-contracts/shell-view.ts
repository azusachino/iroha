import type { Snippet } from "svelte";
import type { DesignLanguage } from "../theme/themes";

export type ShellThemeProps = {
  children: Snippet;
  theme: DesignLanguage;
  /** Brand mark/wordmark. Host-owned markup; the theme decides placement only. */
  brand: Snippet;
  /** Primary navigation, including any disclosure menus. Host-owned interaction. */
  nav: Snippet;
  /** Command trigger, design-language picker, and light/dark toggle. */
  actions: Snippet;
};
