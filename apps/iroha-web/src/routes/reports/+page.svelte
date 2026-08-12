<script lang="ts">
  import { onMount } from "svelte";
  import { FileText, RefreshCw } from "@lucide/svelte";
  import { ApiError, getMonthlyReport, type MonthlyReport } from "$lib/api";
  import BarChart from "$lib/components/BarChart.svelte";
  import MonthNavigator from "@iroha/shared/MonthNavigator.svelte";
  import { currentMonth, formatMonth, shiftMonth } from "@iroha/shared/month";
  import ThemeRouteRenderer from "$lib/themes/ThemeRouteRenderer.svelte";

  let month = $state(currentMonth());
  let timezone = $state("");
  let report = $state<MonthlyReport | null>(null);
  let recentReports = $state<MonthlyReport[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  onMount(() => {
    timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
    void loadReport(month);
  });

  async function loadReport(requestedMonth: string) {
    if (!timezone) return;
    loading = true;
    error = null;
    try {
      const months = Array.from({ length: 6 }, (_, index) =>
        shiftMonth(requestedMonth, -index),
      );
      const responses = await Promise.all(
        months.map((value) => getMonthlyReport(value, timezone)),
      );
      report = responses[0];
      recentReports = responses;
    } catch (cause) {
      if (cause instanceof ApiError && cause.requestId) {
        error = `${cause.message} (${cause.code}, request ${cause.requestId})`;
      } else if (cause instanceof Error) {
        error = cause.message;
      } else {
        error = String(cause);
      }
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

  function sectionState(state: string): string {
    return state === "empty" ? "No canonical records for this month." : "";
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
    )?.currency_exponent ?? 0,
  );
  const categoryTotals = $derived(
    [...(currentExpenseData?.by_category ?? [])]
      .filter((item) => item.currency === primaryCurrency)
      .sort((a, b) => b.amount_minor - a.amount_minor),
  );
  const recentTotals = $derived(
    [...recentReports].reverse().map((item) => {
      const data = expenseData(item);
      const total = data?.totals_by_currency.find(
        (value) => value.currency === primaryCurrency,
      );
      return {
        month: item.period.month,
        label: formatMonth(item.period.month),
        amount: total?.amount_minor ?? 0,
        count: total?.expense_count ?? 0,
      };
    }),
  );
  const previousTotal = $derived(
    recentReports[1]
      ? (expenseData(recentReports[1])?.totals_by_currency.find(
          (value) => value.currency === primaryCurrency,
        )?.amount_minor ?? 0)
      : 0,
  );
  const currentTotal = $derived(
    currentExpenseData?.totals_by_currency.find(
      (value) => value.currency === primaryCurrency,
    )?.amount_minor ?? 0,
  );
  const comparisonLabel = $derived(
    previousTotal === 0
      ? currentTotal === 0
        ? "No movement in either month"
        : "New spending baseline"
      : `${currentTotal >= previousTotal ? "Up" : "Down"} ${Math.round(Math.abs((currentTotal - previousTotal) / previousTotal) * 100)}% vs previous month`,
  );
</script>

<svelte:head>
  <title>Reports · iroha</title>
</svelte:head>

<ThemeRouteRenderer route="reports" props={{ route: "reports" }}>
  {#snippet children()}
    <section class="reports-shell">
      <header class="page-head">
        <div>
          <p class="eyebrow"><FileText size={14} /> Monthly cockpit</p>
          <h1>Reports</h1>
          <p class="intro">
            A server-generated view across the canonical Iroha domains. The
            month and browser timezone are sent to the report API; this page
            does not aggregate client-side.
          </p>
        </div>
        <button
          class="refresh"
          type="button"
          onclick={() => void loadReport(month)}
          disabled={loading}
        >
          <RefreshCw size={15} /> Refresh
        </button>
      </header>

      <section class="period panel" aria-label="Report period">
        <MonthNavigator {month} onMonth={moveMonth} disabled={loading} />
        <span>{timezone || "Detecting browser timezone…"}</span>
      </section>

      {#if error}<p class="error" role="alert">{error}</p>{/if}
      {#if loading}
        <p class="muted">Generating the monthly report…</p>
      {:else if report}
        <p class="generated">
          {report.period.from} → {report.period.to} · Generated {report.generated_at}
        </p>
        <div class="report-grid">
          <section class="report-card panel" aria-labelledby="movement-title">
            <header>
              <p class="eyebrow">Movement</p>
              <h2 id="movement-title">Activity</h2>
            </header>
            {#if report.sections.movement.state === "available" && report.sections.movement.data}
              <div class="stats">
                <div>
                  <b>{report.sections.movement.data.activity_count}</b><span
                    >activities</span
                  >
                </div>
                <div>
                  <b
                    >{(report.sections.movement.data.distance_m / 1000).toFixed(
                      1,
                    )} km</b
                  ><span>distance</span>
                </div>
                <div>
                  <b
                    >{formatDuration(
                      report.sections.movement.data.duration_s,
                    )}</b
                  ><span>duration</span>
                </div>
              </div>
              <ul class="sub-list">
                {#each report.sections.movement.data.by_sport as sport}<li>
                    <span>{sport.sport}</span><span
                      >{sport.activity_count} · {(
                        sport.distance_m / 1000
                      ).toFixed(1)} km</span
                    >
                  </li>{/each}
              </ul>
            {:else}<p class="empty">
                {sectionState(report.sections.movement.state)}
              </p>{/if}
          </section>

          <section class="report-card panel" aria-labelledby="sleep-title">
            <header>
              <p class="eyebrow">Sleep</p>
              <h2 id="sleep-title">Rest</h2>
            </header>
            {#if report.sections.sleep.state === "available" && report.sections.sleep.data}
              <div class="stats">
                <div>
                  <b
                    >{formatDuration(
                      report.sections.sleep.data.average_asleep_s,
                    )}</b
                  ><span>average asleep</span>
                </div>
                <div>
                  <b
                    >{Math.round(
                      report.sections.sleep.data.average_efficiency * 100,
                    )}%</b
                  ><span>efficiency</span>
                </div>
                <div>
                  <b>{report.sections.sleep.data.session_count}</b><span
                    >sessions</span
                  >
                </div>
              </div>
              <p class="card-note">
                {report.sections.sleep.data.main_sleep_count} main sleeps · {report
                  .sections.sleep.data.nap_count} naps
              </p>
            {:else}<p class="empty">
                {sectionState(report.sections.sleep.state)}
              </p>{/if}
          </section>

          <section class="report-card panel" aria-labelledby="health-title">
            <header>
              <p class="eyebrow">Daily health</p>
              <h2 id="health-title">Body signals</h2>
            </header>
            {#if report.sections.daily_health.state === "available" && report.sections.daily_health.data}
              <p class="card-note">
                Observed on {report.sections.daily_health.data.observed_days} days.
              </p>
              <ul class="sub-list">
                {#each report.sections.daily_health.data.metric_averages as metric}<li
                  >
                    <span>{metric.metric}</span><span
                      >{metric.value}
                      {metric.unit} · {metric.observed_days}d</span
                    >
                  </li>{/each}
              </ul>
            {:else}<p class="empty">
                {sectionState(report.sections.daily_health.state)}
              </p>{/if}
          </section>

          <section class="report-card panel" aria-labelledby="media-title">
            <header>
              <p class="eyebrow">Media</p>
              <h2 id="media-title">Library movement</h2>
            </header>
            {#if report.sections.media.state === "available" && report.sections.media.data}
              <div class="stats">
                <div>
                  <b>{report.sections.media.data.event_count}</b><span
                    >events</span
                  >
                </div>
                <div>
                  <b>{report.sections.media.data.completed_count}</b><span
                    >completed</span
                  >
                </div>
                <div>
                  <b>{report.sections.media.data.average_rating ?? "—"}</b><span
                    >average rating</span
                  >
                </div>
              </div>
              <ul class="sub-list">
                {#each report.sections.media.data.completed_items.slice(0, 5) as item}<li
                  >
                    <span>{item.title}</span><span>{item.media_type}</span>
                  </li>{/each}
              </ul>
            {:else}<p class="empty">
                {sectionState(report.sections.media.state)}
              </p>{/if}
          </section>

          <section
            class="report-card panel expenses-card"
            aria-labelledby="reports-expenses-title"
          >
            <header>
              <p class="eyebrow">Expenses</p>
              <h2 id="reports-expenses-title">Ledger movement</h2>
            </header>
            {#if report.sections.expenses.state === "available" && report.sections.expenses.data}
              <div class="stats">
                <div>
                  <b>{report.sections.expenses.data.expense_count}</b><span
                    >records</span
                  >
                </div>
              </div>
              <ul class="sub-list">
                {#each report.sections.expenses.data.totals_by_currency as total}<li
                  >
                    <span>{total.currency}</span><span
                      >{formatMoney(
                        total.amount_minor,
                        total.currency,
                        total.currency_exponent,
                      )} · {total.expense_count}</span
                    >
                  </li>{/each}
              </ul>
              <p class="card-note">Category breakdown</p>
              <ul class="sub-list">
                {#each report.sections.expenses.data.by_category as category}<li
                  >
                    <span>{category.category}</span><span
                      >{formatMoney(
                        category.amount_minor,
                        category.currency,
                        category.currency_exponent,
                      )} · {category.expense_count}</span
                    >
                  </li>{/each}
              </ul>
            {:else}<p class="empty">
                {sectionState(report.sections.expenses.state)}
              </p>{/if}
          </section>
        </div>

        <section class="analysis-grid" aria-label="Expense analysis">
          <article class="analysis-card panel">
            <header>
              <p class="eyebrow">Aggregation</p>
              <h2>Spending by category</h2>
            </header>
            {#if categoryTotals.length}
              <BarChart
                categories={categoryTotals.map((item) => item.category)}
                primary={{
                  name: primaryCurrency,
                  values: categoryTotals.map((item) => item.amount_minor),
                  color: "var(--accent)",
                  formatter: (value) =>
                    formatMoney(value, primaryCurrency, primaryExponent),
                }}
                orientation="horizontal"
                height={280}
              />
            {:else}
              <p class="empty">No expense records to aggregate.</p>
            {/if}
          </article>

          <article class="analysis-card panel">
            <header>
              <p class="eyebrow">Recent months</p>
              <h2>{primaryCurrency} movement</h2>
            </header>
            {#if recentTotals.length}
              <BarChart
                categories={recentTotals.map((item) => item.label)}
                primary={{
                  name: primaryCurrency,
                  values: recentTotals.map((item) => item.amount),
                  color: "var(--accent)",
                  formatter: (value) =>
                    formatMoney(value, primaryCurrency, primaryExponent),
                }}
                activeIndex={recentTotals.findIndex(
                  (item) => item.month === month,
                )}
                height={280}
              />
            {:else}
              <p class="empty">No recent months available.</p>
            {/if}
          </article>

          <article class="comparison-card panel">
            <p class="eyebrow">Month over month</p>
            <h2>{comparisonLabel}</h2>
            <p>
              {formatMoney(currentTotal, primaryCurrency, primaryExponent)} in
              {formatMonth(month)} versus
              {formatMoney(previousTotal, primaryCurrency, primaryExponent)} in
              {formatMonth(shiftMonth(month, -1))}.
            </p>
            <ul class="comparison-list">
              {#each recentTotals.slice().reverse() as item}
                <li>
                  <span>{formatMonth(item.month)}</span><strong
                    >{item.count} records</strong
                  >
                </li>
              {/each}
            </ul>
          </article>
        </section>
      {/if}
    </section>
  {/snippet}
</ThemeRouteRenderer>

<style>
  .reports-shell {
    display: grid;
    gap: 1.25rem;
  }
  h1,
  h2,
  p,
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
  .page-head,
  .period,
  .report-card header {
    display: flex;
    justify-content: space-between;
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
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .intro,
  .muted,
  .empty,
  .card-note,
  .generated {
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
  button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.35rem;
    min-height: 2.4rem;
    padding: 0 0.8rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface);
    color: var(--text);
    font: inherit;
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
  .period {
    align-items: center;
    max-width: 24rem;
    margin-inline: auto;
  }
  .period div {
    display: grid;
    gap: 0.2rem;
    text-align: center;
  }
  .period span {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .generated {
    font-size: 0.75rem;
    text-align: right;
  }
  .report-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
  }
  .report-card {
    display: grid;
    align-content: start;
    gap: 1rem;
  }
  .analysis-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
  }
  .analysis-card,
  .comparison-card {
    display: grid;
    align-content: start;
    gap: 0.9rem;
  }
  .analysis-card header {
    display: grid;
    gap: 0.25rem;
    padding-bottom: 0.8rem;
    border-bottom: 1px solid var(--border);
  }
  .comparison-card {
    grid-column: 1 / -1;
    background:
      linear-gradient(
        120deg,
        color-mix(in srgb, var(--accent) 12%, transparent),
        transparent 55%
      ),
      var(--tile-surface);
  }
  .comparison-card h2 {
    font-size: clamp(1.2rem, 3vw, 2rem);
  }
  .comparison-card > p:not(.eyebrow) {
    color: var(--text-muted);
    line-height: 1.5;
  }
  .comparison-list {
    display: grid;
    grid-template-columns: repeat(6, minmax(0, 1fr));
    gap: 0.5rem;
    padding: 0;
    list-style: none;
  }
  .comparison-list li {
    display: grid;
    gap: 0.25rem;
    padding: 0.65rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
  }
  .comparison-list span {
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  .comparison-list strong {
    font-size: 0.8rem;
  }
  .report-card header {
    align-items: start;
    padding-bottom: 0.8rem;
    border-bottom: 1px solid var(--border);
  }
  .stats {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 0.7rem;
  }
  .stats div {
    display: grid;
    gap: 0.2rem;
    min-width: 0;
  }
  .stats b {
    font-size: 1.25rem;
    overflow-wrap: anywhere;
  }
  .stats span {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .sub-list {
    display: grid;
    gap: 0.4rem;
    padding: 0;
    list-style: none;
  }
  .sub-list li {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    padding-bottom: 0.4rem;
    border-bottom: 1px solid color-mix(in srgb, var(--border) 70%, transparent);
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  .sub-list li span:last-child {
    text-align: right;
  }
  .card-note {
    font-size: 0.8rem;
  }
  @media (max-width: 760px) {
    .page-head {
      align-items: start;
      flex-direction: column;
    }
    .report-grid {
      grid-template-columns: 1fr;
    }
    .analysis-grid {
      grid-template-columns: 1fr;
    }
    .comparison-list {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .stats b {
      font-size: 1rem;
    }
  }
</style>
