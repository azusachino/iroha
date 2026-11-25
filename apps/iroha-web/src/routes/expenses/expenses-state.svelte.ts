import { onMount } from "svelte";
import { replaceState } from "$app/navigation";
import { page } from "$app/state";
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
import { createAsyncResource } from "$lib/asyncResource.svelte";

const CURRENCIES: ExpenseCurrency[] = ["JPY", "USD", "EUR", "GBP"];
const CATEGORIES: ExpenseCategory[] = [
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

export const CURRENCY_OPTIONS = CURRENCIES.map((currency) => ({
  value: currency,
  label: currency,
}));

export const CATEGORY_OPTIONS = [
  { value: "", label: "All categories" },
  ...CATEGORIES.map((category) => ({
    value: category,
    label: expenseCategoryLabel[category],
  })),
];

function metricDimensions(
  chartCurrencies: ExpenseCurrency[],
  chartCategories: ExpenseCategory[] = [],
): string[] {
  return [
    ...chartCurrencies.map((currency) => `currency:${currency}`),
    ...chartCategories.map((category) => `category:${category}`),
  ];
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

export function createExpensesState() {
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
  const boundsResource = createAsyncResource<DateBounds>();

  const expenses = $derived(expensesResource.data?.expenses ?? []);
  const dailySeries = $derived(spendingResource.data?.dailySeries ?? null);
  const categorySeries = $derived(spendingResource.data?.categorySeries ?? []);
  const currencySeries = $derived(spendingResource.data?.currencySeries ?? []);
  const currencyCountSeries = $derived(
    spendingResource.data?.currencyCountSeries ?? [],
  );

  let selected = $state<Expense | null>(null);
  let selectedId = $state("");
  const detailLoading = false;
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
  let month = $state(
    requestedMonthScope.kind === "month" || requestedMonthScope.kind === "year"
      ? (serializeCalendarScope(requestedMonthScope) as string)
      : currentMonth(new Date(), IROHA_TIMEZONE),
  );
  let filterCurrency = $state(
    CURRENCIES.includes(
      page.url.searchParams.get("currency") as ExpenseCurrency,
    )
      ? (page.url.searchParams.get("currency") as ExpenseCurrency)
      : "",
  );
  let filterCategory = $state<ExpenseCategory | "">(
    CATEGORIES.includes(
      page.url.searchParams.get("category") as ExpenseCategory,
    )
      ? (page.url.searchParams.get("category") as ExpenseCategory)
      : "",
  );

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
      : MONTH_OPTIONS.toReversed(),
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
    if (!/^\d{4}-\d{2}$/.test(month)) return;
    if (month < dateBounds.min.slice(0, 7)) month = dateBounds.min.slice(0, 7);
    else if (month > dateBounds.max.slice(0, 7))
      month = dateBounds.max.slice(0, 7);
    else return;
    syncUrl();
    void loadExpenses(month);
  }

  async function loadExpenses(selectedMonth = month, forceRefresh = false) {
    await Promise.all([
      loadRecords(selectedMonth, forceRefresh),
      loadSpending(selectedMonth, forceRefresh),
    ]);
  }

  async function loadRecords(selectedMonth = month, forceRefresh = false) {
    const currency = filterCurrency;
    const category = filterCategory;
    const result = await expensesResource.run(async () => {
      try {
        return {
          month: selectedMonth,
          currency,
          category,
          expenses: await listAllExpenses(
            {
              date: selectedMonth,
              currency: (currency || undefined) as ExpenseCurrency | undefined,
              category: (category || undefined) as ExpenseCategory | undefined,
            },
            { forceRefresh },
          ),
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

  async function loadSpending(selectedMonth = month, forceRefresh = false) {
    const currency = filterCurrency;
    const category = filterCategory;
    await spendingResource.run(async () => {
      try {
        const scope = parseCalendarScope(selectedMonth)!;
        const bounds = scopeBounds(scope)!;
        const totalsGrain = scope.kind === "year" ? "year" : "month";
        const chartCurrencies = currency
          ? [currency as ExpenseCurrency]
          : CURRENCIES;
        const selectedCategory = CATEGORIES.includes(
          category as ExpenseCategory,
        )
          ? [category as ExpenseCategory]
          : [];
        const [currenciesForMonth, countsForCurrency] = await Promise.all([
          getMetricSeries(
            "expenses.amount_minor",
            {
              from: bounds.from,
              to: bounds.to,
              grain: totalsGrain,
              dimensions: metricDimensions(chartCurrencies, selectedCategory),
            },
            { forceRefresh },
          ),
          getMetricSeries(
            "expenses.count",
            {
              from: bounds.from,
              to: bounds.to,
              grain: totalsGrain,
              dimensions: metricDimensions(chartCurrencies, selectedCategory),
            },
            { forceRefresh },
          ),
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
          : CATEGORIES;
        const [daily, categoriesForCurrency] = await Promise.all([
          getMetricSeries(
            "expenses.amount_minor",
            {
              from: bounds.from,
              to: bounds.to,
              grain: "day",
              dimensions: metricDimensions([chartCurrency], selectedCategory),
            },
            { forceRefresh },
          ),
          getMetricSeries(
            "expenses.amount_minor",
            {
              from: bounds.from,
              to: bounds.to,
              grain: totalsGrain,
              dimensions: metricDimensions([chartCurrency], chartCategories),
            },
            { forceRefresh },
          ),
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
      selectMonth(periodYear);
      return;
    }
    if (!/^(?:[1-9]|1[0-2])$/.test(value)) return;
    selectMonth(`${periodYear}-${value.padStart(2, "0")}`);
  }

  function selectCurrency(value: string) {
    filterCurrency = CURRENCIES.includes(value as ExpenseCurrency)
      ? (value as ExpenseCurrency)
      : "";
    syncUrl();
    void loadExpenses();
  }

  function selectCategory(value: string) {
    filterCategory = CATEGORIES.includes(value as ExpenseCategory)
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
      .toSorted(
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
      .toSorted((a, b) => (b.amount ?? -Infinity) - (a.amount ?? -Infinity)),
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

  onMount(() => {
    syncUrl();
    void loadExpenses(month);
    void loadBounds();
  });

  return {
    expensesResource,
    spendingResource,
    boundsResource,
    get month() {
      return month;
    },
    get filterCurrency() {
      return filterCurrency;
    },
    get filterCategory() {
      return filterCategory;
    },
    get dateBounds() {
      return dateBounds;
    },
    get periodYears() {
      return periodYears;
    },
    get periodYear() {
      return periodYear;
    },
    get periodMonth() {
      return periodMonth;
    },
    get periodMonths() {
      return periodMonths;
    },
    get deleteError() {
      return deleteError;
    },
    get confirmDeleteOpen() {
      return confirmDeleteOpen;
    },
    set confirmDeleteOpen(value: boolean) {
      confirmDeleteOpen = value;
    },
    get pendingDelete() {
      return pendingDelete;
    },
    get themeProps() {
      return themeProps;
    },
    loadBounds,
    loadExpenses,
    loadRecords,
    loadSpending,
    selectPeriodYear,
    selectPeriodMonth,
    selectCurrency,
    selectCategory,
    confirmRemoveExpense,
  };
}
