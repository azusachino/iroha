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

  const today = new Date().toISOString().slice(0, 10);
  let openTasks = $state<Task[]>([]);
  let completedTasks = $state<Task[]>([]);
  let jobs = $state<Job[]>([]);
  let title = $state("");
  let dueDate = $state(today);
  let loading = $state(true);
  let saving = $state(false);
  let error = $state<string | null>(null);

  const activeJobs = $derived(
    jobs.filter((job) => job.status === "queued" || job.status === "running"),
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
      const [open, completed, recentJobs] = await Promise.all([
        listTasks({ status: "open", limit: 50 }),
        listTasks({ status: "completed", due: today, limit: 10 }),
        listJobs({ limit: 30 }),
      ]);
      openTasks = open;
      completedTasks = completed;
      jobs = recentJobs;
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
      const task = await createTask({ title, due_date: dueDate || undefined });
      openTasks = [task, ...openTasks];
      title = "";
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
          <button type="submit" disabled={saving || !title.trim()}>Add</button>
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
                  <small
                    >{task.due_date === today
                      ? "Today"
                      : (task.due_date ?? "No due date")}</small
                  >
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
            {#each completedTasks as task (task.id)}<span>{task.title}</span
              >{/each}
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
          <button type="button" onclick={() => runAction("media-sync-anilist")}>
            <span
              ><strong>AniList</strong><small>Refresh anime and manga</small
              ></span
            ><Play size={15} />
          </button>
          <button type="button" onclick={() => runAction("media-sync-bangumi")}>
            <span
              ><strong>Bangumi</strong><small>Refresh the Chinese catalog</small
              ></span
            ><Play size={15} />
          </button>
        </div>
      </section>
    </div>

    <section class="panel queue-panel" aria-labelledby="queue-title">
      <header class="panel-head">
        <div>
          <p class="eyebrow">System work</p>
          <h2 id="queue-title">Recent jobs</h2>
        </div>
        <button
          class="refresh"
          type="button"
          onclick={loadJobs}
          aria-label="Refresh jobs"><RefreshCw size={15} /></button
        >
      </header>
      {#if jobs.length}
        <div class="job-list">
          {#each jobs as job (job.id)}
            <div class="job-row">
              <span class={`job-dot ${job.status}`} aria-hidden="true"></span>
              <span class="job-copy"
                ><strong>{actionLabel(job.kind)}</strong><small
                  >{job.status === "failed" && job.error_message
                    ? job.error_message
                    : statusLabel(job.status)}</small
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
    grid-template-columns: 1fr auto auto;
    gap: 0.45rem;
    margin: 1rem 0;
  }
  input,
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
  .job-copy small,
  .job-row time {
    color: var(--text-muted);
    font-size: 0.72rem;
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
