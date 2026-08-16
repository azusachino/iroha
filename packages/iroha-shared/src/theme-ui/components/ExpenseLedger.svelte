<script lang="ts">
  import { categoryColor } from "../../domain/category-color";
  import type { Expense } from "../../domain/expense";
  import { expenseLedgerCsv } from "../../view-contracts/expense-view";
  import { formatDate } from "../../format/format";

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

  const VISIBLE_ROWS = 100;
  let visibleCount = $state(VISIBLE_ROWS);
  const visibleExpenses = $derived(expenses.slice(0, visibleCount));

  $effect(() => {
    expenses.length;
    visibleCount = VISIBLE_ROWS;
  });

  function downloadCsv() {
    const blob = new Blob(["\ufeff", expenseLedgerCsv(expenses, formatMoney)], {
      type: "text/csv;charset=utf-8",
    });
    const link = document.createElement("a");
    link.href = URL.createObjectURL(blob);
    link.download = "iroha-expenses.csv";
    link.click();
    URL.revokeObjectURL(link.href);
  }
</script>

<section
  class="canonical-ledger"
  aria-labelledby="canonical-expense-records-title"
>
  <header class="ledger-heading">
    <div>
      <p class="eyebrow">Canonical records · exact data</p>
      <h2 id="canonical-expense-records-title">Expense records</h2>
    </div>
    <p class="ledger-description">
      Aggregations above explain the month; this surface preserves every record.
    </p>
  </header>

  <div class="ledger-grid">
    <section class="panel list-panel" aria-labelledby="expense-list-title">
      <header class="panel-head">
        <div>
          <p class="eyebrow">Record index</p>
          <h3 id="expense-list-title">Browse entries</h3>
        </div>
        <div class="ledger-actions">
          <span class="count"
            >{Math.min(visibleCount, expenses.length)} of {expenses.length} shown</span
          >
          <button type="button" class="export" onclick={downloadCsv}>
            Export CSV
          </button>
        </div>
      </header>
      {#if expenses.length}
        <ul class="expense-list">
          {#each visibleExpenses as expense (expense.id)}
            <li>
              <button
                class:chosen={expense.id === selectedId}
                class="expense-row"
                type="button"
                onclick={() => onSelect(expense.id)}
              >
                <span>
                  <strong
                    ><i
                      class="category-mark"
                      style={`background: ${categoryColor(expense.category) ?? "var(--text-muted)"}`}
                    ></i>{expense.merchant || expense.category}</strong
                  >
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
        {#if visibleCount < expenses.length}
          <button
            type="button"
            class="show-more"
            onclick={() => (visibleCount += VISIBLE_ROWS)}
          >
            Show next {Math.min(VISIBLE_ROWS, expenses.length - visibleCount)}
            records
          </button>
        {/if}
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
            <p class="eyebrow">Selected record</p>
            <h3 id="expense-detail-title">
              {selected.merchant || selected.category}
            </h3>
          </div>
          <button
            class="danger"
            type="button"
            onclick={() => onRemove(selected!)}
          >
            <svg
              class="action-icon"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              aria-hidden="true"
            >
              <path
                d="M4 7h16M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7l1-3h4l1 3"
              />
            </svg>
            Delete
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
            <dt>Raw amount</dt>
            <dd>{selected.amount_minor} minor {selected.currency}</dd>
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
            <h4>Items</h4>
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
          Created {formatDate(selected.created_at)} · Updated {formatDate(
            selected.updated_at,
          )}
        </p>
      {:else}
        <div class="empty-detail">
          <svg
            class="empty-icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.7"
            aria-hidden="true"
          >
            <rect x="3" y="5" width="18" height="14" rx="2" />
            <path d="M7 9h10M7 13h6" />
          </svg>
          <p>Select an expense to inspect its canonical fields.</p>
        </div>
      {/if}
    </section>
  </div>
</section>

<style>
  .ledger-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.15fr) minmax(18rem, 0.85fr);
    gap: 0;
    border-top: 1px solid var(--border);
  }
  .canonical-ledger {
    display: grid;
    gap: 1rem;
    padding: 1.15rem;
    border: 1px solid color-mix(in srgb, var(--accent) 32%, var(--border));
    border-top: 3px solid var(--accent);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--accent) 4%, var(--surface));
  }
  .ledger-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: end;
  }
  .ledger-heading h2 {
    margin: 0.25rem 0 0;
    font-size: clamp(1.5rem, 3vw, 2.2rem);
    letter-spacing: -0.06em;
  }
  .ledger-description {
    max-width: 22rem;
    margin: 0;
    color: var(--text-muted);
    font-size: 0.78rem;
    line-height: 1.5;
    text-align: right;
  }
  .panel {
    min-width: 0;
    padding: 1rem 0;
    background: transparent;
  }
  .list-panel {
    padding-right: 1rem;
  }
  .detail-panel {
    padding-left: 1rem;
    border-left: 1px solid var(--border);
  }
  .panel-head {
    display: flex;
    justify-content: space-between;
    align-items: start;
    gap: 1rem;
  }
  .ledger-actions {
    display: flex;
    align-items: center;
    gap: 0.65rem;
  }
  h2,
  h3,
  p,
  ul,
  dl {
    margin: 0;
  }
  h2,
  h3,
  h4 {
    font-size: 1.35rem;
    letter-spacing: -0.04em;
  }
  h4 {
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
  .expense-row strong {
    display: flex;
    align-items: center;
    gap: 0.45rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .category-mark {
    flex: 0 0 auto;
    width: 0.55rem;
    height: 0.55rem;
    border-radius: 50%;
  }
  .expense-row small {
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  .expense-row b {
    white-space: nowrap;
  }
  .export,
  .show-more {
    min-height: 1.9rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface-2);
    color: var(--text-muted);
    font: inherit;
    font-size: 0.68rem;
    cursor: pointer;
  }
  .export {
    padding: 0.25rem 0.55rem;
  }
  .show-more {
    width: 100%;
    margin-top: 0.65rem;
    padding: 0.45rem;
  }
  .export:hover,
  .show-more:hover {
    border-color: var(--accent);
    color: var(--accent);
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
  .action-icon {
    width: 0.9rem;
    height: 0.9rem;
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
  .empty-icon {
    width: 1.5rem;
    height: 1.5rem;
  }
  @media (max-width: 768px) {
    .ledger-heading {
      display: grid;
    }
    .ledger-description {
      text-align: left;
    }
    .ledger-grid {
      grid-template-columns: 1fr;
    }
    .list-panel {
      padding-right: 0;
    }
    .detail-panel {
      padding-left: 0;
      border-top: 1px solid var(--border);
      border-left: 0;
    }
  }
</style>
