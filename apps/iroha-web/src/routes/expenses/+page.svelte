<script lang="ts">
  import { onMount } from "svelte";
  import { replaceState } from "$app/navigation";
  import { page } from "$app/state";
  import { RefreshCw } from "@lucide/svelte";
  import {
    ApiError,
    deleteExpense,
    getExpenseBounds,
    getMetricSeries,
    listAllExpenses,
    type Expense,
    type ExpenseCategory,
    type ExpenseCurrency,
    type MetricSeriesResponse,
  } from "$lib/api";
  import PeriodSelector from "$lib/components/PeriodSelector.svelte";
  import RouteHeader from "@iroha/shared/components/RouteHeader.svelte";
  import FilterSelect from "$lib/components/FilterSelect.svelte";
  import LoadingBoundary from "$lib/components/LoadingBoundary.svelte";
  import ConfirmDialog from "$lib/components/ConfirmDialog.svelte";
  import {
    currentMonth,
    monthOptionsInRange,
    yearOptionsInRange,
    MONTH_OPTIONS,
  } from "@iroha/shared/format/month";
  import {
    currentCalendarScope,
    parseCalendarScope,
    readCalendarScope,
    scopeBounds,
    serializeCalendarScope,
    writeCalendarScope,
    type DateBounds,
  } from "@iroha/shared/format/scope";
  import { IROHA_TIMEZONE } from "$lib/config";
  import {
    expenseCategoryLabel,
    type ExpensePanel,
    type ExpenseThemeProps,
  } from "@iroha/shared/view-contracts/expense-view";
  import { categoryColor } from "@iroha/shared/domain/category-color";
  import ThemeRouteRenderer from "@iroha/shared/theme-ui/ThemeRouteRenderer.svelte";
  import { createAsyncResource } from "$lib/asyncResource.svelte";
  import RetryNotice from "@iroha/shared/theme-ui/components/RetryNotice.svelte";

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
  const currencyOptions = currencies.map((currency) => ({
    value: currency,
    label: currency,
  }));
  const categoryOptions = [
    { value: "", label: "All categories" },
    ...categories.map((category) => ({
      value: category,
      label: expenseCategoryLabel[category],
    })),
  ];

  const expensesResource = createAsyncResource<{
    month: string;
    currency: string;
    category: string;
    expenses: Expense[];
  }>();
  const spendingResource = createAsyncResource<{
    month: string;
    currency: string;
    category: string;
    primaryCurrency: ExpenseCurrency;
    dailySeries: MetricSeriesResponse | null;
    categorySeries: MetricSeriesResponse[];
    currencySeries: MetricSeriesResponse[];
    currencyCountSeries: MetricSeriesResponse[];
  }>();
  const expenses = $derived(expensesResource.data?.expenses ?? []);
  const dailySeries = $derived(spendingResource.data?.dailySeries ?? null);
  const categorySeries = $derived(spendingResource.data?.categorySeries ?? []);
  const currencySeries = $derived(spendingResource.data?.currencySeries ?? []);
  const currencyCountSeries = $derived(
    spendingResource.data?.currencyCountSeries ?? [],
  );
  let recordsTarget = $state<HTMLElement>();
  let spendingTarget = $state<HTMLElement>();
  let boundsTarget = $state<HTMLElement>();
  let selected = $state<Expense | null>(null);
  let selectedId = $state("");
  let detailLoading = $state(false);
  // Only for delete failures -- load failures surface through
  // expensesResource.error instead.
  let deleteError = $state<string | null>(null);
  let confirmDeleteOpen = $state(false);
  let pendingDelete = $state<Expense | null>(null);

  const defaultMonthScope = currentCalendarScope(
    "month",
    new Date(),
    IROHA_TIMEZONE,
  );
  const requestedMonthScope = readCalendarScope(page.url.searchParams, {
    fallback: defaultMonthScope,
    allowDay: false,
  });
  // Either a specific "YYYY-MM" or a whole calendar year "YYYY" -- "All
  // months" in the period selector means the latter, not "no filter".
  let month = $state(
    requestedMonthScope.kind === "month" || requestedMonthScope.kind === "year"
      ? (serializeCalendarScope(requestedMonthScope) as string)
      : currentMonth(new Date(), IROHA_TIMEZONE),
  );
  let filterCurrency = $state(
    currencies.includes(
      page.url.searchParams.get("currency") as ExpenseCurrency,
    )
      ? (page.url.searchParams.get("currency") as ExpenseCurrency)
      : "",
  );
  let filterCategory = $state<ExpenseCategory | "">(
    categories.includes(
      page.url.searchParams.get("category") as ExpenseCategory,
    )
      ? (page.url.searchParams.get("category") as ExpenseCategory)
      : "",
  );
  // The real data range (fetched once, independent of the current
  // selection) -- not a hardcoded 2015 guess, and not every calendar month.
  const boundsResource = createAsyncResource<DateBounds>();
  const dateBounds = $derived(boundsResource.data ?? {});
  const periodYears = $derived(
    yearOptionsInRange(dateBounds).length
      ? yearOptionsInRange(dateBounds)
      : [month.slice(0, 4)],
  );
  const periodYear = $derived(month.slice(0, 4));
  const periodMonth = $derived(
    /^\d{4}-\d{2}$/.test(month) ? String(Number(month.slice(5, 7))) : "",
  );
  const periodMonths = $derived(
    dateBounds.min && dateBounds.max
      ? monthOptionsInRange(periodYear, dateBounds)
      : [...MONTH_OPTIONS].reverse(),
  );

  async function loadBounds() {
    const result = await boundsResource.run(async () => {
      try {
        return await getExpenseBounds();
      } catch (cause) {
        throw new Error(formatError(cause));
      }
    });
    if (!result) return;
    if (!dateBounds.min || !dateBounds.max) return;
    // Only clamp a specific month against the real data range -- a
    // deliberately wider "All months" (year) selection isn't stale state
    // to correct back to, it's what the user asked to see.
    if (!/^\d{4}-\d{2}$/.test(month)) return;
    if (month < dateBounds.min.slice(0, 7)) month = dateBounds.min.slice(0, 7);
    else if (month > dateBounds.max.slice(0, 7))
      month = dateBounds.max.slice(0, 7);
    else return;
    syncUrl();
    void loadExpenses(month);
  }

  function metricDimensions(
    chartCurrencies: ExpenseCurrency[],
    chartCategories: ExpenseCategory[] = [],
  ): string[] {
    return [
      ...chartCurrencies.map((currency) => `currency:${currency}`),
      ...chartCategories.map((category) => `category:${category}`),
    ];
  }

  onMount(() => {
    syncUrl();
    void loadExpenses(month);
    void loadBounds();
  });

  async function loadExpenses(selectedMonth = month) {
    await Promise.all([
      loadRecords(selectedMonth),
      loadSpending(selectedMonth),
    ]);
  }

  async function loadRecords(selectedMonth = month) {
    const currency = filterCurrency;
    const category = filterCategory;
    const result = await expensesResource.run(async () => {
      try {
        return {
          month: selectedMonth,
          currency,
          category,
          expenses: await listAllExpenses({
            date: selectedMonth,
            currency: (currency || undefined) as ExpenseCurrency | undefined,
            category: (category || undefined) as ExpenseCategory | undefined,
          }),
        };
      } catch (cause) {
        throw new Error(formatError(cause));
      }
    });
    if (!result) return;
    selected =
      result.expenses.find((expense) => expense.id === selectedId) ??
      result.expenses[0] ??
      null;
    selectedId = selected?.id ?? "";
  }

  async function loadSpending(selectedMonth = month) {
    const currency = filterCurrency;
    const category = filterCategory;
    await spendingResource.run(async () => {
      try {
        const scope = parseCalendarScope(selectedMonth)!;
        const bounds = scopeBounds(scope)!;
        // A totals bucket must span exactly [bounds.from, bounds.to) so
        // points[0] below is genuinely "the total for this scope" -- a
        // month scope needs a month-grain bucket, a year scope a
        // year-grain one; using "month" for a whole year would return 12
        // points and silently read only the first month's total.
        const totalsGrain = scope.kind === "year" ? "year" : "month";
        const chartCurrencies = currency
          ? [currency as ExpenseCurrency]
          : currencies;
        const selectedCategory = categories.includes(
          category as ExpenseCategory,
        )
          ? [category as ExpenseCategory]
          : [];
        const [currenciesForMonth, countsForCurrency] = await Promise.all([
          getMetricSeries("expenses.amount_minor", {
            from: bounds.from,
            to: bounds.to,
            grain: totalsGrain,
            dimensions: metricDimensions(chartCurrencies, selectedCategory),
          }),
          getMetricSeries("expenses.count", {
            from: bounds.from,
            to: bounds.to,
            grain: totalsGrain,
            dimensions: metricDimensions(chartCurrencies, selectedCategory),
          }),
        ]);
        const chartCurrency = (currency ||
          currenciesForMonth.series.find(
            (_, index) =>
              (seriesPointValue(currenciesForMonth, index) ?? 0) !== 0,
          )?.dimensions.currency ||
          currenciesForMonth.series[0]?.dimensions.currency ||
          "JPY") as ExpenseCurrency;
        const chartCategories: ExpenseCategory[] = category
          ? [category as ExpenseCategory]
          : categories;
        const [daily, categoriesForCurrency] = await Promise.all([
          getMetricSeries("expenses.amount_minor", {
            from: bounds.from,
            to: bounds.to,
            grain: "day",
            dimensions: metricDimensions([chartCurrency], selectedCategory),
          }),
          getMetricSeries("expenses.amount_minor", {
            from: bounds.from,
            to: bounds.to,
            grain: totalsGrain,
            dimensions: metricDimensions([chartCurrency], chartCategories),
          }),
        ]);
        return {
          month: selectedMonth,
          currency,
          category,
          primaryCurrency: chartCurrency,
          dailySeries: daily,
          categorySeries: [categoriesForCurrency],
          currencySeries: [currenciesForMonth],
          currencyCountSeries: [countsForCurrency],
        };
      } catch (cause) {
        throw new Error(formatError(cause));
      }
    });
  }

  function selectMonth(value: string) {
    month = value;
    syncUrl();
    void loadExpenses(value);
  }

  function selectPeriodYear(value: string) {
    if (!/^\d{4}$/.test(value)) return;
    selectMonth(
      periodMonth ? `${value}-${periodMonth.padStart(2, "0")}` : value,
    );
  }

  function selectPeriodMonth(value: string) {
    if (value === "") {
      // "All months" -- zoom out from a specific month to the whole year.
      selectMonth(periodYear);
      return;
    }
    if (!/^(?:[1-9]|1[0-2])$/.test(value)) return;
    selectMonth(`${periodYear}-${value.padStart(2, "0")}`);
  }

  function selectCurrency(value: string) {
    filterCurrency = currencies.includes(value as ExpenseCurrency)
      ? (value as ExpenseCurrency)
      : "";
    syncUrl();
    void loadExpenses();
  }

  function selectCategory(value: string) {
    filterCategory = categories.includes(value as ExpenseCategory)
      ? (value as ExpenseCategory)
      : "";
    syncUrl();
    void loadExpenses();
  }

  async function selectExpense(id: string) {
    selectedId = id;
    selected = expenses.find((expense) => expense.id === id) ?? null;
  }

  function syncUrl() {
    const url = new URL(window.location.href);
    writeCalendarScope(
      url.searchParams,
      parseCalendarScope(month) ?? defaultMonthScope,
    );
    if (filterCurrency) url.searchParams.set("currency", filterCurrency);
    else url.searchParams.delete("currency");
    if (filterCategory) url.searchParams.set("category", filterCategory);
    else url.searchParams.delete("category");
    if (url.search !== window.location.search) replaceState(url, page.state);
  }

  function removeExpense(expense: Expense) {
    pendingDelete = expense;
    confirmDeleteOpen = true;
  }

  async function confirmRemoveExpense() {
    const expense = pendingDelete;
    pendingDelete = null;
    if (!expense) return;
    deleteError = null;
    try {
      await deleteExpense(expense.id);
      await loadExpenses();
    } catch (cause) {
      deleteError = formatError(cause);
    }
  }

  function formatError(cause: unknown): string {
    if (cause instanceof ApiError && cause.requestId) {
      return `${cause.message} (${cause.code}, request ${cause.requestId})`;
    }
    return cause instanceof Error ? cause.message : String(cause);
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
    index = 0,
  ): number | null {
    const point = series?.series[index]?.points[0];
    return point?.value_minor ?? null;
  }

  function numericSeriesPointValue(
    series: MetricSeriesResponse | null,
    index = 0,
  ): number | null {
    const point = series?.series[index]?.points[0];
    return point && "value" in point ? (point.value ?? null) : null;
  }

  const currencyTotals = $derived(
    currencySeries
      .flatMap((response) =>
        response.series.map((dimensionSeries, index) => {
          const currency = dimensionSeries.dimensions
            .currency as ExpenseCurrency;
          const countResponse = currencyCountSeries[0] ?? null;
          const countIndex =
            countResponse?.series.findIndex(
              (candidate) => candidate.dimensions.currency === currency,
            ) ?? -1;
          return {
            currency,
            amountMinor: seriesPointValue(response, index),
            exponent: currency === "JPY" ? 0 : 2,
            count:
              countIndex < 0
                ? null
                : numericSeriesPointValue(countResponse, countIndex),
          };
        }),
      )
      .filter((item) => item.currency)
      .sort(
        (a, b) => (b.amountMinor ?? -Infinity) - (a.amountMinor ?? -Infinity),
      ),
  );
  const primaryCurrency = $derived(
    spendingResource.data?.primaryCurrency ?? "JPY",
  );
  const primaryExponent = $derived(
    expenses.find((item) => item.currency === primaryCurrency)
      ?.currency_exponent ?? (primaryCurrency === "JPY" ? 0 : 2),
  );
  const categoryTotals = $derived(
    categorySeries
      .flatMap((response) =>
        response.series.map((dimensionSeries, index) => ({
          category: dimensionSeries.dimensions.category ?? "",
          amount: seriesPointValue(response, index),
        })),
      )
      .filter((item) => item.category)
      .sort((a, b) => (b.amount ?? -Infinity) - (a.amount ?? -Infinity)),
  );
  const dailyTotals = $derived(
    (dailySeries?.series[0]?.points ?? []).map(
      (point) =>
        [point.period, point.value_minor ?? null] as [string, number | null],
    ),
  );

  const dailyPanel = $derived<ExpensePanel>({
    metricId: "expenses.amount_minor",
    unit: `${primaryCurrency} minor`,
    method: dailySeries?.series[0]?.source.method ?? "expenses.sum_active.v1",
    coverage: dailySeries?.series[0]?.coverage,
    sourceKinds: dailySeries?.series[0]?.source.source_kinds ?? [],
    rowHeader: "Day",
    rows: dailyTotals.map(([day, amount]) => ({
      label: day,
      value: amount,
      display:
        amount == null
          ? "—"
          : formatMoney(amount, primaryCurrency, primaryExponent),
    })),
  });
  const categoryPanel = $derived<ExpensePanel>({
    metricId: "expenses.amount_minor",
    unit: `${primaryCurrency} minor`,
    method:
      categorySeries[0]?.series[0]?.source.method ?? "expenses.sum_active.v1",
    sourceKinds: categorySeries[0]?.series[0]?.source.source_kinds ?? [],
    rowHeader: "Category",
    rows: categoryTotals.map((item) => ({
      label: item.category,
      value: item.amount,
      display:
        item.amount == null
          ? "—"
          : formatMoney(item.amount, primaryCurrency, primaryExponent),
    })),
  });

  const themeProps = $derived<ExpenseThemeProps>({
    month: expensesResource.data?.month ?? month,
    primaryCurrency,
    primaryExponent,
    currencyTotals,
    categoryTotals,
    dailyTotals,
    dailyPanel,
    categoryPanel,
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
  <section
    aria-label="Expense period selection"
    tabindex="-1"
    bind:this={boundsTarget}
  >
    <RouteHeader
      title="Expenses"
      context={expensesResource.data
        ? `Observed period: ${expensesResource.data.month}`
        : expensesResource.error
          ? "Ledger period unavailable"
          : expensesResource.loading
            ? "Loading ledger period…"
            : "No ledger period loaded"}
    >
      {#snippet actions()}
        <button
          class="refresh"
          type="button"
          onclick={() => void loadExpenses()}
          disabled={expensesResource.loading}
          ><RefreshCw size={15} /> Refresh</button
        >
      {/snippet}
      <div class="expense-toolbar-controls">
        <PeriodSelector
          year={periodYear}
          month={periodMonth}
          years={periodYears}
          months={periodMonths}
          bounds={dateBounds}
          showAllYears={false}
          surface="inline"
          onYear={selectPeriodYear}
          onMonth={selectPeriodMonth}
        />
        <div class="expense-dimensions" aria-label="Expense dimensions">
          <FilterSelect
            label="Currency"
            value={filterCurrency}
            options={[
              { value: "", label: "All currencies" },
              ...currencyOptions,
            ]}
            onChange={selectCurrency}
          />
          <FilterSelect
            label="Category"
            value={filterCategory}
            options={categoryOptions}
            markerColor={categoryColor(filterCategory || "other")}
            onChange={selectCategory}
          />
        </div>
      </div>
    </RouteHeader>
    <p>
      Selected period: {month} · {filterCurrency || "All currencies"} · {filterCategory
        ? expenseCategoryLabel[filterCategory]
        : "All categories"}
    </p>
    {#if boundsResource.error}
      <RetryNotice
        message={`Expense range unavailable. ${boundsResource.error}`}
        retryLabel="Retry expense range"
        onRetry={loadBounds}
        focusTarget={boundsTarget}
      />
      <p>
        {dateBounds.min && dateBounds.max
          ? "The last observed expense range remains available."
          : "Period choices are calendar choices, not an observed data inventory."}
      </p>
    {/if}
  </section>
  {#if deleteError}
    <p class="error" role="alert">Delete failed: {deleteError}</p>
  {/if}
  <section
    aria-label="Expense spending"
    tabindex="-1"
    bind:this={spendingTarget}
  >
    {#if spendingResource.error}
      <RetryNotice
        message={`Expense spending unavailable. ${spendingResource.error}`}
        retryLabel="Retry expense spending"
        onRetry={() => loadSpending()}
        focusTarget={spendingTarget}
      />
    {/if}
    <LoadingBoundary
      resource={spendingResource}
      preserveLayout
      label="Loading expense spending…"
    >
      {#snippet children()}
        {#if spendingResource.data}
          <p>
            Observed spending period: {spendingResource.data.month} · {spendingResource
              .data.currency || "All currencies"} · {spendingResource.data
              .category
              ? expenseCategoryLabel[
                  spendingResource.data.category as ExpenseCategory
                ]
              : "All categories"}
          </p>
          <ThemeRouteRenderer
            route="expenses"
            props={{
              ...themeProps,
              month: spendingResource.data.month,
              section: "spending",
            }}
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
    {#if expensesResource.error}
      <RetryNotice
        message={`Expense records unavailable. ${expensesResource.error}`}
        retryLabel="Retry expense records"
        onRetry={() => loadRecords()}
        focusTarget={recordsTarget}
      />
    {/if}
    <LoadingBoundary
      resource={expensesResource}
      preserveLayout
      label="Loading expense records…"
    >
      {#snippet children()}
        {#if expensesResource.data}
          <p>
            Observed ledger period: {expensesResource.data.month} · {expensesResource
              .data.currency || "All currencies"} · {expensesResource.data
              .category
              ? expenseCategoryLabel[
                  expensesResource.data.category as ExpenseCategory
                ]
              : "All categories"}
          </p>
          <ThemeRouteRenderer
            route="expenses"
            props={{ ...themeProps, section: "ledger" }}
          />
        {/if}
      {/snippet}
    </LoadingBoundary>
  </section>
</section>

<ConfirmDialog
  bind:open={confirmDeleteOpen}
  message={pendingDelete
    ? `Delete expense from ${pendingDelete.occurred_on}?`
    : ""}
  onConfirm={() => void confirmRemoveExpense()}
/>

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
    font-size: var(--type-display);
    letter-spacing: -0.09em;
    line-height: 0.9;
  }
  h2 {
    font-size: var(--type-title);
    letter-spacing: -0.04em;
  }
  h3 {
    font-size: var(--type-label);
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
  .eyebrow {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    color: var(--accent);
    font-size: var(--type-caption);
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
    font-size: var(--type-caption);
    font-weight: 700;
  }
  .expense-toolbar-controls {
    display: flex;
    flex-wrap: wrap;
    align-items: end;
    justify-content: flex-end;
    gap: 0.75rem 1rem;
  }
  .expense-dimensions {
    display: flex;
    flex-wrap: wrap;
    align-items: end;
    gap: 0.75rem;
    padding-left: 1rem;
    border-left: 1px solid var(--border);
  }
  .expense-dimensions :global(.select-control) {
    flex: 1 1 9rem;
  }
  button {
    min-height: 2.4rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface);
    color: var(--text);
    font: inherit;
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
    font-size: var(--type-caption);
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
    font-size: var(--type-caption);
  }
  dd {
    min-width: 0;
    overflow-wrap: anywhere;
  }
  .mono {
    font-family: var(--font-mono, monospace);
    font-size: var(--type-caption);
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
    font-size: var(--type-label);
  }
  .timestamps {
    margin-top: 1.2rem;
    font-size: var(--type-caption);
  }
  .empty-detail {
    display: grid;
    justify-items: center;
    gap: 0.65rem;
    padding: 4rem 1rem;
    color: var(--text-muted);
    text-align: center;
  }
  @media (max-width: 768px) {
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
    .expense-toolbar-controls {
      width: 100%;
      justify-content: stretch;
    }
    .expense-toolbar-controls :global(.period-controls),
    .expense-dimensions {
      flex: 1 1 100%;
    }
    .expense-dimensions {
      padding-top: 0.75rem;
      padding-left: 0;
      border-top: 1px solid var(--border);
      border-left: 0;
    }
  }
</style>
