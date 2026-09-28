<script lang="ts">
  import { onMount } from "svelte";
  import { Play, RefreshCw } from "@lucide/svelte";
  import {
    cancelJob,
    listJobs,
    listSchedules,
    retryJob,
    runSchedule,
    setScheduleEnabled,
    type Job,
    type JobSchedule,
  } from "$lib/api";
  import { formatDate } from "$lib/format";

  let schedules = $state<JobSchedule[]>([]);
  let jobs = $state<Job[]>([]);
  let loading = $state(true);
  let busy = $state<string | null>(null);
  let error = $state<string | null>(null);
  let notice = $state<string | null>(null);
  let filter = $state<"all" | "failed" | "active">("all");

  const shown = $derived(
    jobs.filter((job) =>
      filter === "failed"
        ? job.status === "failed"
        : filter === "active"
          ? job.status === "queued" || job.status === "running"
          : true,
    ),
  );

  function kindLabel(kind: string): string {
    return kind.replaceAll("_", " ");
  }

  function tone(status: Job["status"]): string {
    return status === "completed" ? "good" : status === "failed" ? "bad" : "";
  }

  async function load() {
    loading = true;
    error = null;
    try {
      [schedules, jobs] = await Promise.all([
        listSchedules(),
        listJobs({ limit: 50 }),
      ]);
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    } finally {
      loading = false;
    }
  }

  async function act(
    key: string,
    message: string,
    action: () => Promise<unknown>,
  ) {
    busy = key;
    error = null;
    notice = null;
    try {
      await action();
      notice = message;
      await load();
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    } finally {
      busy = null;
    }
  }

  onMount(() => {
    void load();
  });
</script>

<div class="stack">
  <section class="admin-card" aria-labelledby="schedules-title">
    <header>
      <div>
        <h2 id="schedules-title">Schedules</h2>
        <small>Recurring work the background worker runs.</small>
      </div>
      <button
        type="button"
        class="admin-button"
        onclick={() => void load()}
        disabled={loading}><RefreshCw size={14} /> Refresh</button
      >
    </header>
    {#if schedules.length}
      <ul class="admin-list">
        {#each schedules as schedule (schedule.kind)}
          <li>
            <div>
              <strong>{kindLabel(schedule.kind)}</strong>
              <small>
                every {schedule.schedule_expr} ·
                {schedule.enabled
                  ? schedule.next_run_at
                    ? `next ${formatDate(schedule.next_run_at)}`
                    : "not scheduled"
                  : "paused"}
                {#if schedule.last_run_at}· last {formatDate(
                    schedule.last_run_at,
                  )}{/if}
              </small>
            </div>
            <div class="admin-actions">
              <label class="toggle">
                <input
                  type="checkbox"
                  checked={schedule.enabled}
                  disabled={busy !== null}
                  onchange={(event) => {
                    const enabled = event.currentTarget.checked;
                    void act(
                      `toggle-${schedule.kind}`,
                      `${kindLabel(schedule.kind)} ${enabled ? "resumed" : "paused"}.`,
                      () => setScheduleEnabled(schedule.kind, enabled),
                    );
                  }}
                />
                Enabled
              </label>
              <button
                type="button"
                class="admin-button"
                disabled={busy !== null || !schedule.enabled}
                onclick={() =>
                  void act(
                    `run-${schedule.kind}`,
                    `${kindLabel(schedule.kind)} will run on the worker's next poll.`,
                    () => runSchedule(schedule.kind),
                  )}><Play size={14} /> Run now</button
              >
            </div>
          </li>
        {/each}
      </ul>
    {:else if !loading}
      <p class="admin-muted">No schedules are registered.</p>
    {/if}
  </section>

  <section class="admin-card" aria-labelledby="jobs-title">
    <header>
      <div>
        <h2 id="jobs-title">Recent jobs</h2>
        <small>The latest 50 executions.</small>
      </div>
      <div class="admin-actions" role="group" aria-label="Filter jobs">
        {#each [["all", "All"], ["failed", "Failed"], ["active", "Queued or running"]] as [value, label] (value)}
          <button
            type="button"
            class="admin-button"
            class:primary={filter === value}
            aria-pressed={filter === value}
            onclick={() => (filter = value as typeof filter)}>{label}</button
          >
        {/each}
      </div>
    </header>

    {#if notice}<p class="admin-muted" role="status">{notice}</p>{/if}
    {#if error}<p class="admin-error" role="alert">{error}</p>{/if}

    {#if shown.length}
      <ul class="admin-list">
        {#each shown as job (job.id)}
          <li>
            <div>
              <strong>{kindLabel(job.kind)}</strong>
              <span class={`status ${tone(job.status)}`}>{job.status}</span>
              <small
                >created {formatDate(job.created_at)} · attempt {job.attempts}/{job.max_attempts}</small
              >
              {#if job.error_message}<small class="admin-error"
                  >{job.error_message}</small
                >{/if}
            </div>
            <div class="admin-actions">
              {#if job.status === "failed" || job.status === "canceled"}
                <button
                  type="button"
                  class="admin-button"
                  disabled={busy !== null}
                  onclick={() =>
                    void act(job.id, "Queued a new attempt.", () =>
                      retryJob(job.id),
                    )}>Retry</button
                >
              {/if}
              {#if job.status === "queued"}
                <button
                  type="button"
                  class="admin-button"
                  disabled={busy !== null}
                  onclick={() =>
                    void act(job.id, "Job canceled.", () => cancelJob(job.id))}
                  >Cancel</button
                >
              {/if}
            </div>
          </li>
        {/each}
      </ul>
    {:else if !loading}
      <p class="admin-muted">No jobs match this filter.</p>
    {/if}
  </section>
</div>

<style>
  .stack {
    display: grid;
    gap: 0.85rem;
  }

  .toggle {
    display: inline-flex;
    min-height: 2.3rem;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.85rem;
  }

  .toggle input {
    width: 1.1rem;
    height: 1.1rem;
    accent-color: var(--accent);
  }
</style>
