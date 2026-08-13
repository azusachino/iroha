<script lang="ts">
  import { onMount } from "svelte";
  import { replaceState } from "$app/navigation";
  import { page } from "$app/state";
  import { FileText, RefreshCw } from "@lucide/svelte";
  import { ApiError, getMonthlyReport, type MonthlyReport } from "$lib/api";
  import ThemeMonthNavigator from "$lib/themes/ThemeMonthNavigator.svelte";
  import { currentMonth, canonicalMonth } from "@iroha/shared/month";
  import ThemeRouteRenderer from "$lib/themes/ThemeRouteRenderer.svelte";
  import type { ReportThemeProps } from "$lib/report-view";

  let month = $state(
    canonicalMonth(page.url.searchParams.get("month"), currentMonth()),
  );
  let report = $state<MonthlyReport | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let requestVersion = 0;

  onMount(() => {
    void loadReport(month);
  });

  async function loadReport(requestedMonth: string) {
    const version = ++requestVersion;
    loading = true;
    error = null;
    try {
      const current = await getMonthlyReport(requestedMonth);
      if (version !== requestVersion) return;
      report = current;
    } catch (cause) {
      if (version !== requestVersion) return;
      report = null;
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
    const url = new URL(page.url);
    url.searchParams.set("month", value);
    if (url.search !== page.url.search) replaceState(url, page.state);
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

  const currentExpenseData = $derived(expenseData(report));
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
    report: report!,
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
