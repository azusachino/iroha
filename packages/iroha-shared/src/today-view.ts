import type { Activity } from "./activity";
import type { DailyRow } from "./daily";
import type { MediaHomeEvent } from "./media";
import type { SleepSession } from "./sleep";
import type { DesignLanguage } from "./themes";

export type TodayThemeProps = {
  dayLabel: string;
  day: string;
  dRow: DailyRow | undefined;
  mainNight: SleepSession | undefined;
  acts: Activity[];
  mediaEvents: MediaHomeEvent[];
  theme: DesignLanguage;
  onOpenActivity: (id: string) => void;
  onOpenMedia: (id: string) => void;
};
