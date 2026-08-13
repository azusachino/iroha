import { createContext } from "svelte";
import type { Component } from "svelte";
import type { DesignLanguage, ThemeDefinition } from "@iroha/shared/themes";

export type ThemeContext = {
  language: () => DesignLanguage;
  definition: () => ThemeDefinition<Component<any>>;
  select: (language: DesignLanguage) => void;
};

export const [useTheme, provideTheme] = createContext<ThemeContext>();
