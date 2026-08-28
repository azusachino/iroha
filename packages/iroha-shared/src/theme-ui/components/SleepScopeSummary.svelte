<script lang="ts">
  import type { DesignLanguage } from "../../theme/themes";
  import type { SleepAggregateBucket } from "../../domain/sleep";
  import { formatCanonicalMonth } from "../../format/format";

  let {
    summary = null,
    scope = "",
    theme,
  }: {
    summary?: SleepAggregateBucket | null;
    scope?: string;
    theme: DesignLanguage;
  } = $props();

  function count(value: number | undefined): string {
    return value == null || !Number.isFinite(value)
      ? "—"
      : value.toLocaleString();
  }

  function scopeLabel(value: string): string {
    if (/^\d{4}-\d{2}$/.test(value)) return formatCanonicalMonth(value);
    return value || "Lifetime";
  }
</script>

<section
  class="sleep-scope-summary"
  data-theme={theme}
  aria-label={`Sleep totals for ${scopeLabel(scope)}`}
>
  <header>
    <div>
      <p class="eyebrow">Canonical sleep totals</p>
      <h2>{scopeLabel(scope)}</h2>
    </div>
    <span>Rollup owned by Iroha</span>
  </header>
  <div class="stats-grid">
    <div>
      <span>Total sessions</span>
      <strong>{count(summary?.session_count)}</strong>
    </div>
    <div>
      <span>Main sleep</span>
      <strong>{count(summary?.main_sleep_count)}</strong>
    </div>
    <div>
      <span>Naps</span>
      <strong>{count(summary?.nap_count)}</strong>
    </div>
    <div>
      <span>Wake dates</span>
      <strong>{count(summary?.observed_wake_dates)}</strong>
    </div>
  </div>
  <footer class="kind-legend" aria-label="Sleep session types">
    <span><i class="main-dot"></i>Main sleep</span>
    <span><i class="nap-dot"></i>Nap</span>
  </footer>
</section>

<style>
  .sleep-scope-summary {
    display: grid;
    gap: 0.8rem;
    padding: 1rem 1.15rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }

  .sleep-scope-summary[data-theme="atlas"] {
    border-width: 2px;
    border-radius: 2px;
    background-image: linear-gradient(
      color-mix(in srgb, var(--accent) 7%, transparent) 1px,
      transparent 1px
    );
    background-size: 100% 0.8rem;
  }

  .sleep-scope-summary[data-theme="field-journal"] {
    border-style: dashed;
    border-radius: 0;
  }

  .sleep-scope-summary[data-theme="phenology"] {
    border-radius: 1.25rem;
  }

  .sleep-scope-summary[data-theme="cadence"] {
    border-inline-width: 3px;
  }

  .sleep-scope-summary[data-theme="archive"] {
    border-width: 3px;
    border-radius: 0;
  }

  .sleep-scope-summary[data-theme="grapher"] {
    border-radius: 2px;
    border-bottom-width: 3px;
  }

  header {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 1rem;
  }

  header h2,
  header p {
    margin: 0;
  }

  header h2 {
    font-size: 1.05rem;
  }

  header > span {
    color: var(--text-muted);
    font-size: 0.72rem;
  }

  .eyebrow {
    margin: 0 0 0.25rem;
    color: var(--accent);
    font-size: 0.64rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    border-top: 1px solid var(--border);
  }

  .stats-grid div {
    display: grid;
    gap: 0.3rem;
    min-width: 0;
    padding: 0.7rem 0.75rem 0.1rem 0;
    border-right: 1px solid var(--border);
  }

  .stats-grid div:not(:first-child) {
    padding-left: 0.75rem;
  }

  .stats-grid div:last-child {
    border-right: 0;
  }

  .stats-grid span {
    color: var(--text-muted);
    font-size: 0.66rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }

  .stats-grid strong {
    color: var(--text);
    font-size: clamp(1.15rem, 2.4vw, 1.8rem);
    font-variant-numeric: tabular-nums;
  }

  .kind-legend {
    display: flex;
    flex-wrap: wrap;
    gap: 0.8rem;
    color: var(--text-muted);
    font-size: 0.7rem;
  }

  .kind-legend span {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
  }

  .kind-legend i {
    width: 0.55rem;
    height: 0.55rem;
    border-radius: 50%;
  }

  .main-dot {
    background: var(--accent);
  }

  .nap-dot {
    background: var(--accent-2);
  }

  @media (max-width: 640px) {
    header {
      display: grid;
      gap: 0.2rem;
    }

    .stats-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .stats-grid div:nth-child(2) {
      border-right: 0;
    }

    .stats-grid div:nth-child(3),
    .stats-grid div:nth-child(4) {
      border-top: 1px solid var(--border);
    }

    .stats-grid div:nth-child(3) {
      padding-left: 0;
    }
  }
</style>
