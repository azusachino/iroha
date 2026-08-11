<script lang="ts">
  import { onMount } from "svelte";
  import { Check, ListTodo, Play, RefreshCw, Split } from "@lucide/svelte";
  import {
    createTask,
    listJobs,
    listMediaResolutionTasks,
    listTasks,
    triggerAction,
    updateMediaResolutionTask,
    updateTask,
    type Job,
    type MediaResolutionTask,
    type Task,
  } from "$lib/api";
  import { APP_VERSION } from "$lib/config";

  const today = new Date().toISOString().slice(0, 10);
  let openTasks = $state<Task[]>([]);
  let completedTasks = $state<Task[]>([]);
  let jobs = $state<Job[]>([]);
  let resolutionTasks = $state<MediaResolutionTask[]>([]);
  let title = $state("");
  let notes = $state("");
  let dueDate = $state(today);
  let priority = $state("0");
  let loading = $state(true);
  let saving = $state(false);
  let error = $state<string | null>(null);

  const activeJobs = $derived(
    jobs.filter((job) => job.status === "queued" || job.status === "running"),
  );
  const activeActionKinds = $derived(
    new Set(activeJobs.map((job) => job.kind)),
  );

  onMount(() => {
    void load();
    const timer = window.setInterval(() => {
      if (activeJobs.length) void loadJobs();
    }, 4000);
    return () => window.clearInterval(timer);
  });

  async function load() {
    loading = true;
    error = null;
    try {
      const [taskRows, recentJobs, openResolutionTasks] = await Promise.all([
        listTasks({ limit: 100 }),
        listJobs({
          kind: "media_sync_anilist,media_sync_bangumi",
          limit: 30,
        }),
        listMediaResolutionTasks({ status: "open" }),
      ]);
      openTasks = taskRows.filter((task) => task.status === "open");
      completedTasks = taskRows
        .filter(
          (task) => task.status === "completed" && task.due_date === today,
        )
        .slice(0, 10);
      jobs = recentJobs;
      resolutionTasks = openResolutionTasks;
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    } finally {
      loading = false;
    }
  }

  async function loadJobs() {
    try {
      jobs = await listJobs({ limit: 30 });
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    }
  }

  async function addTask() {
    if (!title.trim() || saving) return;
    saving = true;
    error = null;
    try {
      const task = await createTask({
        title,
        notes: notes.trim() || undefined,
        due_date: dueDate || undefined,
        priority: Number(priority) || 0,
      });
      openTasks = [task, ...openTasks];
      title = "";
      notes = "";
      priority = "0";
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    } finally {
      saving = false;
    }
  }

  async function finishTask(task: Task) {
    try {
      const finished = await updateTask(task.id, "completed");
      openTasks = openTasks.filter((item) => item.id !== task.id);
      completedTasks = [finished, ...completedTasks];
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    }
  }

  async function runAction(
    action: "media-sync-anilist" | "media-sync-bangumi",
  ) {
    if (activeActionKinds.has(action.replaceAll("-", "_"))) return;
    try {
      const job = await triggerAction(action);
      jobs = [job, ...jobs.filter((item) => item.id !== job.id)];
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    }
  }

  function actionLabel(action: string): string {
    if (action === "media_sync_anilist") return "AniList sync";
    if (action === "media_sync_bangumi") return "Bangumi sync";
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

  // Resolving here only records the operator's decision (resolution_json)
  // for a future job to act on -- it does not merge media rows or apply a
  // progress choice itself.
  async function resolveTask(
    task: MediaResolutionTask,
    status: "resolved" | "dismissed",
    resolution?: Record<string, unknown>,
  ) {
    try {
      await updateMediaResolutionTask(task.id, status, resolution);
      resolutionTasks = resolutionTasks.filter((item) => item.id !== task.id);
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    }
  }

  function resolutionSummary(task: MediaResolutionTask): string {
    const c = task.candidates;
    if (task.task_type === "progress_conflict") {
      return `local "${c.local_status ?? "?"}" vs remote "${c.remote_status ?? "?"}" (${c.provider ?? "?"})`;
    }
    const candidateCount = Array.isArray(c.candidates)
      ? c.candidates.length
      : 0;
    return `"${c.title ?? "untitled"}" (${c.provider ?? "?"}) — ${candidateCount} possible match${candidateCount === 1 ? "" : "es"}`;
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

  {#if error}<p class="error" role="alert">{error}</p>{/if}

  {#if loading}
    <p class="muted">Loading the control room…</p>
  {:else}
    <div class="admin-grid">
      <section class="panel tasks-panel" aria-labelledby="tasks-title">
        <header class="panel-head">
          <div>
            <p class="eyebrow">Today</p>
            <h2 id="tasks-title">To-go list</h2>
          </div>
          <span class="count">{openTasks.length} open</span>
        </header>

        <form
          class="task-form"
          onsubmit={(event) => {
            event.preventDefault();
            void addTask();
          }}
        >
          <input
            bind:value={title}
            placeholder="Add a small task…"
            aria-label="New task"
          />
          <input bind:value={dueDate} type="date" aria-label="Task due date" />
          <input
            bind:value={priority}
            type="number"
            min="0"
            max="9"
            aria-label="Task priority"
            title="Priority, 0–9"
          />
          <button type="submit" disabled={saving || !title.trim()}>Add</button>
          <textarea
            bind:value={notes}
            placeholder="Context or next step (optional)…"
            aria-label="Task notes"
            rows="2"
          ></textarea>
        </form>

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
                    · p{task.priority} · {task.source} · added {new Date(
                      task.created_at,
                    ).toLocaleDateString()}
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
        <div class="action-list">
          <button
            type="button"
            disabled={actionIsActive("media-sync-anilist")}
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
            disabled={actionIsActive("media-sync-bangumi")}
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
        </div>
      </section>
    </div>

    {#if resolutionTasks.length}
      <section
        class="panel resolution-panel"
        aria-labelledby="resolution-title"
      >
        <header class="panel-head">
          <div>
            <p class="eyebrow"><Split size={14} /> Media sync</p>
            <h2 id="resolution-title">Resolution inbox</h2>
          </div>
          <span class="count">{resolutionTasks.length} open</span>
        </header>
        <p class="panel-copy">
          Items a media sync couldn't auto-match. Resolving records your
          decision only — merging is still manual.
        </p>
        <ul class="task-list">
          {#each resolutionTasks as task (task.id)}
            <li class="resolution-row">
              <span class="task-copy">
                <strong>{resolutionSummary(task)}</strong>
                <small>
                  {task.task_type.replaceAll("_", " ")} · added {new Date(
                    task.created_at,
                  ).toLocaleDateString()}
                </small>
              </span>
              <span class="resolution-actions">
                {#if task.task_type === "progress_conflict"}
                  <button
                    type="button"
                    onclick={() =>
                      resolveTask(task, "resolved", { decision: "keep_local" })}
                    >Keep local</button
                  >
                  <button
                    type="button"
                    onclick={() =>
                      resolveTask(task, "resolved", {
                        decision: "keep_remote",
                      })}>Keep remote</button
                  >
                {:else}
                  <button
                    type="button"
                    onclick={() =>
                      resolveTask(task, "resolved", {
                        decision: "duplicate",
                        candidates: task.candidates.candidates,
                      })}>Confirm duplicate</button
                  >
                {/if}
                <button
                  type="button"
                  onclick={() => resolveTask(task, "dismissed")}>Dismiss</button
                >
              </span>
            </li>
          {/each}
        </ul>
      </section>
    {/if}

    <section class="panel queue-panel" aria-labelledby="queue-title">
      <header class="panel-head">
        <div>
          <p class="eyebrow">System work</p>
          <h2 id="queue-title">Recent syncs</h2>
        </div>
        <button
          class="refresh"
          type="button"
          onclick={loadJobs}
          aria-label="Refresh jobs"><RefreshCw size={15} /></button
        >
      </header>
      <p class="panel-copy queue-note">
        Only top-level AniList/Bangumi syncs are shown here. Their importer jobs
        stay out of this personal control room.
      </p>
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
                >{new Date(job.created_at).toLocaleString()}</time
              >
            </div>
          {/each}
        </div>
      {:else}
        <p class="empty">No jobs have run yet.</p>
      {/if}
    </section>
  {/if}
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
    font-size: clamp(2.7rem, 7vw, 5.8rem);
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
  .resolution-row {
    justify-content: space-between;
  }
  .resolution-actions {
    display: flex;
    flex: 0 0 auto;
    gap: 0.4rem;
  }
  .resolution-actions button {
    min-height: 2.1rem;
    padding: 0 0.7rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: transparent;
    color: var(--text);
    font-size: 0.75rem;
    cursor: pointer;
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
    background: var(--success, #5dbb8d);
  }
  .job-dot.failed {
    background: var(--danger);
  }
  .job-row time {
    margin-left: auto;
    white-space: nowrap;
  }
  @media (max-width: 760px) {
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
