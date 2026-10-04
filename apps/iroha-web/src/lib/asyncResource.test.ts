import { describe, expect, it } from "vitest";
import { createAsyncResource } from "./asyncResource.svelte";

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (cause: Error) => void;
  const promise = new Promise<T>((done, fail) => {
    resolve = done;
    reject = fail;
  });
  return { promise, resolve, reject };
}

describe("createAsyncResource", () => {
  it("invalidates a pending success without clearing observed data", async () => {
    const resource = createAsyncResource<number>();
    await resource.run(async () => 7);
    const pending = deferred<number>();
    const pendingRun = resource.run(() => pending.promise);
    resource.invalidate();
    expect(resource.data).toBe(7);
    expect(resource.ready).toBe(true);
    expect(resource.loading).toBe(false);
    pending.resolve(99);
    expect(await pendingRun).toBeUndefined();
    expect(resource.data).toBe(7);
    expect(resource.error).toBeNull();
  });

  it("does not certify a first load canceled before it succeeds", async () => {
    const resource = createAsyncResource<number>();
    const pending = deferred<number>();
    const pendingRun = resource.run(() => pending.promise);
    resource.invalidate();
    pending.resolve(0);
    expect(await pendingRun).toBeUndefined();
    expect(resource.data).toBeNull();
    expect(resource.ready).toBe(false);
    expect(resource.loading).toBe(false);
  });

  it("does not resurrect an invalidated request failure", async () => {
    const resource = createAsyncResource<number>();
    await resource.run(async () => 7);
    const pending = deferred<number>();
    const pendingRun = resource.run(() => pending.promise);
    resource.invalidate();
    pending.reject(new Error("superseded failure"));
    expect(await pendingRun).toBeUndefined();
    expect(resource.data).toBe(7);
    expect(resource.ready).toBe(true);
    expect(resource.error).toBeNull();
    expect(resource.loading).toBe(false);
  });

  it("clears an existing error when invalidated without clearing observed data", async () => {
    const resource = createAsyncResource<number>();
    await resource.run(async () => 7);
    await resource.run(async () => {
      throw new Error("failed read");
    });
    expect(resource.error).toBe("failed read");
    resource.invalidate();
    expect(resource.error).toBeNull();
    expect(resource.data).toBe(7);
    expect(resource.ready).toBe(true);
    expect(resource.loading).toBe(false);
  });

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
