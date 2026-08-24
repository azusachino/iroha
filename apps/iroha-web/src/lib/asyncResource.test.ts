import { describe, expect, it } from "vitest";
import { createAsyncResource } from "./asyncResource.svelte";

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}

describe("createAsyncResource", () => {
  it("keeps the latest response when an older request finishes last", async () => {
    const resource = createAsyncResource<number>();
    const older = deferred<number>();
    const latest = deferred<number>();

    const olderRun = resource.run(() => older.promise);
    const latestRun = resource.run(() => latest.promise);

    latest.resolve(2);
    expect(await latestRun).toBe(2);
    older.resolve(1);
    expect(await olderRun).toBeUndefined();
    expect(resource.data).toBe(2);
    expect(resource.ready).toBe(true);
    expect(resource.loading).toBe(false);
  });
});
