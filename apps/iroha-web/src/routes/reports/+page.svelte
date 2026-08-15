<script lang="ts">
  import { onMount } from "svelte";
  import { replaceState } from "$app/navigation";
  import { page } from "$app/state";
  import { FileText, RefreshCw } from "@lucide/svelte";
  import {
    ApiError,
    getDailyBounds,
    getMonthlyReportSeries,
    type MonthlyReport,
    type MonthlyReportSeries,
  } from "$lib/api";
  import ReportComparison from "@iroha/shared/theme-ui/components/ReportComparison.svelte";
  import LoadingBoundary from "$lib/components/LoadingBoundary.svelte";
  import PeriodSelector from "$lib/components/PeriodSelector.svelte";
  import PeriodToolbar from "$lib/components/PeriodToolbar.svelte";
  import { formatDate } from "$lib/format";
  import {
    currentMonth,
    monthOptionsInRange,
    yearOptionsInRange,
  } from "@iroha/shared/month";
  import {
    currentCalendarScope,
    readCalendarScope,
    serializeCalendarScope,
    writeCalendarScope,
    type DateBounds,
  } from "@iroha/shared/scope";
  import { IROHA_TIMEZONE } from "$lib/config";
  import ThemeRouteRenderer from "@iroha/shared/theme-ui/ThemeRouteRenderer.svelte";
  import type { ReportThemeProps } from "@iroha/shared/report";
  import { useTheme } from "$lib/themes/context.svelte";

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
    requestedMonthScope.kind === "month"
      ? (serializeCalendarScope(requestedMonthScope) as string)
      : currentMonth(new Date(), IROHA_TIMEZONE),
  );
  let report = $state<MonthlyReport | null>(null);
  let series = $state<MonthlyReportSeries | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let requestVersion = 0;
  // The real cross-domain data range (fetched once, independent of the
  // current selection) -- not a hardcoded 2015 guess, and not every month.
  let bounds = $state<DateBounds>({});
  const periodYears = $derived(yearOptionsInRange(bounds));
  const periodYear = $derived(month.slice(0, 4));
  const periodMonth = $derived(String(Number(month.slice(5, 7))));
  const periodMonths = $derived(monthOptionsInRange(periodYear, bounds));
  const theme = useTheme();

  async function loadBounds() {
    try {
      bounds = await getDailyBounds();
    } catch {
      bounds = {};
    }
    if (!bounds.min || !bounds.max) return;
    if (month < bounds.min.slice(0, 7)) month = bounds.min.slice(0, 7);
    else if (month > bounds.max.slice(0, 7)) month = bounds.max.slice(0, 7);
    else return;
    moveMonth(month);
  }

  function loadingReportFor(value: string): MonthlyReport {
    const [year, monthNumber] = value.split("-").map(Number);
    const to = new Date(Date.UTC(year, monthNumber, 1))
      .toISOString()
      .slice(0, 10);
    const emptySection = (schema: string) => ({
      schema,
      state: "empty" as const,
      data: null,
    });
    return {
      schema: "monthly-report.v1",
      period: {
        kind: "month",
        month: value,
        from: `${value}-01`,
        to,
        timezone: IROHA_TIMEZONE,
      },
      generated_at: "",
      sections: {
        movement: emptySection("loading"),
        sleep: emptySection("loading"),
        daily_health: emptySection("loading"),
        media: emptySection("loading"),
        expenses: emptySection("loading"),
      },
    };
  }

  onMount(() => {
    void loadReport(month);
    void loadBounds();
  });

  async function loadReport(requestedMonth: string) {
    const version = ++requestVersion;
    loading = true;
    error = null;
    try {
      const trend = await getMonthlyReportSeries(requestedMonth);
      if (version !== requestVersion) return;
      report = trend.current_report;
      series = trend;
    } catch (cause) {
      if (version !== requestVersion) return;
      if (cause instanceof ApiError && cause.requestId)
        error = `${cause.message} (${cause.code}, request ${cause.requestId})`;
      else if (cause instanceof Error) error = cause.message;
      else error = String(cause);
    } finally {
      if (version === requestVersion) loading = false;
    }
  }

  function moveMonth(value: string) {
    month = value;
    const url = new URL(window.location.href);
    writeCalendarScope(url.searchParams, {
      kind: "month",
      year: Number(value.slice(0, 4)),
      month: Number(value.slice(5, 7)),
    });
    if (url.search !== window.location.search) replaceState(url, page.state);
    void loadReport(value);
  }

  function selectPeriodYear(value: string) {
    if (!/^\d{4}$/.test(value)) return;
    moveMonth(`${value}-${month.slice(5, 7)}`);
  }

  function selectPeriodMonth(value: string) {
    if (!/^(?:[1-9]|1[0-2])$/.test(value)) return;
    moveMonth(`${periodYear}-${value.padStart(2, "0")}`);
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

  const currentExpenseData = $derived(expenseData(report));
  const reportForView = $derived(report ?? loadingReportFor(month));
  const primaryCurrency = $derived(
    currentExpenseData?.totals_by_currency[0]?.currency ?? "JPY",
  );
  const primaryExponent = $derived(
    currentExpenseData?.totals_by_currency.find(
      (item) => item.currency === primaryCurrency,
    )?.currency_exponent ?? (primaryCurrency === "JPY" ? 0 : 2),
  );
  const themeProps = $derived<ReportThemeProps>({
    month,
    report: reportForView,
    primaryCurrency,
    primaryExponent,
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
  <PeriodToolbar title="Monthly cross-domain report" ariaLabel="Report period">
    <PeriodSelector
      year={periodYear}
      month={periodMonth}
      years={periodYears}
      months={periodMonths}
      {bounds}
      showAllYears={false}
      surface="inline"
      onYear={selectPeriodYear}
      onMonth={selectPeriodMonth}
    />
  </PeriodToolbar>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  <LoadingBoundary
    {loading}
    ready={report != null}
    preserveLayout
    label="Generating the monthly report…"
  >
    {#snippet children()}
      {#if report}
        <p class="generated">
          {report.period.from} → {report.period.to} · Generated {formatDate(
            report.generated_at,
          )}
        </p>
      {/if}
      <ReportComparison {series} {formatMoney} theme={theme.language()} />
      <ThemeRouteRenderer route="reports" props={themeProps} />
    {/snippet}
  </LoadingBoundary>
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
  .generated {
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .error {
    color: var(--danger);
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
  @media (max-width: 768px) {
    .page-head {
      align-items: start;
      flex-direction: column;
    }
  }
</style>
