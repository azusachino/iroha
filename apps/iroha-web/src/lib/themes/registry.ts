// The composition registry is a shared product asset. This app module is a
// compatibility boundary for route code and tests; it does not define a
// second registry or theme implementation.
export {
  THEME_DEFINITIONS,
  getThemeDefinition,
  hasThemeRoute,
  isDesignLanguage,
} from "@iroha/shared/theme-ui/registry";
