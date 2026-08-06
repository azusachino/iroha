import { describe, it, expect } from "vitest";
import {
  boundPercent,
  formatDistance,
  formatDuration,
  formatPercent,
  formatProgressCount,
  progressPercent,
  formatPace,
  formatElevation,
  formatHr,
  formatDate,
  formatDateOnly,
  formatSport,
  formatSwimmingPace,
} from "./format";

const DASH = "—";

describe("formatSport", () => {
  it("returns dash for missing input", () => {
    expect(formatSport(undefined)).toBe(DASH);
    expect(formatSport("")).toBe(DASH);
  });

  it("title-cases short lowercase codes", () => {
    expect(formatSport("run")).toBe("Run");
    expect(formatSport("ride")).toBe("Ride");
  });

  it("splits Apple PascalCase into words", () => {
    expect(formatSport("FitnessGaming")).toBe("Fitness Gaming");
    expect(formatSport("HighIntensityIntervalTraining")).toBe(
      "High Intensity Interval Training",
    );
  });

  it("normalizes underscores", () => {
    expect(formatSport("functional_strength_training")).toBe(
      "Functional Strength Training",
    );
  });
});

describe("formatDate", () => {
  it("renders yyyy-MM-dd HH:mm:ss in the given timezone", () => {
    expect(formatDate("2026-07-09T00:39:38Z", "UTC")).toBe(
      "2026-07-09 00:39:38",
    );
  });
});

describe("formatDateOnly is yyyy-MM-dd", () => {
  it("renders date only in the given timezone", () => {
    expect(formatDateOnly("2026-07-09T00:39:38Z", "UTC")).toBe("2026-07-09");
  });
});

describe("formatDistance", () => {
  it("returns em dash for undefined", () => {
    expect(formatDistance(undefined)).toBe(DASH);
  });

  it("returns em dash for null", () => {
    expect(formatDistance(null as any)).toBe(DASH);
  });

  it("rounds meters under 1000", () => {
    expect(formatDistance(0)).toBe("0 m");
    expect(formatDistance(100)).toBe("100 m");
    expect(formatDistance(999)).toBe("999 m");
    expect(formatDistance(999.5)).toBe("1000 m");
  });

  it("formats kilometers with 2 decimals for values >= 1000", () => {
    expect(formatDistance(1000)).toBe("1.00 km");
    expect(formatDistance(1500)).toBe("1.50 km");
    expect(formatDistance(1234.567)).toBe("1.23 km");
    expect(formatDistance(10000)).toBe("10.00 km");
  });
});

describe("formatDuration", () => {
  it("returns em dash for undefined", () => {
    expect(formatDuration(undefined)).toBe(DASH);
  });

  it("returns em dash for null", () => {
    expect(formatDuration(null as any)).toBe(DASH);
  });

  it("formats seconds as M:SS when hours = 0", () => {
    expect(formatDuration(0)).toBe("0:00");
    expect(formatDuration(30)).toBe("0:30");
    expect(formatDuration(59)).toBe("0:59");
    expect(formatDuration(60)).toBe("1:00");
    expect(formatDuration(65)).toBe("1:05");
    expect(formatDuration(119)).toBe("1:59");
  });

  it("formats as H:MM:SS when hours > 0", () => {
    expect(formatDuration(3600)).toBe("1:00:00");
    expect(formatDuration(3661)).toBe("1:01:01");
    expect(formatDuration(7323)).toBe("2:02:03");
    expect(formatDuration(36000)).toBe("10:00:00");
  });

  it("rounds seconds correctly", () => {
    expect(formatDuration(30.4)).toBe("0:30");
    expect(formatDuration(30.5)).toBe("0:31");
    expect(formatDuration(3599.5)).toBe("1:00:00");
  });

  it("pads minutes and seconds with zeros", () => {
    expect(formatDuration(125)).toBe("2:05");
    expect(formatDuration(1.5)).toBe("0:02");
  });
});

describe("formatPace", () => {
  it("returns em dash for undefined", () => {
    expect(formatPace(undefined)).toBe(DASH);
  });

  it("returns em dash for null", () => {
    expect(formatPace(null as any)).toBe(DASH);
  });

  it("returns em dash for non-finite values", () => {
    expect(formatPace(Infinity)).toBe(DASH);
    expect(formatPace(-Infinity)).toBe(DASH);
    expect(formatPace(NaN)).toBe(DASH);
  });

  it("returns em dash for zero and negative values", () => {
    expect(formatPace(0)).toBe(DASH);
    expect(formatPace(-100)).toBe(DASH);
  });

  it("formats seconds per km as M:SS /km", () => {
    expect(formatPace(60)).toBe("1:00 /km");
    expect(formatPace(120)).toBe("2:00 /km");
    expect(formatPace(300)).toBe("5:00 /km");
  });

  it("rounds seconds correctly", () => {
    expect(formatPace(65.4)).toBe("1:05 /km");
    expect(formatPace(65.5)).toBe("1:06 /km");
  });

  it("pads seconds with zeros", () => {
    expect(formatPace(305)).toBe("5:05 /km");
    expect(formatPace(303)).toBe("5:03 /km");
  });
});

describe("formatElevation", () => {
  it("returns em dash for undefined", () => {
    expect(formatElevation(undefined)).toBe(DASH);
  });

  it("returns em dash for null", () => {
    expect(formatElevation(null as any)).toBe(DASH);
  });

  it("rounds and formats meters", () => {
    expect(formatElevation(0)).toBe("0 m");
    expect(formatElevation(100)).toBe("100 m");
    expect(formatElevation(1234.5)).toBe("1235 m");
    expect(formatElevation(1234.4)).toBe("1234 m");
  });
});

describe("formatHr", () => {
  it("returns em dash for undefined", () => {
    expect(formatHr(undefined)).toBe(DASH);
  });

  it("returns em dash for null", () => {
    expect(formatHr(null as any)).toBe(DASH);
  });

  it("rounds and formats beats per minute", () => {
    expect(formatHr(0)).toBe("0 bpm");
    expect(formatHr(120)).toBe("120 bpm");
    expect(formatHr(120.4)).toBe("120 bpm");
    expect(formatHr(120.5)).toBe("121 bpm");
    expect(formatHr(180)).toBe("180 bpm");
  });
});

describe("formatDate", () => {
  it("returns em dash for undefined", () => {
    expect(formatDate(undefined)).toBe(DASH);
  });

  it("returns em dash for empty string", () => {
    expect(formatDate("")).toBe(DASH);
  });

  it("returns iso string for invalid ISO date", () => {
    expect(formatDate("not-a-date")).toBe("not-a-date");
  });
});

describe("formatDateOnly", () => {
  it("returns em dash for undefined", () => {
    expect(formatDateOnly(undefined)).toBe(DASH);
  });

  it("returns em dash for empty string", () => {
    expect(formatDateOnly("")).toBe(DASH);
  });

  it("returns iso string for invalid ISO date", () => {
    expect(formatDateOnly("not-a-date")).toBe("not-a-date");
  });
});

describe("formatPercent", () => {
  it("rounds and bounds percentages for display", () => {
    expect(formatPercent(93.54838709677419)).toBe("94%");
    expect(formatPercent(-4)).toBe("0%");
    expect(formatPercent(104)).toBe("100%");
    expect(boundPercent(42.5)).toBe(42.5);
  });

  it("returns an em dash for missing or invalid percentages", () => {
    expect(formatPercent(undefined)).toBe(DASH);
    expect(formatPercent(NaN)).toBe(DASH);
  });
});

describe("formatProgressCount", () => {
  it("shows a done/all count when the total is known", () => {
    expect(formatProgressCount(46, 120, "chapters")).toBe("46/120 chapters");
    expect(formatProgressCount(9, 9, "seasons")).toBe("9/9 seasons");
  });

  it("shows just the done count when the total is unknown -- not a fabricated percentage", () => {
    expect(formatProgressCount(46, undefined, "chapters")).toBe("46 chapters");
    expect(formatProgressCount(46, null, "chapters")).toBe("46 chapters");
    expect(formatProgressCount(46, 0, "chapters")).toBe("46 chapters");
  });

  it("omits the unit when none is given", () => {
    expect(formatProgressCount(46, 120)).toBe("46/120");
    expect(formatProgressCount(46)).toBe("46");
  });

  it("returns an em dash only when there's no position at all", () => {
    expect(formatProgressCount(undefined, 120, "chapters")).toBe(DASH);
    expect(formatProgressCount(null, undefined)).toBe(DASH);
  });

  it("treats a completed item's position as its own total when no total was recorded", () => {
    // Real case: Bangumi reports position=10 with status=completed but no
    // total for Rick and Morty Season 9 -- "10 episodes" alone reads as
    // ambiguous progress, not a finished season.
    expect(formatProgressCount(10, undefined, "episodes", "completed")).toBe(
      "10/10 episodes",
    );
  });

  it("does not infer a total from position for a non-completed status", () => {
    expect(formatProgressCount(10, undefined, "episodes", "in_progress")).toBe(
      "10 episodes",
    );
    expect(formatProgressCount(10, undefined, "episodes", "abandoned")).toBe(
      "10 episodes",
    );
  });

  it("prefers a real recorded total over the completed inference", () => {
    expect(formatProgressCount(10, 12, "episodes", "completed")).toBe(
      "10/12 episodes",
    );
  });
});

describe("progressPercent", () => {
  it("uses an explicit percent when present", () => {
    expect(progressPercent("in_progress", 5, 10, 42)).toBe(42);
  });

  it("derives a percent from position/total when no explicit percent exists", () => {
    expect(progressPercent("in_progress", 5, 10, undefined)).toBe(50);
  });

  it("treats a completed item's position as its own total, filling the bar", () => {
    expect(progressPercent("completed", 10, undefined, undefined)).toBe(100);
  });

  it("returns 0 only when nothing is known, and never fabricates a fill for unfinished progress", () => {
    expect(progressPercent("in_progress", 46, undefined, undefined)).toBe(0);
    expect(progressPercent(undefined, undefined, undefined, undefined)).toBe(0);
  });
});

describe("formatSwimmingPace", () => {
  it("formats derived open-water pace per 100 metres", () => {
    expect(formatSwimmingPace(1000, 1500)).toBe("2:30 /100m");
  });

  it("returns an em dash when distance or duration is unavailable", () => {
    expect(formatSwimmingPace(undefined, 1500)).toBe(DASH);
    expect(formatSwimmingPace(1000, undefined)).toBe(DASH);
  });
});
