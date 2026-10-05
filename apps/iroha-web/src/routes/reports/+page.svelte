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
  import RetryNotice from "@iroha/shared/theme-ui/components/RetryNotice.svelte";
  import PeriodSelector from "$lib/components/PeriodSelector.svelte";
  import PeriodToolbar from "$lib/components/PeriodToolbar.svelte";
  import { formatDate } from "$lib/format";
  import {
    currentMonth,
    MONTH_OPTIONS,
    monthOptionsInRange,
    yearOptionsInRange,
  } from "@iroha/shared/format/month";
  import {
    currentCalendarScope,
    readCalendarScope,
    serializeCalendarScope,
    writeCalendarScope,
    type DateBounds,
  } from "@iroha/shared/format/scope";
  import { IROHA_TIMEZONE } from "$lib/config";
  import ThemeRouteRenderer from "@iroha/shared/theme-ui/ThemeRouteRenderer.svelte";
  import type { ReportThemeProps } from "@iroha/shared/domain/report";
  import { useTheme } from "$lib/themes/context.svelte";
  import { createAsyncResource } from "$lib/asyncResource.svelte";

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
  const reportResource = createAsyncResource<MonthlyReportSeries>();
  const series = $derived(reportResource.data);
  const report = $derived(reportResource.data?.current_report ?? null);
  // The real cross-domain data range (fetched once, independent of the
  // current selection) -- not a hardcoded 2015 guess, and not every month.
  const boundsResource = createAsyncResource<DateBounds>();
  const bounds = $derived(boundsResource.data ?? {});
  let periodControls = $state<HTMLElement>();
  let observedPanel = $state<HTMLElement>();
  const periodYear = $derived(month.slice(0, 4));
  const periodMonth = $derived(String(Number(month.slice(5, 7))));
  // Reports has no lifetime fallback. Calendar choices remain usable without
  // range evidence, but are explicitly not an inventory of observed records.
  const periodYears = $derived(
    bounds.min && bounds.max ? yearOptionsInRange(bounds) : [periodYear],
  );
  const periodMonths = $derived(
    bounds.min && bounds.max
      ? monthOptionsInRange(periodYear, bounds)
      : [...MONTH_OPTIONS].reverse(),
  );
  const theme = useTheme();

  async function loadBounds() {
    const result = await boundsResource.run(() => getDailyBounds());
    if (!result?.min || !result.max) return;
    if (month < result.min.slice(0, 7)) month = result.min.slice(0, 7);
    else if (month > result.max.slice(0, 7)) month = result.max.slice(0, 7);
    else return;
    moveMonth(month);
  }

  onMount(() => {
    // Reports cannot serve year/lifetime URLs. Normalize the chosen monthly
    // fallback so even an incoming unsupported scope has an honest URL.
    moveMonth(month);
    void loadBounds();
  });

  async function loadReport(requestedMonth: string) {
    await reportResource.run(async () => {
      try {
        return await getMonthlyReportSeries(requestedMonth);
      } catch (cause) {
        throw new Error(formatError(cause));
      }
    });
  }

  function formatError(cause: unknown): string {
    if (cause instanceof ApiError && cause.requestId) {
      return `${cause.message} (${cause.code}, request ${cause.requestId})`;
    }
    return cause instanceof Error ? cause.message : String(cause);
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
  const primaryCurrency = $derived(
    currentExpenseData?.totals_by_currency[0]?.currency ?? "JPY",
  );
  const primaryExponent = $derived(
    currentExpenseData?.totals_by_currency.find(
      (item) => item.currency === primaryCurrency,
    )?.currency_exponent ?? (primaryCurrency === "JPY" ? 0 : 2),
  );
  const themeProps = $derived<ReportThemeProps | null>(
    report
      ? {
          report,
          primaryCurrency,
          primaryExponent,
          formatMoney,
          formatDuration,
        }
      : null,
  );
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
      disabled={reportResource.loading}><RefreshCw size={15} /> Refresh</button
    >
  </header>
  <p>Selected month: {month}</p>
  <section
    aria-label="Report period controls"
    tabindex="-1"
    bind:this={periodControls}
  >
    <PeriodToolbar
      title="Monthly cross-domain report"
      ariaLabel="Report period"
    >
      <PeriodSelector
        year={periodYear}
        month={periodMonth}
        years={periodYears}
        months={periodMonths}
        {bounds}
        showAllYears={false}
        showAllMonths={false}
        surface="inline"
        onYear={selectPeriodYear}
        onMonth={selectPeriodMonth}
      />
    </PeriodToolbar>
    {#if !bounds.min || !bounds.max}<p>
        Calendar months; record date range unavailable.
      </p>{/if}
    {#if boundsResource.error}<RetryNotice
        message={`Could not load report date range: ${boundsResource.error}`}
        retryLabel="Retry report date range"
        onRetry={loadBounds}
        focusTarget={periodControls}
      />{/if}
    {#if boundsResource.loading}<p role="status">
        Loading report date range…
      </p>{/if}
  </section>
  <section
    aria-label="Observed monthly report"
    tabindex="-1"
    bind:this={observedPanel}
  >
    {#if reportResource.error}<RetryNotice
        message={`Could not load monthly report: ${reportResource.error}`}
        retryLabel="Retry monthly report"
        onRetry={() => loadReport(month)}
        focusTarget={observedPanel}
      />{/if}
    {#if !report && reportResource.error}<p>Monthly report unavailable.</p>{/if}
    <LoadingBoundary
      resource={reportResource}
      preserveLayout
      label="Generating the monthly report…"
    >
      {#snippet children()}
        {#if report}
          <p>Observed month: {report.period.month}</p>
          <p class="generated">
            {report.period.from} → {report.period.to} · Generated {formatDate(
              report.generated_at,
            )}
          </p>
        {/if}
        {#if themeProps}
          <ReportComparison {series} {formatMoney} theme={theme.language()} />
          <ThemeRouteRenderer route="reports" props={themeProps} />
        {/if}
      {/snippet}
    </LoadingBoundary>
  </section>
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
