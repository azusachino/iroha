<script lang="ts">
  import { onMount } from "svelte";
  import {
    Pencil,
    Plus,
    RefreshCw,
    Trash2,
    WalletCards,
    X,
  } from "@lucide/svelte";
  import {
    ApiError,
    deleteExpense,
    getExpense,
    listExpenses,
    updateExpense,
    type Expense,
    type ExpenseCategory,
    type ExpenseCurrency,
    type ExpenseItem,
  } from "$lib/api";
  import MonthNavigator from "@iroha/shared/MonthNavigator.svelte";
  import { currentMonth, monthBounds } from "@iroha/shared/month";
  import ThemeRouteRenderer from "$lib/themes/ThemeRouteRenderer.svelte";

  const currencies: ExpenseCurrency[] = ["JPY", "USD", "EUR", "GBP"];
  const categories: ExpenseCategory[] = [
    "food",
    "groceries",
    "transport",
    "shopping",
    "housing",
    "utilities",
    "health",
    "entertainment",
    "subscriptions",
    "work",
    "other",
  ];

  let expenses = $state<Expense[]>([]);
  let selected = $state<Expense | null>(null);
  let selectedId = $state("");
  let loading = $state(true);
  let detailLoading = $state(false);
  let editing = $state(false);
  let saving = $state(false);
  let error = $state<string | null>(null);

  let month = $state(currentMonth());
  let filterCurrency = $state("");
  let filterCategory = $state("");

  let editOccurredOn = $state("");
  let editCurrency = $state("JPY");
  let editAmountMinor = $state("");
  let editCategory = $state("other");
  let editMerchant = $state("");
  let editNote = $state("");
  let editItems = $state<ExpenseItem[]>([]);

  onMount(() => {
    void loadExpenses(month);
  });

  async function loadExpenses(selectedMonth = month) {
    loading = true;
    error = null;
    try {
      const bounds = monthBounds(selectedMonth);
      const page = await listExpenses({
        from: bounds.from,
        to: bounds.to,
        currency: (filterCurrency || undefined) as ExpenseCurrency | undefined,
        category: (filterCategory || undefined) as ExpenseCategory | undefined,
        limit: 50,
      });
      expenses = page.items;
      if (!page.items.length) {
        selected = null;
        selectedId = "";
      } else if (!page.items.some((expense) => expense.id === selectedId)) {
        await selectExpense(page.items[0].id);
      }
    } catch (cause) {
      showError(cause);
    } finally {
      loading = false;
    }
  }

  function selectMonth(value: string) {
    month = value;
    void loadExpenses(value);
  }

  async function selectExpense(id: string) {
    selectedId = id;
    detailLoading = true;
    error = null;
    try {
      selected = await getExpense(id);
    } catch (cause) {
      showError(cause);
      selected = null;
    } finally {
      detailLoading = false;
    }
  }

  function beginEdit(expense: Expense) {
    editOccurredOn = expense.occurred_on;
    editCurrency = expense.currency;
    editAmountMinor = String(expense.amount_minor);
    editCategory = expense.category;
    editMerchant = expense.merchant;
    editNote = expense.note;
    editItems = expense.items.map((item) => ({ ...item }));
    editing = true;
    error = null;
  }

  function addItem() {
    editItems = [...editItems, { name: "" }];
  }

  function removeItem(index: number) {
    editItems = editItems.filter((_, itemIndex) => itemIndex !== index);
  }

  async function saveExpense() {
    if (!selected || saving) return;
    saving = true;
    error = null;
    try {
      const updated = await updateExpense(selected.id, {
        occurred_on: editOccurredOn,
        currency: editCurrency as ExpenseCurrency,
        amount_minor: Number(editAmountMinor),
        category: editCategory as ExpenseCategory,
        merchant: editMerchant.trim(),
        note: editNote.trim(),
        items: editItems
          .filter((item) => item.name.trim())
          .map((item) => ({
            name: item.name.trim(),
            ...(item.amount_minor != null && item.amount_minor !== 0
              ? { amount_minor: Number(item.amount_minor) }
              : {}),
          })),
      });
      selected = updated;
      expenses = expenses.map((expense) =>
        expense.id === updated.id ? updated : expense,
      );
      editing = false;
    } catch (cause) {
      showError(cause);
    } finally {
      saving = false;
    }
  }

  async function removeExpense(expense: Expense) {
    if (!window.confirm(`Delete expense from ${expense.occurred_on}?`)) return;
    error = null;
    try {
      await deleteExpense(expense.id);
      const remaining = expenses.filter((item) => item.id !== expense.id);
      expenses = remaining;
      editing = false;
      if (remaining.length) {
        await selectExpense(remaining[0].id);
      } else {
        selected = null;
        selectedId = "";
      }
    } catch (cause) {
      showError(cause);
    }
  }

  function showError(cause: unknown) {
    if (cause instanceof ApiError && cause.requestId) {
      error = `${cause.message} (${cause.code}, request ${cause.requestId})`;
    } else if (cause instanceof Error) {
      error = cause.message;
    } else {
      error = String(cause);
    }
  }

  function formatAmount(expense: Expense): string {
    return new Intl.NumberFormat(undefined, {
      style: "currency",
      currency: expense.currency,
      minimumFractionDigits: expense.currency_exponent,
      maximumFractionDigits: expense.currency_exponent,
    }).format(expense.amount_minor / 10 ** expense.currency_exponent);
  }
</script>

<svelte:head>
  <title>Expenses · iroha</title>
</svelte:head>

<ThemeRouteRenderer route="expenses" props={{ route: "expenses" }}>
  {#snippet children()}
    <section class="expenses-shell">
      <header class="page-head">
        <div>
          <p class="eyebrow"><WalletCards size={14} /> Canonical ledger</p>
          <h1>Expenses</h1>
          <p class="intro">
            Review and correct the records held by iroha. Import and OCR agents
            can write here through the stable API; this page only edits
            canonical data.
          </p>
        </div>
        <button
          class="refresh"
          type="button"
          onclick={() => void loadExpenses()}
          disabled={loading}
        >
          <RefreshCw size={15} /> Refresh
        </button>
      </header>

      <div class="month-row">
        <MonthNavigator {month} onMonth={selectMonth} disabled={loading} />
      </div>

      {#if error}<p class="error" role="alert">{error}</p>{/if}

      <form
        class="filters panel"
        onsubmit={(event) => {
          event.preventDefault();
          void loadExpenses();
        }}
      >
        <label>
          Currency
          <select bind:value={filterCurrency}>
            <option value="">All currencies</option>
            {#each currencies as currency}<option value={currency}
                >{currency}</option
              >{/each}
          </select>
        </label>
        <label>
          Category
          <select bind:value={filterCategory}>
            <option value="">All categories</option>
            {#each categories as category}<option value={category}
                >{category}</option
              >{/each}
          </select>
        </label>
        <button type="submit">Apply filters</button>
      </form>

      {#if loading}
        <p class="muted">Loading expenses…</p>
      {:else}
        <div class="ledger-grid">
          <section
            class="panel list-panel"
            aria-labelledby="expense-list-title"
          >
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
                      onclick={() => void selectExpense(expense.id)}
                    >
                      <span>
                        <strong>{expense.merchant || expense.category}</strong>
                        <small>{expense.occurred_on} · {expense.category}</small
                        >
                      </span>
                      <b>{formatAmount(expense)}</b>
                    </button>
                  </li>
                {/each}
              </ul>
            {:else}
              <p class="empty">No canonical expenses match these filters.</p>
            {/if}
          </section>

          <section
            class="panel detail-panel"
            aria-labelledby="expense-detail-title"
          >
            {#if detailLoading}
              <p class="muted">Loading record…</p>
            {:else if selected}
              <header class="panel-head">
                <div>
                  <p class="eyebrow">Canonical record</p>
                  <h2 id="expense-detail-title">
                    {selected.merchant || selected.category}
                  </h2>
                </div>
                <div class="detail-actions">
                  {#if editing}
                    <button
                      class="quiet"
                      type="button"
                      onclick={() => (editing = false)}
                    >
                      <X size={14} /> Cancel
                    </button>
                  {:else}
                    <button
                      class="quiet"
                      type="button"
                      onclick={() => beginEdit(selected!)}
                    >
                      <Pencil size={14} /> Edit
                    </button>
                  {/if}
                  <button
                    class="danger"
                    type="button"
                    onclick={() => void removeExpense(selected!)}
                  >
                    <Trash2 size={14} /> Delete
                  </button>
                </div>
              </header>

              {#if editing}
                <form
                  class="edit-form"
                  onsubmit={(event) => {
                    event.preventDefault();
                    void saveExpense();
                  }}
                >
                  <label
                    >Date <input
                      bind:value={editOccurredOn}
                      type="date"
                      required
                    /></label
                  >
                  <label>
                    Currency
                    <select bind:value={editCurrency}
                      >{#each currencies as currency}<option value={currency}
                          >{currency}</option
                        >{/each}</select
                    >
                  </label>
                  <label
                    >Amount (minor units) <input
                      bind:value={editAmountMinor}
                      type="number"
                      min="1"
                      required
                    /></label
                  >
                  <label>
                    Category
                    <select bind:value={editCategory}
                      >{#each categories as category}<option value={category}
                          >{category}</option
                        >{/each}</select
                    >
                  </label>
                  <label>Merchant <input bind:value={editMerchant} /></label>
                  <label class="wide"
                    >Note <textarea bind:value={editNote} rows="3"
                    ></textarea></label
                  >
                  <div class="items wide">
                    <div class="items-head">
                      <span>Items</span><button
                        class="quiet"
                        type="button"
                        onclick={addItem}><Plus size={14} /> Add item</button
                      >
                    </div>
                    {#each editItems as item, index (index)}
                      <div class="item-row">
                        <input
                          bind:value={item.name}
                          aria-label={`Item ${index + 1} name`}
                          placeholder="Item name"
                        />
                        <input
                          bind:value={item.amount_minor}
                          aria-label={`Item ${index + 1} amount`}
                          type="number"
                          min="0"
                          placeholder="Minor amount"
                        />
                        <button
                          class="icon-button"
                          type="button"
                          aria-label={`Remove item ${index + 1}`}
                          onclick={() => removeItem(index)}
                          ><X size={14} /></button
                        >
                      </div>
                    {/each}
                  </div>
                  <button class="save wide" type="submit" disabled={saving}
                    >{saving ? "Saving…" : "Save changes"}</button
                  >
                </form>
              {:else}
                <dl class="detail-list">
                  <div>
                    <dt>Amount</dt>
                    <dd>{formatAmount(selected)}</dd>
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
                          <span>{item.name}</span
                          >{#if item.amount_minor != null}<span
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
              {/if}
            {:else}
              <div class="empty-detail">
                <WalletCards size={22} />
                <p>Select an expense to inspect its canonical fields.</p>
              </div>
            {/if}
          </section>
        </div>
      {/if}
    </section>
  {/snippet}
</ThemeRouteRenderer>

<style>
  .expenses-shell {
    display: grid;
    gap: 1.25rem;
  }
  h1,
  h2,
  h3,
  p,
  dl,
  ul {
    margin: 0;
  }
  h1 {
    font-size: clamp(2.7rem, 7vw, 5.8rem);
    letter-spacing: -0.09em;
    line-height: 0.9;
  }
  h2 {
    font-size: 1.45rem;
    letter-spacing: -0.04em;
  }
  h3 {
    font-size: 0.9rem;
  }
  .page-head,
  .panel-head,
  .items-head {
    display: flex;
    justify-content: space-between;
    align-items: start;
    gap: 1rem;
  }
  .page-head {
    align-items: end;
    padding-bottom: 1.5rem;
    border-bottom: 1px solid var(--border);
  }
  .month-row {
    display: flex;
    justify-content: center;
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
  .muted,
  .empty,
  .timestamps {
    color: var(--text-muted);
    line-height: 1.5;
  }
  .intro {
    max-width: 42rem;
    margin-top: 0.8rem;
  }
  .error {
    color: var(--danger);
  }
  .panel {
    min-width: 0;
    padding: 1.25rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--tile-surface);
    box-shadow: var(--tile-shadow);
  }
  .filters {
    display: flex;
    flex-wrap: wrap;
    align-items: end;
    gap: 0.7rem;
  }
  label {
    display: grid;
    gap: 0.3rem;
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  input,
  select,
  textarea,
  button {
    min-height: 2.4rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface);
    color: var(--text);
    font: inherit;
  }
  input,
  select,
  textarea {
    padding: 0.45rem 0.65rem;
  }
  button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.35rem;
    padding: 0 0.8rem;
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
  .refresh,
  .save {
    color: var(--accent);
  }
  .danger {
    color: var(--danger);
  }
  .quiet {
    color: var(--text-muted);
  }
  .ledger-grid {
    display: grid;
    grid-template-columns: minmax(18rem, 0.85fr) minmax(0, 1.15fr);
    gap: 1rem;
  }
  .panel-head {
    padding-bottom: 0.9rem;
    border-bottom: 1px solid var(--border);
  }
  .count {
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .expense-list {
    display: grid;
    gap: 0.35rem;
    padding: 0;
    list-style: none;
  }
  .expense-row {
    width: 100%;
    justify-content: space-between;
    margin-top: 0.35rem;
    padding: 0.75rem 0.6rem;
    text-align: left;
  }
  .expense-row.chosen {
    border-color: var(--accent);
    background: color-mix(in srgb, var(--accent) 10%, var(--surface));
  }
  .expense-row span {
    display: grid;
    gap: 0.2rem;
    min-width: 0;
  }
  .expense-row strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .expense-row small {
    color: var(--text-muted);
  }
  .expense-row b {
    white-space: nowrap;
  }
  .detail-actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: end;
    gap: 0.45rem;
  }
  .detail-list {
    display: grid;
    gap: 0.8rem;
    padding: 1.1rem 0;
  }
  .detail-list div {
    display: grid;
    grid-template-columns: 7rem 1fr;
    gap: 1rem;
    border-bottom: 1px solid color-mix(in srgb, var(--border) 70%, transparent);
    padding-bottom: 0.6rem;
  }
  dt {
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  dd {
    min-width: 0;
    overflow-wrap: anywhere;
  }
  .mono {
    font-family: var(--font-mono, monospace);
    font-size: 0.8rem;
  }
  .item-detail {
    display: grid;
    gap: 0.5rem;
    padding-top: 0.4rem;
  }
  .item-detail ul {
    display: grid;
    gap: 0.35rem;
    padding: 0;
    list-style: none;
  }
  .item-detail li {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    color: var(--text-muted);
    font-size: 0.85rem;
  }
  .timestamps {
    margin-top: 1.2rem;
    font-size: 0.72rem;
  }
  .edit-form {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.8rem;
    padding-top: 1.1rem;
  }
  .wide {
    grid-column: 1 / -1;
  }
  .items {
    display: grid;
    gap: 0.5rem;
    padding-top: 0.3rem;
  }
  .items-head {
    align-items: center;
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .item-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 9rem auto;
    gap: 0.45rem;
  }
  .icon-button {
    width: 2.4rem;
    padding: 0;
    color: var(--danger);
  }
  .empty-detail {
    display: grid;
    justify-items: center;
    gap: 0.65rem;
    padding: 4rem 1rem;
    color: var(--text-muted);
    text-align: center;
  }
  @media (max-width: 760px) {
    .page-head {
      align-items: start;
      flex-direction: column;
    }
    .ledger-grid {
      grid-template-columns: 1fr;
    }
    .filters label {
      flex: 1 1 9rem;
    }
  }
</style>
