import { createContext } from "svelte";
import type { DesignLanguage, ThemeDefinition } from "../theme/themes";
import type { ThemeComponent } from "./registry";

export type ThemeContext = {
  language: () => DesignLanguage;
  definition: () => ThemeDefinition<ThemeComponent>;
  select: (language: DesignLanguage) => void;
};

export const [useTheme, provideTheme] = createContext<ThemeContext>();
