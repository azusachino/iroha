<script lang="ts">
  import PeriodDrill from "@iroha/shared/theme-ui/components/PeriodDrill.svelte";

  type Gran = "day" | "month" | "year";
  type Row = {
    label: string;
    period: string;
    days: number | null;
    move: number | null;
    exercise: number | null;
    stand: number | null;
    moveClosedPct: number | null;
    steps: number | null;
    distance: number | null;
    resting_hr: number | null;
    hrv_sdnn: number | null;
    spo2_avg: number | null;
    respiratory_rate: number | null;
    vo2max: number | null;
    body_mass_kg: number | null;
  };

  let {
    gran,
    aggregated,
    rows,
    format,
    onDrill,
  }: {
    gran: Gran;
    aggregated: boolean;
    rows: Row[];
    format: (value: number | null | undefined, digits: number) => string;
    onDrill: (period: string) => void;
  } = $props();
</script>

<table>
  <thead>
    <tr>
      <th class="l"
        >{gran === "day" ? "Day" : gran === "month" ? "Month" : "Year"}</th
      >
      {#if aggregated}<th>Days</th>{/if}
      <th>Move</th><th>Exer</th><th>Stand</th><th>Move ✓</th>
      <th>Steps{aggregated ? "/d" : ""}</th><th>Dist{aggregated ? "/d" : ""}</th
      >
      <th>rHR</th><th>HRV</th><th>SpO₂</th><th>Resp</th><th>VO₂</th><th>Mass</th
      >
    </tr>
  </thead>
  <tbody>
    {#each rows as row}
      <tr>
        <td class="l">
          {#if gran !== "day"}<PeriodDrill
              label={row.label}
              period={row.period}
              value={row.steps}
              {onDrill}
            />{:else}{row.label}{/if}
        </td>
        {#if aggregated}<td>{row.days}</td>{/if}
        <td>{format(row.move, 0)}</td>
        <td>{format(row.exercise, 0)}</td>
        <td>{format(row.stand, 0)}</td>
        <td
          >{row.moveClosedPct == null
            ? "—"
            : `${Math.round(row.moveClosedPct)}%`}</td
        >
        <td data-period-evidence>{format(row.steps, 0)}</td>
        <td>{format(row.distance, 1)}</td>
        <td>{format(row.resting_hr, 0)}</td>
        <td>{format(row.hrv_sdnn, 0)}</td>
        <td>{format(row.spo2_avg, 1)}</td>
        <td>{format(row.respiratory_rate, 1)}</td>
        <td>{format(row.vo2max, 1)}</td>
        <td>{format(row.body_mass_kg, 1)}</td>
      </tr>
    {/each}
  </tbody>
</table>

<style>
  table {
    width: 100%;
    border-collapse: collapse;
    font-variant-numeric: tabular-nums;
    font-size: 0.84rem;
  }
  th,
  td {
    padding: 0.5rem 0.6rem;
    text-align: right;
    white-space: nowrap;
  }
  th {
    color: var(--text-muted);
    font-weight: 600;
    border-bottom: 1px solid var(--border);
    position: sticky;
    top: 0;
    background: var(--surface);
  }
  td {
    color: var(--text);
  }
  tbody tr + tr td {
    border-top: 1px solid color-mix(in srgb, var(--border) 55%, transparent);
  }
  .l {
    text-align: left;
  }
</style>
