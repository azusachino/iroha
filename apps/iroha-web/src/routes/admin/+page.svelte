<script lang="ts">
  import { onMount } from "svelte";
  import {
    Activity,
    CheckCircle2,
    Database,
    RefreshCw,
    Server,
  } from "@lucide/svelte";
  import {
    getMetricCatalog,
    listJobs,
    type Job,
    type MetricDefinition,
  } from "$lib/api";
  import { APP_VERSION } from "$lib/config";
  import { formatDate } from "$lib/format";
  import { groupJobs } from "$lib/jobs";
  import { useTheme } from "$lib/themes/context.svelte";

  type HealthState = "checking" | "healthy" | "unavailable";

  let health = $state<HealthState>("checking");
  let metrics = $state<MetricDefinition[]>([]);
  let jobs = $state<Job[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);
  const theme = useTheme();

  const activeJobs = $derived(
    jobs.filter((job) => job.status === "queued" || job.status === "running"),
  );
  const failedJobs = $derived(jobs.filter((job) => job.status === "failed"));
  const executionGroups = $derived(groupJobs(jobs));
  const domains = $derived(
    [...new Set(metrics.map((metric) => metric.domain))].sort(),
  );

  async function load(): Promise<void> {
    loading = true;
    error = null;
    health = "checking";
    try {
      const [catalog, recentJobs] = await Promise.all([
        getMetricCatalog(),
        listJobs({ limit: 30 }),
      ]);
      health = "healthy";
      metrics = catalog.metrics;
      jobs = recentJobs;
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
      health = "unavailable";
      metrics = [];
      jobs = [];
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    void load();
  });
</script>

<svelte:head><title>Admin · iroha</title></svelte:head>

<section class="admin-page" data-theme={theme.definition().identity.id}>
  <header class="page-head">
    <div>
      <p class="eyebrow"><Server size={14} /> System administration</p>
      <h1>Admin</h1>
      <p class="intro">
        Read-only operational facts for the canonical cockpit: server health,
        registered metrics, and background work.
      </p>
    </div>
    <button type="button" onclick={() => void load()} disabled={loading}>
      <RefreshCw size={15} /> Refresh
    </button>
  </header>

  {#if error}<p class="error" role="alert">{error}</p>{/if}

  <section class="status-grid" aria-label="System status">
    <article class="status-card">
      <span><Server size={15} /> API health</span>
      <strong
        class:healthy={health === "healthy"}
        class:bad={health === "unavailable"}
      >
        {health === "checking"
          ? "Checking"
          : health === "healthy"
            ? "Healthy"
            : "Unavailable"}
      </strong>
      <small>API read path · iroha v{APP_VERSION}</small>
    </article>
    <article class="status-card">
      <span><Activity size={15} /> Metric catalog</span>
      <strong>{loading ? "—" : metrics.length}</strong>
      <small>{domains.length ? domains.join(" · ") : "No catalog loaded"}</small
      >
    </article>
    <article class="status-card">
      <span><Database size={15} /> Background work</span>
      <strong>{loading ? "—" : activeJobs.length}</strong>
      <small>queued or running jobs</small>
    </article>
  </section>

  <div class="admin-grid">
    <section class="panel" aria-labelledby="domains-title">
      <header>
        <div>
          <p class="eyebrow">Metric catalog</p>
          <h2 id="domains-title">Metric definitions</h2>
        </div>
        <span>{metrics.length} total</span>
      </header>
      <p class="panel-note">
        Definitions describe canonical and derived values; canonical records
        remain owned by the Iroha APIs.
      </p>
      {#if metrics.length}
        <ul class="metric-list">
          {#each metrics as metric (metric.id)}
            <li>
              <span><b>{metric.domain}</b> {metric.label}</span>
              <small>{metric.kind} · {metric.id} · {metric.unit}</small>
            </li>
          {/each}
        </ul>
      {:else if !loading}
        <p class="muted">The metric catalog could not be loaded.</p>
      {:else}
        <p class="muted">Loading the metric catalog…</p>
      {/if}
    </section>

    <section class="panel" aria-labelledby="jobs-title">
      <header>
        <div>
          <p class="eyebrow">Execution ledger</p>
          <h2 id="jobs-title">Recent jobs</h2>
        </div>
        <span>{executionGroups.length} kinds · {failedJobs.length} failed</span>
      </header>
      {#if executionGroups.length}
        <ul class="job-list">
          {#each executionGroups.slice(0, 12) as group (group.kind)}
            <li>
              <span class={`job-status ${group.latest.status}`}
                ><CheckCircle2 size={14} /> {group.latest.status}</span
              >
              <strong>{group.kind.replaceAll("_", " ")}</strong>
              <small
                >{group.count} execution{group.count === 1 ? "" : "s"} · latest
                {formatDate(group.latest.created_at)}</small
              >
            </li>
          {/each}
        </ul>
      {:else if !loading}
        <p class="muted">No background jobs are recorded.</p>
      {:else}
        <p class="muted">Loading the execution ledger…</p>
      {/if}
    </section>
  </div>
</section>

<style>
  .admin-page {
    display: grid;
    gap: 1.25rem;
  }

  .page-head,
  .panel > header {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
  }

  .page-head {
    padding-bottom: 1.5rem;
    border-bottom: 1px solid var(--border);
  }

  h1,
  h2,
  p {
    margin: 0;
  }

  h1 {
    font-size: clamp(2.7rem, 7vw, 5.8rem);
    letter-spacing: -0.09em;
    line-height: 0.9;
  }

  h2 {
    font-size: 1.25rem;
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

  .intro {
    max-width: 42rem;
    margin-top: 0.8rem;
    color: var(--text-muted);
    line-height: 1.5;
  }

  button {
    display: inline-flex;
    min-height: 2.4rem;
    align-items: center;
    gap: 0.35rem;
    padding: 0 0.8rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface);
    color: var(--text);
    font: inherit;
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

  .status-grid,
  .admin-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 0.85rem;
  }

  .admin-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .status-card,
  .panel {
    min-width: 0;
    padding: 1rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }

  .status-card {
    display: grid;
    gap: 0.45rem;
  }

  .status-card > span {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    color: var(--text-muted);
    font-size: 0.76rem;
  }

  .status-card strong {
    font-size: 1.5rem;
  }

  .status-card strong.healthy {
    color: var(--success, var(--accent));
  }

  .status-card strong.bad,
  .error {
    color: var(--danger);
  }

  small,
  .muted {
    color: var(--text-muted);
    font-size: 0.74rem;
  }

  .panel {
    display: grid;
    align-content: start;
    gap: 1rem;
  }

  .panel > header > span {
    color: var(--text-muted);
    font-size: 0.76rem;
  }

  .metric-list,
  .job-list {
    display: grid;
    max-height: 32rem;
    gap: 0.45rem;
    margin: 0;
    padding: 0;
    overflow: auto;
    list-style: none;
  }

  .metric-list li,
  .job-list li {
    display: grid;
    gap: 0.2rem;
    padding: 0.65rem 0;
    border-top: 1px solid var(--border);
  }

  .metric-list b {
    color: var(--accent);
    text-transform: capitalize;
  }

  .job-status {
    display: inline-flex;
    width: fit-content;
    align-items: center;
    gap: 0.25rem;
    color: var(--text-muted);
    font-size: 0.7rem;
    text-transform: capitalize;
  }

  .job-status.completed {
    color: var(--success, var(--accent));
  }

  .job-status.failed {
    color: var(--danger);
  }

  @media (max-width: 768px) {
    .page-head {
      align-items: start;
      flex-direction: column;
    }

    .status-grid,
    .admin-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
