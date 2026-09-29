import type { ActivityActiveDay } from "../../domain/activity";

export interface ActivityHeatmapCell {
  date: string | null;
  count: number;
  level: number;
}

export interface ActivityHeatmapModel {
  weeks: ActivityHeatmapCell[][];
  activeDayCount: number;
  totalActivityCount: number;
}

const DAYS_IN_YEAR = 365;

function shiftDay(day: string, amount: number): string {
  const date = new Date(`${day}T00:00:00Z`);
  date.setUTCDate(date.getUTCDate() + amount);
  return date.toISOString().slice(0, 10);
}

function mondayIndex(day: string): number {
  return (new Date(`${day}T00:00:00Z`).getUTCDay() + 6) % 7;
}

function levelFor(count: number): number {
  if (count <= 0) return 0;
  if (count === 1) return 1;
  if (count === 2) return 2;
  if (count <= 4) return 3;
  return 4;
}

export function buildActivityHeatmap(
  activeDays: ActivityActiveDay[],
  endDay: string,
): ActivityHeatmapModel {
  const startDay = shiftDay(endDay, -(DAYS_IN_YEAR - 1));
  const counts = new Map<string, number>();
  for (const { day, activity_count } of activeDays) {
    if (day >= startDay && day <= endDay) {
      counts.set(day, Math.max(0, activity_count));
    }
  }
  const cells: ActivityHeatmapCell[] = [];

  for (let offset = 0; offset < DAYS_IN_YEAR; offset++) {
    const date = shiftDay(startDay, offset);
    const count = counts.get(date) ?? 0;
    cells.push({ date, count, level: levelFor(count) });
  }

  const padding = mondayIndex(startDay);
  const calendar: ActivityHeatmapCell[] = [
    ...Array.from({ length: padding }, () => ({
      date: null,
      count: 0,
      level: 0,
    })),
    ...cells,
  ];
  while (calendar.length % 7 !== 0) {
    calendar.push({ date: null, count: 0, level: 0 });
  }

  return {
    weeks: Array.from({ length: calendar.length / 7 }, (_, index) =>
      calendar.slice(index * 7, index * 7 + 7),
    ),
    activeDayCount: cells.filter((cell) => cell.count > 0).length,
    totalActivityCount: cells.reduce((total, cell) => total + cell.count, 0),
  };
}
