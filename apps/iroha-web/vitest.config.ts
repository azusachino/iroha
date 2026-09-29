import { defineConfig, mergeConfig } from "vitest/config";
import viteConfig from "./vite.config.ts";

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      coverage: {
        provider: "v8",
        include: [
          "src/**/*.{js,ts,svelte}",
          "../../packages/iroha-shared/src/**/*.{js,ts,svelte}",
        ],
        exclude: ["**/*.{test,spec}.{js,ts}"],
        reporter: ["text", "json-summary"],
        reportsDirectory: "coverage",
        thresholds: {
          statements: 10.34,
          branches: 12.21,
          functions: 8.6,
          lines: 10.3,
        },
      },
    },
  }),
);
