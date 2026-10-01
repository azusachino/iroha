<script lang="ts">
  import { onMount } from "svelte";
  import { RefreshCw } from "@lucide/svelte";
  import {
    executeConnectionAction,
    getConnections,
    type Connection,
    type ConnectionAction,
  } from "$lib/api";
  import { formatDate } from "$lib/format";
  import { createAsyncResource } from "$lib/asyncResource.svelte";

  const refreshIntervalMs = 60_000;
  const resource = createAsyncResource<Connection[]>();
  const connections = $derived(resource.data ?? []);
  const loading = $derived(resource.loading);
  let actionError = $state<string | null>(null);
  const error = $derived(actionError ?? resource.error);
  let running = $state<string | null>(null);

  const attention = $derived(
    connections.filter(
      (connection) =>
        connection.operation === "failed" || connection.freshness === "overdue",
    ).length,
  );

  function status(connection: Connection): { text: string; tone: string } {
    if (connection.availability !== "supported")
      return { text: connection.availability, tone: "" };
    if (connection.operation === "failed")
      return { text: "Import failed", tone: "bad" };
    if (connection.operation === "importing")
      return { text: "Importing", tone: "" };
    if (connection.freshness === "overdue")
      return { text: "Overdue", tone: "bad" };
    if (connection.freshness === "within_cadence")
      return { text: "Within cadence", tone: "good" };
    if (connection.freshness === "not_scheduled")
      return { text: "Not scheduled", tone: "" };
    return { text: "Freshness unknown", tone: "" };
  }

  function actionLabel(action: ConnectionAction): string {
    if (action.kind === "retry_import") return "Retry import";
    if (action.kind === "sync") return "Sync now";
    return "Send data";
  }

  function executable(connection: Connection): ConnectionAction[] {
    return connection.next_actions.filter(
      (action) => action.kind === "retry_import" || action.kind === "sync",
    );
  }

  async function load() {
    actionError = null;
    await resource.run(async () => (await getConnections()).connections);
  }

  async function run(connection: Connection, action: ConnectionAction) {
    running = connection.id;
    actionError = null;
    try {
      await executeConnectionAction(action);
      await load();
    } catch (cause) {
      actionError = cause instanceof Error ? cause.message : String(cause);
    } finally {
      running = null;
    }
  }

  onMount(() => {
    void load();
    const timer = window.setInterval(() => void load(), refreshIntervalMs);
    return () => window.clearInterval(timer);
  });
</script>

<section class="admin-card" aria-labelledby="sources-title">
  <header>
    <div>
      <h2 id="sources-title">Sources</h2>
      <small role="status"
        >{connections.length} source{connections.length === 1 ? "" : "s"} ·
        {attention
          ? `${attention} need${attention === 1 ? "s" : ""} attention`
          : "no overdue or failed sources"}</small
      >
    </div>
    <button
      type="button"
      class="admin-button"
      onclick={() => void load()}
      disabled={loading}><RefreshCw size={14} /> Refresh</button
    >
  </header>

  {#if error}<p class="admin-error" role="alert">{error}</p>{/if}

  {#if connections.length}
    <ul class="admin-list">
      {#each connections as connection (connection.id)}
        {@const state = status(connection)}
        <li>
          <div>
            <strong>{connection.display_name || connection.provider}</strong>
            <span class={`status ${state.tone}`}>{state.text}</span>
            <small>
              {#if connection.coverage.length}
                {connection.collection.replaceAll("_", " ")} coverage
              {:else}
                no coverage recorded
              {/if}
              {#if connection.last_receipt}
                · last received {formatDate(
                  connection.last_receipt.received_at,
                )}
              {/if}
              {#if !connection.last_receipt}
                · never received
              {/if}
              {#if connection.freshness === "overdue"}
                · delivery overdue
              {/if}
              {#if connection.next_expected_at}
                · next expected {formatDate(connection.next_expected_at)}
              {/if}
              {#if connection.last_import}
                · last import {connection.last_import.status}
              {/if}
            </small>
            {#if connection.last_import?.error_message}
              <small class="admin-error"
                >{connection.last_import.error_message}</small
              >
            {/if}
          </div>
          <div class="admin-actions">
            {#each executable(connection) as action (action.path)}
              <button
                type="button"
                class="admin-button"
                disabled={running !== null}
                onclick={() => void run(connection, action)}
                >{running === connection.id
                  ? "Working…"
                  : actionLabel(action)}</button
              >
            {/each}
          </div>
        </li>
      {/each}
    </ul>
  {:else if !loading}
    <p class="admin-muted">No sources have delivered data yet.</p>
  {:else}
    <p class="admin-muted" role="status">Loading sources…</p>
  {/if}
</section>
