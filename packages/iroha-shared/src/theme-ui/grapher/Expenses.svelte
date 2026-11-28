<script lang="ts">
  import BarChart from "../components/BarChart.svelte";
  import ExpenseLedger from "../components/ExpenseLedger.svelte";
  import FocalBand, {
    type FocalPrimaryMetric,
    type FocalSecondaryItem,
  } from "../../components/FocalBand.svelte";
  import MetricMetadata from "../../components/MetricMetadata.svelte";
  import MetricTable from "../../components/MetricTable.svelte";
  import PanelFrame from "../../components/PanelFrame.svelte";
  import { panelCsv } from "../../components/metric-panel";
  import type { ExpenseThemeProps } from "../../view-contracts/expense-view";
  import { formatExpenseDay } from "../../view-contracts/expense-view";

  let {
    section = "all",
    month,
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
    onSelect,
    onRemove,
    formatMoney,
  }: ExpenseThemeProps = $props();

  let chartView = $state<"chart" | "table">("chart");

  const activeCurrencies = $derived(
    currencyTotals.filter((total) => total.amountMinor != null),
  );
  const emptyCurrencies = $derived(
    currencyTotals.filter((total) => total.amountMinor == null),
  );
  const primaryTotal = $derived(
    activeCurrencies.find((total) => total.currency === primaryCurrency) ??
      activeCurrencies[0] ??
      null,
  );

  const primaryMetric = $derived<FocalPrimaryMetric>(
    primaryTotal
      ? {
          label: `Period spend · ${primaryTotal.currency}`,
          value: formatMoney(
            primaryTotal.amountMinor!,
            primaryTotal.currency,
            primaryTotal.exponent,
          ),
          detail:
            primaryTotal.count == null
              ? "Record count unavailable"
              : `${primaryTotal.count} ${primaryTotal.count === 1 ? "record" : "records"}`,
        }
      : {
          label: `Period spend · ${primaryCurrency}`,
          value: "—",
          detail: "Record count unavailable",
        },
  );

  const secondaryItems = $derived<FocalSecondaryItem[]>(
    activeCurrencies
      .filter((total) => total.currency !== primaryTotal?.currency)
      .map((total) => ({
        label: total.currency,
        value: formatMoney(total.amountMinor!, total.currency, total.exponent),
        detail:
          total.count == null
            ? undefined
            : `${total.count} ${total.count === 1 ? "record" : "records"}`,
      })),
  );

  const emptyCurrenciesNote = $derived(
    emptyCurrencies.length > 0
      ? `${emptyCurrencies.length} ${emptyCurrencies.length === 1 ? "currency" : "currencies"} empty: ${emptyCurrencies.map((total) => total.currency).join(", ")}`
      : undefined,
  );

  const hasChartRows = $derived(
    dailyPanel.rows.length > 0 || categoryPanel.rows.length > 0,
  );

  function downloadSpendingCsv() {
    const sections = [
      dailyPanel.rows.length > 0
        ? panelCsv(
            dailyPanel.metricId,
            dailyPanel.unit,
            dailyPanel.rowHeader,
            dailyPanel.rows,
          )
        : "",
      categoryPanel.rows.length > 0
        ? panelCsv(
            categoryPanel.metricId,
            categoryPanel.unit,
            categoryPanel.rowHeader,
            categoryPanel.rows,
          )
        : "",
    ].filter(Boolean);
    if (!sections.length) return;
    const blob = new Blob([sections.join("\n")], {
      type: "text/csv;charset=utf-8",
    });
    const link = document.createElement("a");
    link.href = URL.createObjectURL(blob);
    link.download = `${dailyPanel.metricId}${month ? `-${month}` : ""}.csv`;
    link.click();
    URL.revokeObjectURL(link.href);
  }
</script>

<section class="grapher-expenses" aria-label="Expense evidence">
  {#if section !== "ledger"}
    <FocalBand
      heading="Spending summary"
      primary={primaryMetric}
      secondary={secondaryItems}
      empty={emptyCurrenciesNote}
    >
      {#snippet controls()}
        <span class="scope-statement">Observed period: {month}</span>
      {/snippet}
    </FocalBand>

    <div class="chart-controls-bar">
      <MetricMetadata
        unit={dailyPanel.unit}
        method={dailyPanel.method}
        coverage={dailyPanel.coverage}
        sourceKinds={dailyPanel.sourceKinds}
      />
      <div class="chart-actions" role="group" aria-label="Spending charts view">
        <button
          type="button"
          aria-pressed={chartView === "chart"}
          onclick={() => (chartView = "chart")}>Chart</button
        >
        <button
          type="button"
          aria-pressed={chartView === "table"}
          onclick={() => (chartView = "table")}>Table</button
        >
        <button
          type="button"
          onclick={downloadSpendingCsv}
          disabled={!hasChartRows}>CSV</button
        >
      </div>
    </div>

    <PanelFrame label="Daily spend">
      <h2>Daily spend · {primaryCurrency}</h2>
      {#if chartView === "chart"}
        <BarChart
          categories={dailyTotals.map(([day]) => formatExpenseDay(day))}
          primary={{
            name: primaryCurrency,
            values: dailyTotals.map(([, amount]) => amount),
            color: "var(--accent)",
            axis: {
              unit: primaryCurrency,
              scale: 10 ** -primaryExponent,
              minimumInterval: 1,
            },
            formatter: (value) =>
              formatMoney(value, primaryCurrency, primaryExponent),
          }}
          primaryType="line"
          height={300}
        />
      {:else}
        <MetricTable
          rows={dailyPanel.rows}
          unit={dailyPanel.unit}
          rowHeader={dailyPanel.rowHeader}
          numericAlign
        />
      {/if}
    </PanelFrame>

    <PanelFrame label="Spend by category">
      <h2>Spend by category · {primaryCurrency}</h2>
      {#if chartView === "chart"}
        <BarChart
          categories={categoryTotals.map((item) => item.category)}
          primary={{
            name: primaryCurrency,
            values: categoryTotals.map((item) => item.amount),
            color: "var(--accent-2)",
            axis: {
              unit: primaryCurrency,
              scale: 10 ** -primaryExponent,
              minimumInterval: 1,
            },
            formatter: (value) =>
              formatMoney(value, primaryCurrency, primaryExponent),
          }}
          orientation="horizontal"
          categorical
          height={250}
        />
      {:else}
        <MetricTable
          rows={categoryPanel.rows}
          unit={categoryPanel.unit}
          rowHeader={categoryPanel.rowHeader}
          numericAlign
        />
      {/if}
    </PanelFrame>
  {/if}
  {#if section !== "spending"}
    <ExpenseLedger
      {expenses}
      {selected}
      {selectedId}
      {detailLoading}
      {onSelect}
      {onRemove}
      {formatMoney}
    />
  {/if}
</section>

<style>
  .grapher-expenses {
    --accent: var(--mark-teal);
    display: grid;
    gap: var(--space-4);
    min-width: 0;
  }
  .grapher-expenses :global(.focal-primary-value),
  .grapher-expenses :global(.focal-secondary-item dd > span:first-child) {
    font-family: var(--font-mono);
  }
  .scope-statement {
    color: var(--text-muted);
    font-size: var(--type-caption);
    font-family: var(--font-mono);
    font-variant-numeric: tabular-nums;
  }
  .chart-controls-bar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2) var(--space-4);
    min-width: 0;
  }
  .chart-actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1);
  }
  .chart-actions button {
    min-height: var(--control-height);
    padding: var(--space-1) var(--space-3);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: var(--surface-2);
    color: var(--text-muted);
    font: inherit;
    font-size: var(--type-caption);
    letter-spacing: 0.06em;
    text-transform: uppercase;
    cursor: pointer;
    transition:
      border-color var(--motion-micro),
      color var(--motion-micro);
  }
  .chart-actions button[aria-pressed="true"] {
    border-color: var(--accent);
    color: var(--accent);
  }
  .chart-actions button:disabled {
    cursor: default;
    opacity: 0.5;
  }
  .chart-actions button:focus-visible {
    outline: 2px solid var(--color-focus);
    outline-offset: 2px;
  }
  h2 {
    margin: 0;
    font-size: var(--type-title-small);
  }
</style>
