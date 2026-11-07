import { realpathSync } from "node:fs";
import { createRequire } from "node:module";
import { describe, expect, it } from "vitest";

const root = new URL("../../../../", import.meta.url);
const hosts = ["apps/iroha-web/"];
const shared = "packages/iroha-shared/";

function packagePath(directory: string, name: string) {
  const require = createRequire(new URL(`${directory}package.json`, root));
  return realpathSync(require.resolve(`${name}/package.json`));
}

describe("frontend workspace resolution", () => {
  it("links host to the owning shared package", () => {
    const expected = realpathSync(new URL(`${shared}package.json`, root));
    for (const host of hosts) {
      expect(packagePath(host, "@iroha/shared")).toBe(expected);
    }
  });

  it("uses one Svelte runtime for host, shared peers and test root", () => {
    const expected = packagePath("", "svelte");
    for (const directory of [...hosts, shared]) {
      expect(packagePath(directory, "svelte")).toBe(expected);
    }
  });

  it("uses one ECharts registry across host and shared charts", () => {
    const expected = packagePath(shared, "echarts");
    for (const host of hosts) {
      expect(packagePath(host, "echarts")).toBe(expected);
    }
  });
});
