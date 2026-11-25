import { createContext } from "svelte";
import type { DesignLanguage, ThemeDefinition } from "../theme/themes";
import type { ThemeComponents } from "./registry";

export type ThemeContext = {
  language: () => DesignLanguage;
  definition: () => ThemeDefinition<ThemeComponents>;
  select: (language: DesignLanguage) => void;
};

export const [useTheme, provideTheme] = createContext<ThemeContext>();
