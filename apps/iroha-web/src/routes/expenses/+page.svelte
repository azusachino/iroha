<script lang="ts">
  import { onMount } from "svelte";
  import { RefreshCw, WalletCards } from "@lucide/svelte";
  import {
    ApiError,
    deleteExpense,
    getExpense,
    getMetricSeries,
    listAllExpenses,
    type Expense,
    type ExpenseCategory,
    type ExpenseCurrency,
    type MetricSeriesResponse,
  } from "$lib/api";
  import MonthNavigator from "@iroha/shared/MonthNavigator.svelte";
  import { currentMonth, monthBounds } from "@iroha/shared/month";
  import type { ExpenseThemeProps } from "$lib/expense-view";
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
  let error = $state<string | null>(null);
  let dailySeries = $state<MetricSeriesResponse | null>(null);
  let categorySeries = $state<MetricSeriesResponse[]>([]);
  let currencySeries = $state<MetricSeriesResponse[]>([]);
  let currencyCountSeries = $state<MetricSeriesResponse[]>([]);

  let month = $state(currentMonth());
  let filterCurrency = $state("");
  let filterCategory = $state("");

  onMount(() => {
    void loadExpenses(month);
  });

  async function loadExpenses(selectedMonth = month) {
    loading = true;
    error = null;
    try {
      const bounds = monthBounds(selectedMonth);
      const monthExpenses = await listAllExpenses({
        from: bounds.from,
        to: bounds.to,
        currency: (filterCurrency || undefined) as ExpenseCurrency | undefined,
        category: (filterCategory || undefined) as ExpenseCategory | undefined,
      });
      expenses = monthExpenses;
      const chartCurrencies = filterCurrency
        ? [filterCurrency as ExpenseCurrency]
        : currencies;
      const [currenciesForMonth, countsForCurrency] = await Promise.all([
        Promise.all(
          chartCurrencies.map((currency) =>
            getMetricSeries("expenses.amount_minor", {
              from: bounds.from,
              to: bounds.to,
              grain: "month",
              timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
              dimensions: [`currency:${currency}`],
            }),
          ),
        ),
        Promise.all(
          chartCurrencies.map((currency) =>
            getMetricSeries("expenses.count", {
              from: bounds.from,
              to: bounds.to,
              grain: "month",
              timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
              dimensions: [`currency:${currency}`],
            }),
          ),
        ),
      ]);
      const chartCurrency =
        filterCurrency ||
        currenciesForMonth.find((series) => seriesPointValue(series) != null)
          ?.series[0]?.dimensions.currency ||
        "JPY";
      const chartCategories = filterCategory
        ? [filterCategory as ExpenseCategory]
        : categories;
      const [daily, categoriesForCurrency] = await Promise.all([
        getMetricSeries("expenses.amount_minor", {
          from: bounds.from,
          to: bounds.to,
          grain: "day",
          timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
          dimensions: [`currency:${chartCurrency}`],
        }),
        Promise.all(
          chartCategories.map((category) =>
            getMetricSeries("expenses.amount_minor", {
              from: bounds.from,
              to: bounds.to,
              grain: "month",
              timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
              dimensions: [`currency:${chartCurrency}`, `category:${category}`],
            }),
          ),
        ),
      ]);
      dailySeries = daily;
      categorySeries = categoriesForCurrency;
      currencySeries = currenciesForMonth;
      currencyCountSeries = countsForCurrency;
      if (!monthExpenses.length) {
        selected = null;
        selectedId = "";
      } else if (!monthExpenses.some((expense) => expense.id === selectedId)) {
        await selectExpense(monthExpenses[0].id);
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

  async function removeExpense(expense: Expense) {
    if (!window.confirm(`Delete expense from ${expense.occurred_on}?`)) return;
    error = null;
    try {
      await deleteExpense(expense.id);
      await loadExpenses();
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

  function formatMoney(
    amountMinor: number,
    currency: string,
    exponent: number,
  ): string {
    return new Intl.NumberFormat(undefined, {
      style: "currency",
      currency,
      minimumFractionDigits: exponent,
      maximumFractionDigits: exponent,
    }).format(amountMinor / 10 ** exponent);
  }

  function seriesPointValue(
    series: MetricSeriesResponse | null,
  ): number | null {
    const point = series?.series[0]?.points[0];
    return point?.value_minor ?? null;
  }

  function numericSeriesPointValue(
    series: MetricSeriesResponse | null,
  ): number | null {
    const point = series?.series[0]?.points[0];
    return point && "value" in point ? (point.value ?? null) : null;
  }

  const currencyTotals = $derived(
    currencySeries
      .map((series) => ({
        currency: series.series[0]?.dimensions.currency as ExpenseCurrency,
        amountMinor: seriesPointValue(series) ?? 0,
        exponent: series.series[0]?.dimensions.currency === "JPY" ? 0 : 2,
        count:
          numericSeriesPointValue(
            currencyCountSeries.find(
              (countSeries) =>
                countSeries.series[0]?.dimensions.currency ===
                series.series[0]?.dimensions.currency,
            ) ?? null,
          ) ?? 0,
      }))
      .filter((item) => item.currency)
      .sort((a, b) => b.amountMinor - a.amountMinor),
  );
  const primaryCurrency = $derived(
    (filterCurrency ||
      currencyTotals.find((item) => item.amountMinor !== 0)?.currency ||
      currencyTotals[0]?.currency ||
      "JPY") as ExpenseCurrency,
  );
  const primaryExponent = $derived(
    expenses.find((item) => item.currency === primaryCurrency)
      ?.currency_exponent ?? (primaryCurrency === "JPY" ? 0 : 2),
  );
  const categoryTotals = $derived(
    categorySeries
      .map((series) => ({
        category: series.series[0]?.dimensions.category ?? "",
        amount: seriesPointValue(series) ?? 0,
      }))
      .filter((item) => item.category && item.amount > 0)
      .sort((a, b) => b.amount - a.amount),
  );
  const dailyTotals = $derived(
    (dailySeries?.series[0]?.points ?? []).map(
      (point) =>
        [point.period, point.value_minor ?? null] as [string, number | null],
    ),
  );

  const themeProps = $derived<ExpenseThemeProps>({
    month,
    primaryCurrency,
    primaryExponent,
    currencyTotals,
    categoryTotals,
    dailyTotals,
    expenses,
    selected,
    selectedId,
    detailLoading,
    onSelect: (id) => void selectExpense(id),
    onRemove: (expense) => void removeExpense(expense),
    formatMoney,
  });
</script>

<svelte:head>
  <title>Expenses · iroha</title>
</svelte:head>

<section class="expenses-shell">
  <header class="page-head">
    <div>
      <p class="eyebrow"><WalletCards size={14} /> Canonical ledger</p>
      <h1>Expenses</h1>
      <p class="intro">
        Read server-computed spending series first; open canonical records only
        when you need the source detail.
      </p>
    </div>
    <button
      class="refresh"
      type="button"
      onclick={() => void loadExpenses()}
      disabled={loading}><RefreshCw size={15} /> Refresh</button
    >
  </header>
  <section class="period panel" aria-label="Expense period">
    <div class="period-copy">
      <span>Period</span>
      <strong>Monthly ledger scope</strong>
    </div>
    <MonthNavigator {month} onMonth={selectMonth} disabled={loading} />
  </section>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  <div class="filters panel" aria-label="Expense filters">
    <label
      >Currency<select
        value={filterCurrency}
        onchange={(event) => {
          filterCurrency = (event.currentTarget as HTMLSelectElement).value;
          void loadExpenses();
        }}
        ><option value="">All currencies</option
        >{#each currencies as currency}<option value={currency}
            >{currency}</option
          >{/each}</select
      ></label
    >
    <label
      >Category<select
        value={filterCategory}
        onchange={(event) => {
          filterCategory = (event.currentTarget as HTMLSelectElement).value;
          void loadExpenses();
        }}
        ><option value="">All categories</option
        >{#each categories as category}<option value={category}
            >{category}</option
          >{/each}</select
      ></label
    >
  </div>
  {#if loading}<p class="muted">Loading expenses…</p>{:else}<ThemeRouteRenderer
      route="expenses"
      props={themeProps}
    />{/if}
</section>

<!-- svelte-ignore css_unused_selector -->
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
  .panel-head {
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
  .period {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.75rem 1rem;
  }
  .period-copy {
    display: grid;
    gap: 0.2rem;
  }
  .period-copy span {
    color: var(--accent);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .period-copy strong {
    font-size: 0.9rem;
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
  .expense-overview {
    display: grid;
    gap: 1rem;
  }
  .stat-strip {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 0.75rem;
  }
  .visual-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
  }
  .visual-card {
    display: grid;
    align-content: start;
    gap: 0.8rem;
  }
  .visual-head {
    display: flex;
    align-items: start;
    justify-content: space-between;
    gap: 1rem;
    padding-bottom: 0.75rem;
    border-bottom: 1px solid var(--border);
  }
  .visual-head h2 {
    margin-top: 0.25rem;
  }
  .visual-head > span {
    color: var(--text-muted);
    font-size: 0.75rem;
    font-weight: 700;
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
  select,
  button {
    min-height: 2.4rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface);
    color: var(--text);
    font: inherit;
  }
  select {
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
  .refresh {
    color: var(--accent);
  }
  .danger {
    color: var(--danger);
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
    .stat-strip,
    .visual-grid {
      grid-template-columns: 1fr;
    }
    .filters label {
      flex: 1 1 9rem;
    }
    .period {
      align-items: stretch;
      flex-direction: column;
    }
    .period :global(.month-navigator) {
      align-self: stretch;
      justify-content: space-between;
    }
  }
</style>
