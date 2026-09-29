<script lang="ts">
  import { page } from "$app/state";
  import { goto } from "$app/navigation";
  import { Server } from "@lucide/svelte";
  import { useTheme } from "$lib/themes/context.svelte";
  import AdminSystem from "$lib/components/admin/AdminSystem.svelte";
  import AdminSources from "$lib/components/admin/AdminSources.svelte";
  import AdminJobs from "$lib/components/admin/AdminJobs.svelte";
  import AdminImports from "$lib/components/admin/AdminImports.svelte";
  import IntakeTokens from "$lib/components/IntakeTokens.svelte";
  import { nextTabId } from "$lib/tab-keys";

  const TABS = [
    { id: "system", label: "System" },
    { id: "sources", label: "Sources" },
    { id: "jobs", label: "Jobs" },
    { id: "imports", label: "Imports" },
    { id: "intake", label: "Intake tokens" },
  ] as const;
  type TabId = (typeof TABS)[number]["id"];

  const theme = useTheme();
  const tab = $derived<TabId>(
    (TABS.find((item) => item.id === page.url.searchParams.get("tab"))?.id ??
      "system") as TabId,
  );

  function onTabKeydown(event: KeyboardEvent) {
    const next = nextTabId(
      TABS.map((item) => item.id),
      tab,
      event.key,
    );
    if (!next) return;
    event.preventDefault();
    select(next);
    document.getElementById(`admin-tab-${next}`)?.focus();
  }

  function select(id: TabId) {
    const url = new URL(page.url);
    url.searchParams.set("tab", id);
    void goto(`${url.pathname}${url.search}`, {
      replaceState: true,
      noScroll: true,
      keepFocus: true,
    });
  }
</script>

<svelte:head><title>Admin · iroha</title></svelte:head>

<section class="admin-page" data-theme={theme.definition().identity.id}>
  <header class="page-head">
    <div>
      <p class="eyebrow"><Server size={14} /> Operations</p>
      <h1>Admin</h1>
      <p class="intro">
        System state, data sources, background jobs, imports, and intake tokens.
      </p>
    </div>
  </header>

  <div class="admin-tabs" role="tablist" aria-label="Admin sections">
    {#each TABS as item (item.id)}
      <button
        type="button"
        role="tab"
        id={`admin-tab-${item.id}`}
        aria-selected={tab === item.id}
        aria-controls="admin-panel"
        tabindex={tab === item.id ? 0 : -1}
        onkeydown={onTabKeydown}
        onclick={() => select(item.id)}>{item.label}</button
      >
    {/each}
  </div>

  <div
    id="admin-panel"
    role="tabpanel"
    aria-labelledby={`admin-tab-${tab}`}
    class="admin-panel"
  >
    {#if tab === "system"}
      <AdminSystem />
    {:else if tab === "sources"}
      <AdminSources />
    {:else if tab === "jobs"}
      <AdminJobs />
    {:else if tab === "imports"}
      <AdminImports />
    {:else}
      <IntakeTokens />
    {/if}
  </div>
</section>

<style>
  .admin-page {
    display: grid;
    gap: 1.25rem;
  }

  .page-head {
    padding-bottom: 1rem;
  }

  h1,
  p {
    margin: 0;
  }

  h1 {
    font-size: var(--grapher-utility-title-size);
    letter-spacing: -0.09em;
    line-height: 0.95;
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

  .admin-tabs {
    display: flex;
    gap: 0.25rem;
    overflow-x: auto;
    border-bottom: 1px solid var(--border);
    scrollbar-width: none;
  }

  .admin-tabs button {
    flex: none;
    min-height: 2.6rem;
    padding: 0 0.9rem;
    border: 0;
    border-bottom: 2px solid transparent;
    background: none;
    color: var(--text-muted);
    font: inherit;
    font-size: 0.9rem;
    cursor: pointer;
  }

  .admin-tabs button[aria-selected="true"] {
    border-bottom-color: var(--accent);
    color: var(--text);
    font-weight: 650;
  }

  .admin-tabs button:focus-visible {
    outline: 2px solid var(--color-focus);
    outline-offset: -2px;
  }

  /* Shared admin building blocks, used by every tab component. */
  .admin-panel :global(.admin-card) {
    display: grid;
    min-width: 0;
    align-content: start;
    gap: 0.9rem;
    padding: 1rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }

  .admin-panel :global(.admin-card > header) {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
  }

  .admin-panel :global(.admin-card h2) {
    margin: 0;
    font-size: 1.15rem;
  }

  .admin-panel :global(.admin-muted),
  .admin-panel :global(.admin-card small) {
    color: var(--text-muted);
    font-size: 0.78rem;
    line-height: 1.45;
  }

  .admin-panel :global(.admin-list) {
    display: grid;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .admin-panel :global(.admin-list > li) {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem 1rem;
    padding: 0.7rem 0;
    border-top: 1px solid var(--border);
  }

  .admin-panel :global(.admin-list > li > div:first-child) {
    display: grid;
    min-width: 0;
    gap: 0.15rem;
  }

  .admin-panel :global(.admin-actions) {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
  }

  .admin-panel :global(.admin-button) {
    display: inline-flex;
    min-height: 2.3rem;
    align-items: center;
    gap: 0.35rem;
    padding: 0 0.75rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface);
    color: var(--text);
    font: inherit;
    font-size: 0.85rem;
    cursor: pointer;
  }

  .admin-panel :global(.admin-button:hover) {
    border-color: var(--accent);
    color: var(--accent);
  }

  .admin-panel :global(.admin-button.primary) {
    border-color: var(--text);
    background: var(--text);
    color: var(--surface);
    font-weight: 650;
  }

  .admin-panel :global(.admin-button:disabled) {
    cursor: default;
    opacity: 0.5;
  }

  .admin-panel :global(.admin-button:focus-visible) {
    outline: 2px solid var(--color-focus);
    outline-offset: 2px;
  }

  .admin-panel :global(.status) {
    display: inline-flex;
    width: fit-content;
    align-items: center;
    gap: 0.3rem;
    padding: 0.1rem 0.5rem;
    border-radius: 999px;
    background: color-mix(in srgb, var(--text-muted) 12%, transparent);
    color: var(--text-muted);
    font-size: 0.72rem;
    font-weight: 650;
    text-transform: capitalize;
  }

  .admin-panel :global(.status.good) {
    background: color-mix(in srgb, var(--color-positive) 14%, transparent);
    color: var(--color-positive);
  }

  .admin-panel :global(.status.bad) {
    background: color-mix(in srgb, var(--danger) 12%, transparent);
    color: var(--danger);
  }

  .admin-panel :global(.admin-error) {
    margin: 0;
    color: var(--danger);
    font-size: 0.85rem;
  }

  .admin-panel :global(.admin-grid) {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.85rem;
  }

  @media (max-width: 768px) {
    .admin-panel :global(.admin-grid) {
      grid-template-columns: 1fr;
    }
  }
</style>
