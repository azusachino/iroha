import type { Snippet } from "svelte";
import type { DesignLanguage } from "../theme/themes";
import type { SleepAggregateBucket, SleepSession } from "../domain/sleep";

export type SleepThemeProps = {
  sessions: SleepSession[];
  selected: SleepSession | null;
  averageAsleep: number | null;
  averageEfficiency: number | null;
  sessionsLoading?: boolean;
  sessionsReady?: boolean;
  sessionsError?: string | null;
  recordsScope?: string;
  onRetrySessions?: () => Promise<void>;
  onOpenDetail: (session: SleepSession) => void;
  sleepSummary?: SleepAggregateBucket | null;
  rollupBuckets?: SleepAggregateBucket[];
  rollupGranularity?: "month" | "year";
  rollupScope?: string;
  theme: DesignLanguage;
  children?: Snippet;
};
