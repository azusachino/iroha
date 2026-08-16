<script lang="ts">
  import { todayInTimezone } from "@iroha/shared/format/date";
  import { IROHA_TIMEZONE } from "$lib/config";

  const WEEKDAYS = ["", "Mon", "", "Wed", "", "Fri", ""];
  const LEGEND_LEVELS = [0, 1, 2, 3, 4];

  let {
    dates,
    title = "Activity",
  }: {
    dates: string[];
    title?: string;
  } = $props();

  type DayCell = {
    key: string;
    day: number;
    count: number;
    level: number;
    label: string;
  };

  const today = $derived(todayInTimezone(new Date(), IROHA_TIMEZONE));
  const cells = $derived(buildCells(dates, today));
  const weeks = $derived(chunkWeeks(cells));
  const total = $derived(cells.reduce((sum, cell) => sum + cell.count, 0));

  function previousOrNextDay(day: string, delta: number): string {
    const date = new Date(`${day}T00:00:00Z`);
    date.setUTCDate(date.getUTCDate() + delta);
    return date.toISOString().slice(0, 10);
  }

  function keyFor(value: string): string | null {
    if (/^\d{4}-\d{2}-\d{2}$/.test(value)) return value;
    const parsed = new Date(value);
    return Number.isNaN(parsed.getTime()) ? null : todayInTimezone(parsed);
  }

  function buildCells(sourceDates: string[], endDay: string): DayCell[] {
    const counts = new Map<string, number>();
    for (const value of sourceDates) {
      const key = keyFor(value);
      if (!key) continue;
      counts.set(key, (counts.get(key) ?? 0) + 1);
    }

    const start = previousOrNextDay(endDay, -364);
    const cells: DayCell[] = [];
    for (let i = 0; i < 365; i++) {
      const key = previousOrNextDay(start, i);
      const count = counts.get(key) ?? 0;
      cells.push({
        key,
        day: new Date(`${key}T00:00:00Z`).getUTCDay(),
        count,
        level: heatLevel(count),
        label: `${key}: ${count} ${count === 1 ? "activity" : "activities"}`,
      });
    }
    return cells;
  }

  function heatLevel(count: number): number {
    if (count <= 0) return 0;
    if (count === 1) return 1;
    if (count === 2) return 2;
    if (count <= 4) return 3;
    return 4;
  }

  function chunkWeeks(sourceCells: DayCell[]): DayCell[][] {
    const result: DayCell[][] = [];
    let week: DayCell[] = [];
    for (const cell of sourceCells) {
      if (week.length === 0) {
        for (let i = 0; i < cell.day; i++) {
          week.push({
            key: `pad-start-${i}`,
            day: i,
            count: 0,
            level: 0,
            label: "",
          });
        }
      }
      week.push(cell);
      if (week.length === 7) {
        result.push(week);
        week = [];
      }
    }
    if (week.length > 0) {
      const startLength = week.length;
      for (let i = startLength; i < 7; i++) {
        week.push({
          key: `pad-end-${i}`,
          day: i,
          count: 0,
          level: 0,
          label: "",
        });
      }
      result.push(week);
    }
    return result;
  }
</script>

<section class="heatmap tile" aria-label={`${title} heatmap`}>
  <header class="heatmap-header">
    <div>
      <h2>{title}</h2>
      <p>{total} activities in the last year</p>
    </div>
    <div class="legend" aria-label="Activity count intensity">
      <span>Less</span>
      {#each LEGEND_LEVELS as level}
        <span class="cell legend-cell" data-level={level}></span>
      {/each}
      <span>More</span>
    </div>
  </header>

  <div class="heatmap-grid">
    <div class="weekday-labels" aria-hidden="true">
      {#each WEEKDAYS as day}
        <span>{day}</span>
      {/each}
    </div>
    <div class="weeks">
      {#each weeks as week}
        <div class="week">
          {#each week as cell}
            <span
              class="cell"
              class:empty={cell.label === ""}
              data-level={cell.level}
              title={cell.label}
              aria-label={cell.label || undefined}
            ></span>
          {/each}
        </div>
      {/each}
    </div>
  </div>
</section>

<style>
  .heatmap {
    padding: 1rem;
    overflow: hidden;
  }

  .heatmap-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 1rem;
  }

  h2 {
    margin: 0;
    font-size: 0.95rem;
  }

  p {
    margin: 0.25rem 0 0;
    color: var(--text-muted);
    font-size: 0.84rem;
  }

  .heatmap-grid {
    display: flex;
    gap: 0.5rem;
    overflow-x: auto;
    padding-bottom: 0.15rem;
  }

  .weekday-labels,
  .week {
    display: grid;
    grid-template-rows: repeat(7, 0.72rem);
    gap: 0.22rem;
  }

  .weekday-labels {
    min-width: 1.6rem;
    color: var(--text-muted);
    font-size: 0.62rem;
    line-height: 0.72rem;
    text-align: right;
  }

  .weeks {
    display: flex;
    gap: 0.22rem;
  }

  .cell {
    width: 0.72rem;
    height: 0.72rem;
    border-radius: 2px;
    background: var(--surface-2);
  }

  .cell.empty {
    visibility: hidden;
  }

  .cell[data-level="1"] {
    background: color-mix(in srgb, var(--sport-run) 28%, var(--surface-2));
  }

  .cell[data-level="2"] {
    background: color-mix(in srgb, var(--sport-run) 48%, var(--surface-2));
  }

  .cell[data-level="3"] {
    background: color-mix(in srgb, var(--sport-run) 68%, var(--surface-2));
  }

  .cell[data-level="4"] {
    background: var(--sport-run);
  }

  .legend {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    color: var(--text-muted);
    font-size: 0.72rem;
    white-space: nowrap;
  }

  .legend-cell {
    flex: 0 0 auto;
  }

  @media (max-width: 640px) {
    .heatmap-header {
      flex-direction: column;
    }
  }
</style>
