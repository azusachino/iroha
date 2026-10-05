<script lang="ts">
  import { onMount } from "svelte";
  import { Check, ListTodo, Play, RefreshCw } from "@lucide/svelte";
  import {
    createTask,
    listJobs,
    listTasks,
    triggerAction,
    updateTask,
    type Job,
    type Task,
  } from "$lib/api";
  import { APP_VERSION } from "$lib/config";
  import { formatDate } from "$lib/format";
  import { todayInTimezone } from "@iroha/shared/format/date";
  import { IROHA_TIMEZONE } from "$lib/config";
  import { createAsyncResource } from "$lib/asyncResource.svelte";
  import LoadingBoundary from "$lib/components/LoadingBoundary.svelte";
  import RetryNotice from "@iroha/shared/theme-ui/components/RetryNotice.svelte";

  const TASK_READ_LIMIT = 100;
  const COMPLETED_TASK_DISPLAY_LIMIT = 10;
  const JOB_READ_LIMIT = 30;
  const ACTION_JOB_KINDS =
    "media_sync_anilist,media_sync_bangumi,media_bridge_refresh";
  const POLL_INITIAL_DELAY_MS = 4000;
  const POLL_MAX_DELAY_MS = 30_000;
  const POLL_BACKOFF_MULTIPLIER = 1.5;

  const today = todayInTimezone(new Date(), IROHA_TIMEZONE);
  const tasksResource = createAsyncResource<Task[]>();
  const jobsResource = createAsyncResource<Job[]>();
  const openTasks = $derived(
    (tasksResource.data ?? []).filter((task) => task.status === "open"),
  );
  const completedTasks = $derived(
    (tasksResource.data ?? [])
      .filter((task) => {
        const completed = new Date(task.completed_at ?? "");
        return (
          task.status === "completed" &&
          Number.isFinite(completed.getTime()) &&
          todayInTimezone(completed, IROHA_TIMEZONE) === today
        );
      })
      .slice(0, COMPLETED_TASK_DISPLAY_LIMIT),
  );
  const jobs = $derived(jobsResource.data ?? []);
  let tasksTarget = $state<HTMLElement>();
  let jobsTarget = $state<HTMLElement>();
  let title = $state("");
  let notes = $state("");
  let dueDate = $state(today);
  let priority = $state("0");
  let saving = $state(false);
  let createError = $state<string | null>(null);
  let finishError = $state<{ id: string; message: string } | null>(null);
  let actionError = $state<string | null>(null);
  let actionPending = $state(false);
  let taskReceipt = $state<string | null>(null);
  // Request metadata only; the resource owns errors, readiness and stale guards.
  let lastJobRead = $state<"manual" | "poll">("manual");
  const pollError = $derived(
    lastJobRead === "poll" ? jobsResource.error : null,
  );
  const jobsError = $derived(
    lastJobRead === "manual" ? jobsResource.error : null,
  );
  let pollTimer: number | null = null;
  let pollDelay = POLL_INITIAL_DELAY_MS;

  const activeJobs = $derived(
    jobs.filter((job) => job.status === "queued" || job.status === "running"),
  );
  const activeActionKinds = $derived(
    new Set(activeJobs.map((job) => job.kind)),
  );

  onMount(() => {
    const onVisibilityChange = () => {
      pollDelay = POLL_INITIAL_DELAY_MS;
      schedulePolling();
    };
    document.addEventListener("visibilitychange", onVisibilityChange);
    void loadTasks();
    void loadJobs().finally(schedulePolling);
    return () => {
      stopPolling();
      document.removeEventListener("visibilitychange", onVisibilityChange);
    };
  });

  function stopPolling() {
    if (pollTimer != null) {
      window.clearTimeout(pollTimer);
      pollTimer = null;
    }
  }

  function schedulePolling() {
    stopPolling();
    if (document.visibilityState === "hidden" || activeJobs.length === 0)
      return;
    pollTimer = window.setTimeout(async () => {
      pollTimer = null;
      await loadJobs("poll");
      pollDelay = activeJobs.length
        ? Math.min(
            Math.round(pollDelay * POLL_BACKOFF_MULTIPLIER),
            POLL_MAX_DELAY_MS,
          )
        : POLL_INITIAL_DELAY_MS;
      schedulePolling();
    }, pollDelay);
  }

  async function loadTasks() {
    await tasksResource.run(() => listTasks({ limit: TASK_READ_LIMIT }));
  }

  async function loadJobs(kind: "manual" | "poll" = "manual") {
    lastJobRead = kind;
    await jobsResource.run(() =>
      listJobs({ kind: ACTION_JOB_KINDS, limit: JOB_READ_LIMIT }),
    );
  }

  async function refreshJobs() {
    await loadJobs();
    schedulePolling();
  }

  async function retryPolling() {
    await loadJobs("poll");
    schedulePolling();
  }

  function failureMessage(cause: unknown): string {
    return cause instanceof Error ? cause.message : String(cause);
  }

  async function addTask() {
    if (!title.trim() || saving) return;
    saving = true;
    createError = null;
    taskReceipt = null;
    try {
      const task = await createTask({
        title,
        notes: notes.trim() || undefined,
        due_date: dueDate || undefined,
        priority: Number(priority) || 0,
      });
      if (tasksResource.loading) tasksResource.invalidate();
      tasksResource.mutate((previous) => [
        task,
        ...(previous ?? []).filter((row) => row.id !== task.id),
      ]);
      taskReceipt = `${task.title} added.`;
      title = "";
      notes = "";
      priority = "0";
      if (!tasksResource.ready) await loadTasks();
    } catch (cause) {
      createError = failureMessage(cause);
    } finally {
      saving = false;
    }
  }

  async function finishTask(task: Task) {
    if (finishError?.id === task.id) finishError = null;
    try {
      const finished = await updateTask(task.id, "completed");
      if (tasksResource.loading) tasksResource.invalidate();
      tasksResource.mutate((previous) => [
        finished,
        ...(previous ?? []).filter((row) => row.id !== finished.id),
      ]);
    } catch (cause) {
      finishError = {
        id: task.id,
        message: `Could not complete ${task.title}: ${failureMessage(cause)}`,
      };
    }
  }

  async function runAction(
    action:
      "media-sync-anilist" | "media-sync-bangumi" | "media-bridge-refresh",
  ) {
    if (
      actionPending ||
      !jobsResource.ready ||
      jobsResource.error ||
      activeActionKinds.has(action.replaceAll("-", "_"))
    )
      return;
    actionError = null;
    actionPending = true;
    try {
      const job = await triggerAction(action);
      if (jobsResource.loading) jobsResource.invalidate();
      jobsResource.mutate((previous) => [
        job,
        ...(previous ?? []).filter((item) => item.id !== job.id),
      ]);
      schedulePolling();
    } catch (cause) {
      actionError = failureMessage(cause);
    } finally {
      actionPending = false;
    }
  }

  function actionLabel(action: string): string {
    if (action === "media_sync_anilist") return "AniList sync";
    if (action === "media_sync_bangumi") return "Bangumi sync";
    if (action === "media_bridge_refresh") return "Media bridge refresh";
    return action.replaceAll("_", " ");
  }

  function statusLabel(status: string): string {
    return status.replaceAll("_", " ");
  }

  function actionKind(action: string): string {
    return action.replaceAll("-", "_");
  }

  function actionIsActive(action: string): boolean {
    return activeActionKinds.has(actionKind(action));
  }
</script>

<svelte:head>
  <title>To-go · iroha</title>
</svelte:head>

<section class="admin-shell">
  <header class="admin-head">
    <div>
      <p class="eyebrow"><ListTodo size={14} /> Personal control room</p>
      <h1>What should happen next?</h1>
      <p class="intro">
        Keep today’s small intentions close, and start background work without
        leaving the cockpit.
      </p>
    </div>
    <span class="version-note">iroha v{APP_VERSION}</span>
  </header>

  <div class="admin-grid">
    <section
      class="panel tasks-panel"
      aria-labelledby="tasks-title"
      tabindex="-1"
      bind:this={tasksTarget}
    >
      <header class="panel-head">
        <div>
          <p class="eyebrow">Today</p>
          <h2 id="tasks-title">To-go list</h2>
        </div>
        <span class="count"
          >{tasksResource.ready
            ? `${openTasks.length} open in loaded tasks`
            : "Task count unavailable"}</span
        >
      </header>

      {#if tasksResource.error}
        <RetryNotice
          message={`Tasks unavailable. ${tasksResource.error}`}
          retryLabel="Retry tasks"
          onRetry={loadTasks}
          focusTarget={tasksTarget}
        />
      {/if}
      {#if createError}<p class="error" role="alert">
          Could not add task: {createError}
        </p>{/if}
      {#if finishError}<p class="error" role="alert">
          {finishError.message}
        </p>{/if}
      {#if taskReceipt}<p role="status">{taskReceipt}</p>{/if}
      <p class="panel-copy">
        Read limit: {TASK_READ_LIMIT} tasks; locally confirmed changes included. Completed
        today shows up to {COMPLETED_TASK_DISPLAY_LIMIT} from this inventory.
      </p>
      <form
        class="task-form"
        onsubmit={(event) => {
          event.preventDefault();
          void addTask();
        }}
      >
        <input
          bind:value={title}
          disabled={saving}
          placeholder="Add a small task…"
          aria-label="New task"
        />
        <input
          bind:value={dueDate}
          disabled={saving}
          type="date"
          aria-label="Task due date"
        />
        <input
          bind:value={priority}
          disabled={saving}
          type="number"
          min="0"
          max="9"
          aria-label="Task priority"
          title="Priority, 0–9"
        />
        <button type="submit" disabled={saving || !title.trim()}>Add</button>
        <textarea
          bind:value={notes}
          disabled={saving}
          placeholder="Context or next step (optional)…"
          aria-label="Task notes"
          rows="2"></textarea>
      </form>

      <LoadingBoundary
        resource={tasksResource}
        preserveLayout
        label="Loading tasks…"
      >
        {#if tasksResource.ready}
          {#if openTasks.length}
            <ul class="task-list">
              {#each openTasks as task (task.id)}
                <li>
                  <button
                    class="check-task"
                    type="button"
                    aria-label={`Complete ${task.title}`}
                    onclick={() => finishTask(task)}
                  >
                    <Check size={15} />
                  </button>
                  <span class="task-copy">
                    <strong>{task.title}</strong>
                    {#if task.notes}<p>{task.notes}</p>{/if}
                    <small>
                      {task.due_date === today
                        ? "Today"
                        : (task.due_date ?? "No due date")}
                      · p{task.priority} · {task.source} · added {formatDate(
                        task.created_at,
                      )}
                    </small>
                  </span>
                </li>
              {/each}
            </ul>
          {:else}
            <p class="empty">Nothing pressing. Add one thing worth carrying.</p>
          {/if}

          {#if completedTasks.length}
            <div class="completed-block">
              <p class="eyebrow">Completed today</p>
              {#each completedTasks as task (task.id)}
                <span>{task.title} · p{task.priority}</span>
              {/each}
            </div>
          {/if}
        {/if}
      </LoadingBoundary>
    </section>

    <section class="panel actions-panel" aria-labelledby="actions-title">
      <header class="panel-head">
        <div>
          <p class="eyebrow">Actions</p>
          <h2 id="actions-title">Start background work</h2>
        </div>
        <Play size={17} class="panel-icon" />
      </header>
      <p class="panel-copy">
        These use the durable worker queue and can be safely followed below.
      </p>
      {#if actionError}<p class="error" role="alert">
          Could not start action: {actionError}
        </p>{/if}
      {#if !jobsResource.ready || jobsResource.error}<p class="panel-copy">
          Recover the job status read before starting an action.
        </p>{/if}
      <div class="action-list">
        <button
          type="button"
          disabled={actionPending ||
            !jobsResource.ready ||
            !!jobsResource.error ||
            actionIsActive("media-sync-anilist")}
          onclick={() => runAction("media-sync-anilist")}
        >
          <span
            ><strong>AniList</strong><small
              >{actionIsActive("media-sync-anilist")
                ? "Sync already running"
                : "Refresh anime and manga"}</small
            ></span
          ><Play size={15} />
        </button>
        <button
          type="button"
          disabled={actionPending ||
            !jobsResource.ready ||
            !!jobsResource.error ||
            actionIsActive("media-sync-bangumi")}
          onclick={() => runAction("media-sync-bangumi")}
        >
          <span
            ><strong>Bangumi</strong><small
              >{actionIsActive("media-sync-bangumi")
                ? "Sync already running"
                : "Refresh the Chinese catalog"}</small
            ></span
          ><Play size={15} />
        </button>
        <button
          type="button"
          disabled={actionPending ||
            !jobsResource.ready ||
            !!jobsResource.error ||
            actionIsActive("media-bridge-refresh")}
          onclick={() => runAction("media-bridge-refresh")}
        >
          <span
            ><strong>Media bridge</strong><small
              >{actionIsActive("media-bridge-refresh")
                ? "Refresh already running"
                : "Re-fetch the Bangumi/AniList crosswalk"}</small
            ></span
          ><Play size={15} />
        </button>
      </div>
    </section>
  </div>

  <section
    class="panel queue-panel"
    aria-labelledby="queue-title"
    tabindex="-1"
    bind:this={jobsTarget}
  >
    <header class="panel-head">
      <div>
        <p class="eyebrow">System work</p>
        <h2 id="queue-title">Recent syncs</h2>
      </div>
      <button
        class="refresh"
        type="button"
        onclick={() => void refreshJobs()}
        aria-label="Refresh jobs"><RefreshCw size={15} /></button
      >
    </header>
    <p class="panel-copy queue-note">
      Only the actions above are shown here. Their importer jobs stay out of
      this personal control room.
    </p>
    {#if jobsError}
      <RetryNotice
        message={`Jobs unavailable. ${jobsError}`}
        retryLabel="Retry jobs"
        onRetry={refreshJobs}
        focusTarget={jobsTarget}
      />
    {/if}
    {#if pollError}
      <RetryNotice
        message={`Job polling failed. Last observed jobs remain available. ${pollError}`}
        retryLabel="Retry job polling"
        onRetry={retryPolling}
        focusTarget={jobsTarget}
      />
    {/if}
    <LoadingBoundary
      resource={jobsResource}
      preserveLayout
      label="Loading jobs…"
    >
      {#if jobsResource.ready}
        {#if jobs.length}
          <div class="job-list">
            {#each jobs as job (job.id)}
              <div class="job-row">
                <span class={`job-dot ${job.status}`} aria-hidden="true"></span>
                <span class="job-copy"
                  ><strong>{actionLabel(job.kind)}</strong><small
                    >{job.status === "failed" && job.error_message
                      ? job.error_message
                      : `${statusLabel(job.status)} · ${job.attempts}/${job.max_attempts} attempts`}</small
                  ></span
                >
                <time datetime={job.created_at}
                  >{formatDate(job.created_at)}</time
                >
              </div>
            {/each}
          </div>
        {:else}
          <p class="empty">No jobs have run yet.</p>
        {/if}
      {/if}
    </LoadingBoundary>
  </section>
</section>

<style>
  .admin-shell {
    display: grid;
    gap: 1.25rem;
  }
  h1,
  h2,
  p {
    margin: 0;
  }
  h1 {
    max-width: 12ch;
    font-size: clamp(2rem, 5vw, 3.4rem);
    letter-spacing: -0.09em;
    line-height: 0.9;
  }
  h2 {
    font-size: 1.45rem;
    letter-spacing: -0.04em;
  }
  .admin-head {
    display: flex;
    justify-content: space-between;
    align-items: end;
    gap: 2rem;
    padding-bottom: 1.5rem;
    border-bottom: 1px solid var(--border);
  }
  .eyebrow {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    color: var(--accent);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .intro,
  .panel-copy,
  .muted,
  .empty {
    color: var(--text-muted);
    line-height: 1.5;
  }
  .intro {
    max-width: 38rem;
    margin-top: 0.8rem;
  }
  .version-note {
    color: var(--text-muted);
    font: 0.7rem var(--font-mono);
    white-space: nowrap;
  }
  .error {
    color: var(--danger);
  }
  .admin-grid {
    display: grid;
    grid-template-columns: 1.25fr 0.75fr;
    gap: 1rem;
  }
  .panel {
    min-width: 0;
    padding: 1.25rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--tile-surface);
    box-shadow: var(--tile-shadow);
  }
  .panel-head {
    display: flex;
    justify-content: space-between;
    align-items: start;
    gap: 1rem;
    padding-bottom: 0.9rem;
    border-bottom: 1px solid var(--border);
  }
  .count {
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .task-form {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto 5rem auto;
    gap: 0.45rem;
    margin: 1rem 0;
  }
  input,
  textarea,
  .task-form button {
    min-height: 2.45rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface);
    color: var(--text);
    font: inherit;
  }
  input {
    min-width: 0;
    padding: 0 0.7rem;
  }
  textarea {
    grid-column: 1 / -1;
    width: 100%;
    padding: 0.55rem 0.7rem;
    resize: vertical;
  }
  .task-form button,
  .action-list button {
    padding: 0 0.9rem;
    cursor: pointer;
  }
  button:hover {
    border-color: var(--accent);
    color: var(--accent);
  }
  button:disabled {
    cursor: default;
    opacity: 0.5;
  }
  .task-list,
  .job-list {
    display: grid;
    gap: 0.45rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .task-list li,
  .job-row {
    display: flex;
    align-items: center;
    gap: 0.7rem;
    min-width: 0;
    padding: 0.65rem 0;
    border-bottom: 1px solid color-mix(in srgb, var(--border) 70%, transparent);
  }
  .check-task,
  .refresh {
    display: grid;
    flex: 0 0 auto;
    width: 2rem;
    height: 2rem;
    place-items: center;
    border: 1px solid var(--border);
    border-radius: 50%;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
  }
  .task-copy,
  .job-copy {
    display: grid;
    gap: 0.2rem;
    min-width: 0;
  }
  .task-copy strong,
  .job-copy strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .task-copy small,
  .task-copy p,
  .job-copy small,
  .job-row time {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .task-copy p {
    margin: 0;
    overflow: hidden;
    color: var(--text-muted);
    font-size: 0.82rem;
    line-height: 1.35;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .completed-block {
    display: grid;
    gap: 0.35rem;
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px solid var(--border);
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  .panel-copy {
    margin: 1rem 0;
    font-size: 0.85rem;
  }
  .action-list {
    display: grid;
    gap: 0.5rem;
  }
  .action-list button {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    min-height: 3.8rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 2px);
    background: transparent;
    color: var(--text);
    text-align: left;
  }
  .action-list span {
    display: grid;
    gap: 0.2rem;
  }
  .action-list small {
    color: var(--text-muted);
  }
  .panel-icon {
    color: var(--accent);
  }
  .queue-panel {
    display: grid;
    gap: 1rem;
  }
  .queue-note {
    margin: -0.35rem 0 0;
  }
  .job-dot {
    flex: 0 0 auto;
    width: 0.55rem;
    height: 0.55rem;
    border-radius: 50%;
    background: var(--text-muted);
  }
  .job-dot.queued {
    background: var(--accent-2);
  }
  .job-dot.running {
    background: var(--accent);
    box-shadow: 0 0 0.5rem var(--accent);
  }
  .job-dot.completed {
    background: var(--color-positive);
  }
  .job-dot.failed {
    background: var(--danger);
  }
  .job-row time {
    margin-left: auto;
    white-space: nowrap;
  }
  @media (max-width: 768px) {
    .admin-head {
      align-items: start;
      flex-direction: column;
    }
    .admin-grid {
      grid-template-columns: 1fr;
    }
    .task-form {
      grid-template-columns: 1fr 1fr;
    }
    .task-form input:first-child {
      grid-column: 1 / -1;
    }
    .job-row time {
      display: none;
    }
  }
</style>
