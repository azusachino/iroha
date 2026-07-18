import { createContext } from "svelte";
import type { DesignLanguage, ThemeDefinition } from "$lib/themes/types";

export type ThemeContext = {
  language: () => DesignLanguage;
  definition: () => ThemeDefinition;
  select: (language: DesignLanguage) => void;
};

export const [useTheme, provideTheme] = createContext<ThemeContext>();
