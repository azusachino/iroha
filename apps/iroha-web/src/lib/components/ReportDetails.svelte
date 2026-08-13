<script lang="ts">
  import type { MonthlyReport } from "$lib/api";
  import { formatMetricValue } from "$lib/format";
  let {
    report,
    formatDuration,
    formatMoney,
  }: {
    report: MonthlyReport;
    formatDuration: (seconds: number) => string;
    formatMoney: (
      amountMinor: number,
      currency: string,
      exponent: number,
    ) => string;
  } = $props();
  const empty = (state: string) =>
    state === "empty" ? "No canonical records for this month." : "";
</script>

<section class="details" aria-label="Monthly report details">
  <div class="detail-grid">
    <article>
      <p class="eyebrow">Movement</p>
      <h3>Activity</h3>
      {#if report.sections.movement.data}<div class="facts">
          <b
            >{report.sections.movement.data.activity_count}<small
              >activities</small
            ></b
          ><b
            >{(report.sections.movement.data.distance_m / 1000).toFixed(
              1,
            )}<small>km</small></b
          ><b
            >{formatDuration(report.sections.movement.data.duration_s)}<small
              >duration</small
            ></b
          >
        </div>
        <ul>
          {#each report.sections.movement.data.by_sport as item}<li>
              <span>{item.sport}</span><span
                >{item.activity_count} · {(item.distance_m / 1000).toFixed(1)} km</span
              >
            </li>{/each}
        </ul>{:else}<p class="empty">
          {empty(report.sections.movement.state)}
        </p>{/if}
    </article>
    <article>
      <p class="eyebrow">Sleep</p>
      <h3>Rest</h3>
      {#if report.sections.sleep.data}<div class="facts">
          <b
            >{formatDuration(report.sections.sleep.data.average_asleep_s)}<small
              >average asleep</small
            ></b
          ><b
            >{Math.round(
              report.sections.sleep.data.average_efficiency * 100,
            )}%<small>efficiency</small></b
          ><b
            >{report.sections.sleep.data.session_count}<small>sessions</small
            ></b
          >
        </div>
        <p class="note">
          {report.sections.sleep.data.main_sleep_count} main sleeps · {report
            .sections.sleep.data.nap_count} naps
        </p>{:else}<p class="empty">
          {empty(report.sections.sleep.state)}
        </p>{/if}
    </article>
    <article>
      <p class="eyebrow">Daily health</p>
      <h3>Body signals</h3>
      {#if report.sections.daily_health.data}<p class="note">
          Observed on {report.sections.daily_health.data.observed_days} days.
        </p>
        <ul>
          {#each report.sections.daily_health.data.metric_averages as item}<li>
              <span>{item.metric}</span><span
                >{formatMetricValue(item.value, item.unit)}
                {item.unit} · {item.observed_days}d</span
              >
            </li>{/each}
        </ul>{:else}<p class="empty">
          {empty(report.sections.daily_health.state)}
        </p>{/if}
    </article>
    <article>
      <p class="eyebrow">Media</p>
      <h3>Library movement</h3>
      {#if report.sections.media.data}<div class="facts">
          <b>{report.sections.media.data.event_count}<small>events</small></b><b
            >{report.sections.media.data.completed_count}<small>completed</small
            ></b
          ><b
            >{report.sections.media.data.average_rating ?? "—"}<small
              >rating</small
            ></b
          >
        </div>
        <ul>
          {#each report.sections.media.data.completed_items.slice(0, 5) as item}<li
            >
              <span>{item.title}</span><span>{item.media_type}</span>
            </li>{/each}
        </ul>{:else}<p class="empty">
          {empty(report.sections.media.state)}
        </p>{/if}
    </article>
    <article class="expense-detail">
      <p class="eyebrow">Expenses</p>
      <h3>Canonical ledger</h3>
      {#if report.sections.expenses.data}<div class="facts">
          <b
            >{report.sections.expenses.data.expense_count}<small>records</small
            ></b
          ><b
            >{report.sections.expenses.data.totals_by_currency.length}<small
              >currencies</small
            ></b
          >
        </div>
        <ul>
          {#each report.sections.expenses.data.totals_by_currency as item}<li>
              <span>{item.currency}</span><span
                >{formatMoney(
                  item.amount_minor,
                  item.currency,
                  item.currency_exponent,
                )} · {item.expense_count}</span
              >
            </li>{/each}
        </ul>
        <p class="note">Category breakdown</p>
        <ul>
          {#each report.sections.expenses.data.by_category as item}<li>
              <span>{item.category}</span><span
                >{formatMoney(
                  item.amount_minor,
                  item.currency,
                  item.currency_exponent,
                )} · {item.expense_count}</span
              >
            </li>{/each}
        </ul>{:else}<p class="empty">
          {empty(report.sections.expenses.state)}
        </p>{/if}
    </article>
  </div>
</section>

<style>
  .details {
    display: grid;
    gap: 1rem;
  }
  .detail-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
  }
  article {
    display: grid;
    align-content: start;
    gap: 0.7rem;
    min-width: 0;
    border: 1px solid var(--border);
    padding: 1rem;
    background: var(--surface);
  }
  .expense-detail {
    grid-column: 1 / -1;
  }
  h3,
  p,
  ul {
    margin: 0;
  }
  h3 {
    font-size: 1.25rem;
    letter-spacing: -0.04em;
  }
  .eyebrow {
    color: var(--accent);
    font-size: 0.66rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .facts {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 0.65rem;
  }
  .facts b {
    display: grid;
    gap: 0.2rem;
    font-size: 1.35rem;
  }
  .facts small,
  .note,
  .empty,
  li {
    color: var(--text-muted);
    font-size: 0.76rem;
    font-weight: 400;
  }
  ul {
    display: grid;
    gap: 0.4rem;
    padding: 0;
    list-style: none;
  }
  li {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-bottom: 1px solid var(--border);
    padding-bottom: 0.4rem;
  }
  li span:last-child {
    text-align: right;
  }
  @media (max-width: 700px) {
    .detail-grid {
      grid-template-columns: 1fr;
    }
    .expense-detail {
      grid-column: auto;
    }
  }
</style>
