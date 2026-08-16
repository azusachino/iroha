import type { Activity } from "../domain/activity";
import type { DailyRow } from "../domain/daily";
import type { MediaChange, MediaHomeEvent } from "../domain/media";
import type { SleepSession } from "../domain/sleep";
import type { DesignLanguage } from "../theme/themes";

export type TodayThemeProps = {
  dayLabel: string;
  day: string;
  dRow: DailyRow | undefined;
  mainNight: SleepSession | undefined;
  acts: Activity[];
  mediaEvents: MediaHomeEvent[];
  mediaUpdates: MediaChange[];
  theme: DesignLanguage;
  onOpenActivity: (id: string) => void;
  onOpenMedia: (id: string) => void;
};
