<script lang="ts">
  import { onMount } from "svelte";
  import {
    executeConnectionAction,
    getConnections,
    type Connection,
    type ConnectionAction,
  } from "$lib/api";

  let connections = $state<Connection[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let running = $state<string | null>(null);

  const failed = $derived(
    connections.filter((connection) => connection.operation === "failed"),
  );
  const available = $derived(
    connections.filter((connection) => connection.availability === "supported")
      .length,
  );

  function label(connection: Connection): string {
    if (connection.operation === "failed") return "Import failed";
    if (connection.operation === "importing") return "Importing";
    return connection.collection.replaceAll("_", " ");
  }

  function executableAction(connection: Connection): ConnectionAction | null {
    return (
      connection.next_actions.find(
        (action) => action.kind === "retry_import" || action.kind === "sync",
      ) ?? null
    );
  }

  async function load() {
    loading = true;
    error = null;
    try {
      connections = (await getConnections()).connections;
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    } finally {
      loading = false;
    }
  }

  async function run(connection: Connection, action: ConnectionAction) {
    running = connection.id;
    error = null;
    try {
      await executeConnectionAction(action);
      await load();
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    } finally {
      running = null;
    }
  }

  onMount(() => {
    void load();
  });
</script>

{#if !loading && (connections.length > 0 || error)}
  <section class="connection-strip tile" aria-label="Source status">
    <div class="connection-summary">
      <span class="status-dot" class:bad={failed.length > 0} aria-hidden="true"
      ></span>
      <div>
        <p class="eyebrow">Sources</p>
        <strong>
          {#if failed.length > 0}
            {failed.length} source{failed.length === 1 ? "" : "s"} need attention
          {:else}
            {available} source{available === 1 ? "" : "s"} available
          {/if}
        </strong>
      </div>
    </div>
    {#if failed.length > 0}
      <div class="connection-actions">
        {#each failed.slice(0, 2) as connection (connection.id)}
          {@const action = executableAction(connection)}
          <div class="connection-action">
            <span
              >{connection.display_name || connection.provider} · {label(
                connection,
              )}</span
            >
            {#if action}
              <button
                type="button"
                onclick={() => void run(connection, action)}
                disabled={running === connection.id}
              >
                {running === connection.id
                  ? "Running…"
                  : action.kind === "retry_import"
                    ? "Retry"
                    : "Sync"}
              </button>
            {:else}
              <span class="muted">Use the source action</span>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
    {#if error}
      <p class="error" role="alert">{error}</p>
    {/if}
    <a href="/admin">Source details →</a>
  </section>
{/if}

<style>
  .connection-strip {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr) auto;
    gap: 1rem;
    align-items: center;
    padding: 0.75rem 1rem;
    border-color: color-mix(in srgb, var(--accent) 28%, var(--border));
  }

  .connection-summary {
    display: flex;
    align-items: center;
    gap: 0.65rem;
    white-space: nowrap;
  }

  .connection-summary .eyebrow {
    margin: 0 0 0.1rem;
  }

  .connection-summary strong {
    font-size: 0.85rem;
  }

  .status-dot {
    width: 0.6rem;
    height: 0.6rem;
    flex: 0 0 auto;
    border-radius: 50%;
    background: var(--success, #49b58a);
    box-shadow: 0 0 0 0.25rem
      color-mix(in srgb, var(--success, #49b58a) 15%, transparent);
  }

  .status-dot.bad {
    background: var(--danger);
    box-shadow: 0 0 0 0.25rem color-mix(in srgb, var(--danger) 15%, transparent);
  }

  .connection-actions {
    display: grid;
    gap: 0.35rem;
  }

  .connection-action {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    color: var(--text-muted);
    font-size: 0.76rem;
  }

  .connection-action button {
    border: 1px solid var(--accent);
    border-radius: 999px;
    padding: 0.25rem 0.65rem;
    background: transparent;
    color: var(--accent);
    cursor: pointer;
    font: inherit;
  }

  .connection-action button:disabled {
    cursor: wait;
    opacity: 0.6;
  }

  .connection-strip > a {
    color: var(--accent);
    font-size: 0.75rem;
    white-space: nowrap;
  }

  .muted,
  .error {
    font-size: 0.75rem;
  }

  .error {
    margin: 0;
    color: var(--danger);
  }

  @media (max-width: 640px) {
    .connection-strip {
      grid-template-columns: 1fr auto;
    }

    .connection-actions {
      grid-column: 1 / -1;
    }
  }
</style>
