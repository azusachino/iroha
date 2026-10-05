import type { Page } from "@playwright/test";
import type { Task, Job } from "../src/lib/api";
import { fakeSession } from "./session";

const instant = "2026-10-04T04:00:00Z";
const task: Task = {
  id: "synthetic-task",
  title: "Synthetic intention",
  notes: "Synthetic context",
  status: "open",
  due_date: "2026-10-04",
  priority: 2,
  source: "manual",
  created_at: instant,
  updated_at: instant,
};
const job: Job = {
  id: "synthetic-job",
  kind: "media_sync_anilist",
  status: "completed",
  attempts: 1,
  max_attempts: 3,
  run_after: instant,
  created_at: instant,
  updated_at: instant,
};

export async function installTogoFixtures(page: Page) {
  await page.clock.install({ time: new Date(instant) });
  const fixture = {
    failures: new Set<string>(),
    holds: new Map<string, Promise<void>>(),
    requests: [] as string[],
    unknown: [] as string[],
    tasks: [task] as Task[],
    jobs: [job] as Job[],
  };
  await page.route("**/api/v1/**", async (route) => {
    const url = new URL(route.request().url());
    const path = url.pathname;
    const method = route.request().method();
    const dependency = path.startsWith("/api/v1/actions/")
      ? "action"
      : path === "/api/v1/tasks"
        ? method === "GET"
          ? "tasks"
          : "add"
        : path.startsWith("/api/v1/tasks/")
          ? "finish"
          : path === "/api/v1/jobs"
            ? "jobs"
            : "unknown";
    fixture.requests.push(`${method} ${path}${url.search}`);
    const taskRows = fixture.tasks.map((row) => ({ ...row }));
    const jobRows = fixture.jobs.map((row) => ({ ...row }));
    const readNumber = fixture.requests.filter((request) =>
      request.startsWith(`GET ${path}?`),
    ).length;
    await (fixture.holds.get(`${dependency}:${readNumber}`) ??
      fixture.holds.get(dependency));
    if (fixture.failures.has(dependency))
      return route.fulfill({
        status: 503,
        json: { error: `Synthetic ${dependency} failure` },
      });
    if (dependency === "tasks") return route.fulfill({ json: taskRows });
    if (dependency === "jobs") {
      const kinds = url.searchParams.get("kind")?.split(",");
      return route.fulfill({
        json: jobRows.filter((row) => !kinds || kinds.includes(row.kind)),
      });
    }
    if (dependency === "add") {
      const input = route.request().postDataJSON();
      const added: Task = {
        ...task,
        ...input,
        id: "synthetic-added",
        source: "web",
      };
      fixture.tasks = [added, ...fixture.tasks];
      return route.fulfill({ status: 201, json: added });
    }
    if (dependency === "finish") {
      const id = path.split("/").at(-1);
      const finished: Task = {
        ...fixture.tasks.find((row) => row.id === id)!,
        status: "completed",
        completed_at: instant,
      };
      fixture.tasks = fixture.tasks.map((row) =>
        row.id === id ? finished : row,
      );
      return route.fulfill({ json: finished });
    }
    if (dependency === "action") {
      const queued: Job = {
        ...job,
        id: "synthetic-queued",
        kind: path.split("/").at(-1)!.replaceAll("-", "_"),
        status: "queued",
        attempts: 0,
      };
      fixture.jobs = [queued, ...fixture.jobs];
      return route.fulfill({ status: 202, json: queued });
    }
    fixture.unknown.push(path);
    return route.fulfill({
      status: 404,
      json: { error: "Unmodelled endpoint" },
    });
  });
  await fakeSession(page, {
    setup_required: false,
    authenticated: true,
    username: "synthetic-owner",
    display_name: "Synthetic owner",
  });
  return fixture;
}
