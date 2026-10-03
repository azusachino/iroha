import { expect, test } from "@playwright/test";
import { resolve } from "node:path";
import { resolveConfig } from "vite";

for (const consumer of ["iroha-web", "iroha-public-site"]) {
  test(`${consumer} resolves shared and host chart imports to one runtime`, async () => {
    const root = resolve(process.cwd(), "..", consumer);
    const config = await resolveConfig(
      { root, configFile: resolve(root, "vite.config.ts") },
      "serve",
    );
    const resolver = config.createResolver();
    const shared = resolve(
      root,
      "../../packages/iroha-shared/src/components/YearProgressChart.svelte",
    );
    const host = resolve(root, "src/routes/+page.svelte");
    for (const dependency of [
      "echarts/core",
      "echarts/charts",
      "echarts/components",
      "echarts/renderers",
      "zrender",
    ]) {
      const hostModule = await resolver(dependency, host);
      const sharedModule = await resolver(dependency, shared);
      expect(hostModule).toBeDefined();
      expect(sharedModule).toBe(hostModule);
    }
  });
}
