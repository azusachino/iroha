<script lang="ts">
  import type { ExpenseCategory } from "$lib/api";
  import PeriodSelector from "@iroha/shared/components/PeriodSelector.svelte";
  import PeriodToolbar from "@iroha/shared/components/PeriodToolbar.svelte";
  import SelectControl from "@iroha/shared/components/SelectControl.svelte";
  import LoadingBoundary from "$lib/components/LoadingBoundary.svelte";
  import ConfirmDialog from "$lib/components/ConfirmDialog.svelte";
  import { expenseCategoryLabel } from "@iroha/shared/view-contracts/expense-view";
  import { categoryColor } from "@iroha/shared/domain/category-color";
  import GrapherExpenses from "@iroha/shared/theme-ui/grapher/Expenses.svelte";
  import RetryNotice from "@iroha/shared/theme-ui/components/RetryNotice.svelte";
  import {
    CATEGORY_OPTIONS,
    CURRENCY_OPTIONS,
    createExpensesState,
  } from "./expenses-state.svelte";

  const e = createExpensesState();
  let recordsTarget = $state<HTMLElement>();
  let spendingTarget = $state<HTMLElement>();
  let boundsTarget = $state<HTMLElement>();
</script>

<svelte:head>
  <title>iroha · Expenses</title>
</svelte:head>

<section class="expenses-shell">
  <header class="domain-header" role="region" aria-label="Expenses header">
    <div class="header-copy">
      <h1 id="expenses-title">Expenses</h1>
      <p>Spending records and currency ledgers</p>
    </div>
  </header>

  <section
    aria-label="Expense period selection"
    tabindex="-1"
    bind:this={boundsTarget}
  >
    <PeriodToolbar
      title="Expense spending scope"
      ariaLabel="Expense period"
      onRefresh={() => void e.loadExpenses(e.month, true)}
      refreshing={e.expensesResource.loading}
    >
      <PeriodSelector
        year={e.periodYear}
        month={e.periodMonth}
        years={e.periodYears}
        months={e.periodMonths}
        bounds={e.dateBounds}
        showAllYears={false}
        surface="inline"
        onYear={e.selectPeriodYear}
        onMonth={e.selectPeriodMonth}
      />
    </PeriodToolbar>

    {#if e.expensesResource.data}
      <p class="muted">Observed period: {e.expensesResource.data.month}</p>
    {:else if e.expensesResource.error}
      <p class="error" role="alert">Ledger period unavailable</p>
    {:else if e.expensesResource.loading}
      <p class="muted" role="status">Loading ledger period…</p>
    {/if}

    <div class="expense-filters" aria-label="Expense filters">
      <SelectControl
        label="Currency"
        value={e.filterCurrency}
        options={[{ value: "", label: "All currencies" }, ...CURRENCY_OPTIONS]}
        onChange={e.selectCurrency}
      />
      <SelectControl
        label="Category"
        value={e.filterCategory}
        options={CATEGORY_OPTIONS}
        markerColor={categoryColor(e.filterCategory || "other")}
        onChange={e.selectCategory}
      />
    </div>
    <p>
      Selected period: {e.month} · {e.filterCurrency || "All currencies"} · {e.filterCategory
        ? expenseCategoryLabel[e.filterCategory]
        : "All categories"}
    </p>
    {#if e.boundsResource.error}
      <RetryNotice
        message={`Expense range unavailable. ${e.boundsResource.error}`}
        retryLabel="Retry expense range"
        onRetry={e.loadBounds}
        focusTarget={boundsTarget}
      />
      <p>
        {e.dateBounds.min && e.dateBounds.max
          ? "The last observed expense range remains available."
          : "Period choices are calendar choices, not an observed data inventory."}
      </p>
    {/if}
  </section>
  {#if e.deleteError}
    <p class="error" role="alert">Delete failed: {e.deleteError}</p>
  {/if}
  <section
    aria-label="Expense spending"
    tabindex="-1"
    bind:this={spendingTarget}
  >
    {#if e.spendingResource.error}
      <RetryNotice
        message={`Expense spending unavailable. ${e.spendingResource.error}`}
        retryLabel="Retry expense spending"
        onRetry={() => e.loadSpending()}
        focusTarget={spendingTarget}
      />
    {/if}
    <LoadingBoundary
      resource={e.spendingResource}
      preserveLayout
      label="Loading expense spending…"
    >
      {#snippet children()}
        {#if e.spendingResource.data}
          <p>
            Observed spending period: {e.spendingResource.data.month} · {e
              .spendingResource.data.currency || "All currencies"} · {e
              .spendingResource.data.category
              ? expenseCategoryLabel[
                  e.spendingResource.data.category as ExpenseCategory
                ]
              : "All categories"}
          </p>
          <GrapherExpenses
            {...e.themeProps}
            month={e.spendingResource.data.month}
            section="spending"
          />
        {/if}
      {/snippet}
    </LoadingBoundary>
  </section>
  <section
    aria-label="Expense records read"
    tabindex="-1"
    bind:this={recordsTarget}
  >
    {#if e.expensesResource.error}
      <RetryNotice
        message={`Expense records unavailable. ${e.expensesResource.error}`}
        retryLabel="Retry expense records"
        onRetry={() => e.loadRecords()}
        focusTarget={recordsTarget}
      />
    {/if}
    <LoadingBoundary
      resource={e.expensesResource}
      preserveLayout
      label="Loading expense records…"
    >
      {#snippet children()}
        {#if e.expensesResource.data}
          <p>
            Observed ledger period: {e.expensesResource.data.month} · {e
              .expensesResource.data.currency || "All currencies"} · {e
              .expensesResource.data.category
              ? expenseCategoryLabel[
                  e.expensesResource.data.category as ExpenseCategory
                ]
              : "All categories"}
          </p>
          <GrapherExpenses {...e.themeProps} section="ledger" />
        {/if}
      {/snippet}
    </LoadingBoundary>
  </section>
</section>

<ConfirmDialog
  bind:open={e.confirmDeleteOpen}
  message={e.pendingDelete
    ? `Delete expense from ${e.pendingDelete.occurred_on}?`
    : ""}
  onConfirm={() => void e.confirmRemoveExpense()}
/>

<style>
  .expenses-shell {
    display: grid;
    gap: 1.25rem;
  }
  h1,
  p {
    margin: 0;
  }
  .muted {
    color: var(--text-muted);
    line-height: 1.5;
  }
  .error {
    color: var(--danger);
  }
  .domain-header {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    padding-bottom: 1.25rem;
    border-bottom: 1px solid var(--border);
  }
  .header-copy h1 {
    margin: 0;
    font-size: var(--type-title);
    line-height: 1.25;
    letter-spacing: normal;
  }
  .header-copy p {
    margin: 0.25rem 0 0;
    font-size: var(--type-caption);
    color: var(--text-muted);
  }
  .expense-filters {
    display: flex;
    flex-wrap: wrap;
    align-items: end;
    gap: 0.75rem;
    padding: 0.75rem 0;
    border-bottom: 1px solid var(--border);
  }
  .expense-filters :global(.select-control) {
    flex: 1 1 9rem;
  }
</style>
