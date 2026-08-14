import type { Job } from "$lib/api";

export type JobGroup = {
  kind: string;
  latest: Job;
  count: number;
  failedCount: number;
  activeCount: number;
};

export function groupJobs(jobs: Job[]): JobGroup[] {
  const groups = new Map<string, JobGroup>();
  for (const job of jobs) {
    const current = groups.get(job.kind);
    if (!current) {
      groups.set(job.kind, {
        kind: job.kind,
        latest: job,
        count: 1,
        failedCount: job.status === "failed" ? 1 : 0,
        activeCount:
          job.status === "queued" || job.status === "running" ? 1 : 0,
      });
      continue;
    }
    current.count += 1;
    if (job.status === "failed") current.failedCount += 1;
    if (job.status === "queued" || job.status === "running")
      current.activeCount += 1;
    if (job.created_at > current.latest.created_at) current.latest = job;
  }
  return [...groups.values()].sort(
    (left, right) =>
      right.latest.created_at.localeCompare(left.latest.created_at) ||
      left.kind.localeCompare(right.kind),
  );
}
