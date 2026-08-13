<script lang="ts">
  import { onMount } from "svelte";
  import { FileText, RefreshCw } from "@lucide/svelte";
  import {
    ApiError,
    getMetricSeries,
    getMonthlyReport,
    type MetricSeriesResponse,
    type MonthlyReport,
  } from "$lib/api";
  import ThemeMonthNavigator from "$lib/themes/ThemeMonthNavigator.svelte";
  import {
    currentMonth,
    formatMonth,
    monthBounds,
    shiftMonth,
  } from "@iroha/shared/month";
  import ThemeRouteRenderer from "$lib/themes/ThemeRouteRenderer.svelte";
  import type { ReportThemeProps, ReportTrendPoint } from "$lib/report-view";

  let month = $state(currentMonth());
  let report = $state<MonthlyReport | null>(null);
  let trendSeries = $state<MetricSeriesResponse | null>(null);
  let countSeries = $state<MetricSeriesResponse | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);

  onMount(() => {
    void loadReport(month);
  });

  async function loadReport(requestedMonth: string) {
    loading = true;
    error = null;
    try {
      const current = await getMonthlyReport(requestedMonth);
      report = current;
      const expenseData = current.sections.expenses.data;
      const primaryCurrency =
        expenseData?.totals_by_currency[0]?.currency ?? "JPY";
      const from = monthBounds(shiftMonth(requestedMonth, -5)).from;
      const to = monthBounds(shiftMonth(requestedMonth, 1)).to;
      [trendSeries, countSeries] = await Promise.all([
        getMetricSeries("expenses.amount_minor", {
          from,
          to,
          grain: "month",
          dimensions: [`currency:${primaryCurrency}`],
        }),
        getMetricSeries("expenses.count", {
          from,
          to,
          grain: "month",
          dimensions: [`currency:${primaryCurrency}`],
        }),
      ]);
    } catch (cause) {
      if (cause instanceof ApiError && cause.requestId)
        error = `${cause.message} (${cause.code}, request ${cause.requestId})`;
      else if (cause instanceof Error) error = cause.message;
      else error = String(cause);
    } finally {
      loading = false;
    }
  }

  function moveMonth(value: string) {
    month = value;
    void loadReport(value);
  }

  function formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.round((seconds % 3600) / 60);
    return hours ? `${hours}h ${minutes}m` : `${minutes}m`;
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

  function expenseData(value: MonthlyReport | null) {
    return value?.sections.expenses.state === "available"
      ? value.sections.expenses.data
      : null;
  }

  function pointValue(
    series: MetricSeriesResponse | null,
    period: string,
  ): number | null {
    const point = series?.series[0]?.points.find(
      (item) => item.period === period,
    );
    if (!point) return null;
    return "value_minor" in point
      ? (point.value_minor ?? null)
      : (point.value ?? null);
  }

  function pointCount(period: string): number | null {
    const point = countSeries?.series[0]?.points.find(
      (item) => item.period === period,
    );
    return point && "value" in point ? (point.value ?? null) : null;
  }

  const currentExpenseData = $derived(expenseData(report));
  const primaryCurrency = $derived(
    currentExpenseData?.totals_by_currency[0]?.currency ?? "JPY",
  );
  const primaryExponent = $derived(
    currentExpenseData?.totals_by_currency.find(
      (item) => item.currency === primaryCurrency,
    )?.currency_exponent ?? (primaryCurrency === "JPY" ? 0 : 2),
  );
  const categoryTotals = $derived(
    [...(currentExpenseData?.by_category ?? [])]
      .filter((item) => item.currency === primaryCurrency)
      .sort((a, b) => b.amount_minor - a.amount_minor),
  );
  const trend = $derived<ReportTrendPoint[]>(
    trendSeries?.series[0]?.points.map((point) => ({
      month: point.period,
      label: formatMonth(point.period),
      amount:
        "value_minor" in point
          ? (point.value_minor ?? null)
          : (point.value ?? null),
      count: pointCount(point.period),
    })) ?? [],
  );
  const currentTotal = $derived(
    pointValue(trendSeries, month) ??
      currentExpenseData?.totals_by_currency.find(
        (item) => item.currency === primaryCurrency,
      )?.amount_minor ??
      0,
  );
  const previousTotal = $derived(
    pointValue(trendSeries, shiftMonth(month, -1)) ?? 0,
  );
  const expenseRecordCount = $derived(
    currentExpenseData?.expense_count ?? pointCount(month) ?? 0,
  );
  const topCategory = $derived(categoryTotals[0]?.category ?? "—");
  const currencyCount = $derived(
    currentExpenseData?.totals_by_currency.length ?? 0,
  );
  const comparisonLabel = $derived(
    previousTotal === 0
      ? currentTotal === 0
        ? "No movement in either month"
        : "New spending baseline"
      : `${currentTotal >= previousTotal ? "Up" : "Down"} ${Math.round(Math.abs((currentTotal - previousTotal) / previousTotal) * 100)}% vs previous month`,
  );
  const themeProps = $derived<ReportThemeProps>({
    month,
    report: report!,
    trend,
    trendSeries,
    primaryCurrency,
    primaryExponent,
    categoryTotals,
    currentTotal,
    previousTotal,
    expenseRecordCount,
    topCategory,
    currencyCount,
    comparisonLabel,
    formatMoney,
    formatDuration,
  });
</script>

<svelte:head><title>Reports · iroha</title></svelte:head>

<section class="reports-shell">
  <header class="page-head">
    <div>
      <p class="eyebrow"><FileText size={14} /> Monthly cockpit</p>
      <h1>Reports</h1>
      <p class="intro">
        A server-generated monthly view across canonical Iroha domains. Charts
        use the metric-series contract; details retain the report envelope and
        provenance.
      </p>
    </div>
    <button
      class="refresh"
      type="button"
      onclick={() => void loadReport(month)}
      disabled={loading}><RefreshCw size={15} /> Refresh</button
    >
  </header>
  <section class="period panel" aria-label="Report period">
    <div class="period-copy">
      <span>Period</span>
      <strong>Monthly cross-domain report</strong>
    </div>
    <ThemeMonthNavigator {month} onMonth={moveMonth} disabled={loading} />
  </section>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if loading}<p class="muted">
      Generating the monthly report…
    </p>{:else if report}<p class="generated">
      {report.period.from} → {report.period.to} · Generated {report.generated_at}
    </p>
    <ThemeRouteRenderer route="reports" props={themeProps} />{/if}
</section>

<style>
  .reports-shell {
    display: grid;
    gap: 1.25rem;
  }
  h1,
  p {
    margin: 0;
  }
  h1 {
    font-size: clamp(2.7rem, 7vw, 5.8rem);
    letter-spacing: -0.09em;
    line-height: 0.9;
  }
  .page-head {
    display: flex;
    justify-content: space-between;
    align-items: end;
    gap: 1rem;
    padding-bottom: 1.5rem;
    border-bottom: 1px solid var(--border);
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
  .period {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: center;
    padding: 1rem;
    border: 1px solid var(--border);
    background: var(--surface);
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
  .muted,
  .generated {
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .error {
    color: var(--danger);
  }
  .panel {
    min-width: 0;
  }
  button {
    min-height: 2.4rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface);
    color: var(--text);
    font: inherit;
    cursor: pointer;
  }
  button {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    padding: 0 0.8rem;
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
  @media (max-width: 760px) {
    .page-head,
    .period {
      align-items: start;
      flex-direction: column;
    }
  }
</style>
