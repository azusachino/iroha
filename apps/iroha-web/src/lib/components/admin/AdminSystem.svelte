<script lang="ts">
  import { onMount } from "svelte";
  import { RefreshCw } from "@lucide/svelte";
  import {
    getMetricCatalog,
    getSystemInfo,
    type MetricDefinition,
    type SystemInfo,
  } from "$lib/api";
  import { APP_VERSION } from "$lib/config";
  import { formatDate } from "$lib/format";
  import { formatBytes } from "$lib/format-bytes";

  let system = $state<SystemInfo | null>(null);
  let metrics = $state<MetricDefinition[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  async function load() {
    loading = true;
    error = null;
    try {
      const [info, catalog] = await Promise.all([
        getSystemInfo(),
        getMetricCatalog(),
      ]);
      system = info;
      metrics = catalog.metrics;
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    void load();
  });
</script>

<div class="admin-grid">
  <section class="admin-card" aria-labelledby="system-title">
    <header>
      <div>
        <h2 id="system-title">System</h2>
        <small>What this deployment is running.</small>
      </div>
      <button
        type="button"
        class="admin-button"
        onclick={() => void load()}
        disabled={loading}><RefreshCw size={14} /> Refresh</button
      >
    </header>
    {#if error}
      <p class="admin-error" role="alert">API unavailable: {error}</p>
    {:else if system}
      <dl class="facts">
        <dt>API</dt>
        <dd><span class="status good">Healthy</span></dd>
        <dt>Web app</dt>
        <dd>v{APP_VERSION}</dd>
        <dt>Parser</dt>
        <dd><code>{system.parser_version}</code></dd>
        <dt>Schema</dt>
        <dd>migration {system.migration_version}</dd>
        <dt>Database</dt>
        <dd>{formatBytes(system.database_bytes)}</dd>
        <dt>Raw evidence</dt>
        <dd>
          {system.raw_file_count} files · {formatBytes(system.raw_file_bytes)}
        </dd>
        <dt>Cache</dt>
        <dd>{system.cache_backend}</dd>
        <dt>Timezone</dt>
        <dd>{system.timezone}</dd>
        <dt>Passkeys</dt>
        <dd>{system.passkeys_enabled ? "Enabled" : "Not configured"}</dd>
        <dt>Server time</dt>
        <dd>{formatDate(system.server_time)}</dd>
      </dl>
    {:else}
      <p class="admin-muted" role="status">Loading…</p>
    {/if}
  </section>

  <section class="admin-card" aria-labelledby="catalog-title">
    <header>
      <div>
        <h2 id="catalog-title">Metric catalog</h2>
        <small>Canonical and derived values the cockpit reads.</small>
      </div>
      <small>{metrics.length} total</small>
    </header>
    {#if metrics.length}
      <ul class="admin-list catalog">
        {#each metrics as metric (metric.id)}
          <li>
            <div>
              <strong>{metric.label}</strong>
              <small>{metric.domain} · {metric.kind} · {metric.unit}</small>
            </div>
            <code>{metric.id}</code>
          </li>
        {/each}
      </ul>
    {:else if !loading}
      <p class="admin-muted">The metric catalog could not be loaded.</p>
    {/if}
  </section>
</div>

<style>
  .facts {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 0.55rem 1rem;
    margin: 0;
    font-size: 0.88rem;
  }

  dt {
    color: var(--text-muted);
  }

  dd {
    margin: 0;
    overflow-wrap: anywhere;
  }

  .catalog {
    max-height: 30rem;
    overflow: auto;
  }

  code {
    font-size: 0.75rem;
  }
</style>
