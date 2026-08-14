import { describe, expect, it } from "vitest";
import {
  sleepStageColor,
  sleepStageLabel,
  SLEEP_STAGE_DEFINITIONS,
} from "./sleep-stages";

describe("sleep stage presentation", () => {
  it("keeps the Apple Health stage vocabulary explicit", () => {
    expect(Object.keys(SLEEP_STAGE_DEFINITIONS)).toEqual([
      "core",
      "deep",
      "rem",
      "awake",
      "in_bed",
      "asleep_unspecified",
    ]);
    expect(sleepStageLabel("rem")).toBe("REM");
    expect(sleepStageLabel("awake")).toBe("Awake");
  });

  it("does not reuse the in-bed grey for unclassified sleep", () => {
    expect(sleepStageColor("asleep_unspecified")).not.toBe(
      sleepStageColor("in_bed"),
    );
    expect(sleepStageColor("unknown_stage")).toBe(
      sleepStageColor("asleep_unspecified"),
    );
  });
});
