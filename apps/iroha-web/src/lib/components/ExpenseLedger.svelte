<script lang="ts">
  import { Trash2, WalletCards } from "@lucide/svelte";
  import type { Expense } from "$lib/api";

  let {
    expenses,
    selected,
    selectedId,
    detailLoading,
    onSelect,
    onRemove,
    formatMoney,
  }: {
    expenses: Expense[];
    selected: Expense | null;
    selectedId: string;
    detailLoading: boolean;
    onSelect: (id: string) => void;
    onRemove: (expense: Expense) => void;
    formatMoney: (
      amountMinor: number,
      currency: string,
      exponent: number,
    ) => string;
  } = $props();
</script>

<div class="ledger-grid">
  <section class="panel list-panel" aria-labelledby="expense-list-title">
    <header class="panel-head">
      <div>
        <p class="eyebrow">Stored records</p>
        <h2 id="expense-list-title">Expense ledger</h2>
      </div>
      <span class="count">{expenses.length} shown</span>
    </header>
    {#if expenses.length}
      <ul class="expense-list">
        {#each expenses as expense (expense.id)}
          <li>
            <button
              class:chosen={expense.id === selectedId}
              class="expense-row"
              type="button"
              onclick={() => onSelect(expense.id)}
            >
              <span>
                <strong>{expense.merchant || expense.category}</strong>
                <small>{expense.occurred_on} · {expense.category}</small>
              </span>
              <b
                >{formatMoney(
                  expense.amount_minor,
                  expense.currency,
                  expense.currency_exponent,
                )}</b
              >
            </button>
          </li>
        {/each}
      </ul>
    {:else}
      <p class="empty">No canonical expenses match these filters.</p>
    {/if}
  </section>

  <section class="panel detail-panel" aria-labelledby="expense-detail-title">
    {#if detailLoading}
      <p class="muted">Loading record…</p>
    {:else if selected}
      <header class="panel-head">
        <div>
          <p class="eyebrow">Canonical record · viewer</p>
          <h2 id="expense-detail-title">
            {selected.merchant || selected.category}
          </h2>
        </div>
        <button
          class="danger"
          type="button"
          onclick={() => onRemove(selected!)}
        >
          <Trash2 size={14} /> Delete
        </button>
      </header>

      <dl class="detail-list">
        <div>
          <dt>Amount</dt>
          <dd>
            {formatMoney(
              selected.amount_minor,
              selected.currency,
              selected.currency_exponent,
            )}
          </dd>
        </div>
        <div>
          <dt>Date</dt>
          <dd>{selected.occurred_on}</dd>
        </div>
        <div>
          <dt>Category</dt>
          <dd>{selected.category}</dd>
        </div>
        <div>
          <dt>Note</dt>
          <dd>{selected.note || "—"}</dd>
        </div>
        <div>
          <dt>Source</dt>
          <dd>{selected.source.kind} · {selected.source.ref}</dd>
        </div>
        <div>
          <dt>Record ID</dt>
          <dd class="mono">{selected.id}</dd>
        </div>
      </dl>
      {#if selected.items.length}
        <div class="item-detail">
          <h3>Items</h3>
          <ul>
            {#each selected.items as item}
              <li>
                <span>{item.name}</span>
                {#if item.amount_minor != null}<span
                    >{item.amount_minor} minor</span
                  >{/if}
              </li>
            {/each}
          </ul>
        </div>
      {/if}
      <p class="timestamps">
        Created {selected.created_at} · Updated {selected.updated_at}
      </p>
    {:else}
      <div class="empty-detail">
        <WalletCards size={22} />
        <p>Select an expense to inspect its canonical fields.</p>
      </div>
    {/if}
  </section>
</div>

<style>
  .ledger-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.15fr) minmax(18rem, 0.85fr);
    gap: 1rem;
  }
  .panel {
    min-width: 0;
    padding: 1.25rem;
    border: 1px solid var(--border);
    background: var(--surface);
  }
  .panel-head {
    display: flex;
    justify-content: space-between;
    align-items: start;
    gap: 1rem;
  }
  h2,
  h3,
  p,
  ul,
  dl {
    margin: 0;
  }
  h2 {
    font-size: 1.35rem;
    letter-spacing: -0.04em;
  }
  h3 {
    font-size: 0.9rem;
  }
  .eyebrow {
    color: var(--accent);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .count,
  .muted,
  .empty,
  .timestamps {
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .expense-list,
  .item-detail ul {
    display: grid;
    gap: 0.35rem;
    padding: 1rem 0 0;
    list-style: none;
  }
  .expense-row {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    width: 100%;
    padding: 0.8rem;
    border: 1px solid transparent;
    background: transparent;
    color: var(--text);
    text-align: left;
    cursor: pointer;
  }
  .expense-row:hover,
  .expense-row.chosen {
    border-color: var(--accent);
    background: color-mix(in srgb, var(--accent) 8%, transparent);
  }
  .expense-row span {
    display: grid;
    gap: 0.25rem;
  }
  .expense-row small {
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  .expense-row b {
    white-space: nowrap;
  }
  .danger {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    border: 1px solid var(--danger);
    padding: 0.45rem 0.6rem;
    background: transparent;
    color: var(--danger);
    cursor: pointer;
  }
  .detail-list {
    display: grid;
    gap: 0.7rem;
    padding-top: 1.25rem;
  }
  .detail-list div,
  .item-detail li {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-bottom: 1px solid var(--border);
    padding-bottom: 0.55rem;
  }
  dt {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  dd {
    margin: 0;
    text-align: right;
    overflow-wrap: anywhere;
  }
  .mono {
    font-family: var(--font-mono);
    font-size: 0.7rem;
  }
  .item-detail {
    display: grid;
    gap: 0.65rem;
    padding-top: 1.25rem;
  }
  .item-detail ul {
    padding-top: 0;
  }
  .timestamps {
    padding-top: 1.25rem;
  }
  .empty-detail {
    display: grid;
    justify-items: center;
    gap: 0.7rem;
    padding: 4rem 1rem;
    color: var(--text-muted);
    text-align: center;
  }
  @media (max-width: 760px) {
    .ledger-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
