import { describe, expect, it } from "vitest";
import { groupJobs } from "$lib/jobs";
import type { Job } from "$lib/api";

function job(overrides: Partial<Job>): Job {
  return {
    id: "job_1",
    kind: "geocode_refresh",
    status: "completed",
    attempts: 1,
    max_attempts: 3,
    run_after: "2026-08-13T08:20:31Z",
    created_at: "2026-08-13T08:20:31Z",
    updated_at: "2026-08-13T08:20:32Z",
    ...overrides,
  };
}

describe("groupJobs", () => {
  it("collapses repeated executions while retaining the latest status", () => {
    const groups = groupJobs([
      job({ id: "job_1", created_at: "2026-08-13T08:20:31Z" }),
      job({
        id: "job_2",
        status: "failed",
        created_at: "2026-08-13T08:20:30Z",
      }),
      job({
        id: "job_3",
        kind: "media_sync_anilist",
        created_at: "2026-08-12T08:20:31Z",
      }),
    ]);

    expect(groups).toHaveLength(2);
    expect(groups[0]).toMatchObject({
      kind: "geocode_refresh",
      count: 2,
      failedCount: 1,
      latest: { id: "job_1" },
    });
  });
});
