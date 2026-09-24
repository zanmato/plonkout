import { describe, it, expect } from "vitest";
import {
  buildExerciseHistory,
  getMaxWeight,
  getBestRepsAtWeight,
  getBestEstimated1RM,
  getLastEntry,
  formatEntrySets,
  type HistoryEntry,
} from "@/utils/exerciseHistory";
import type { Intensity, Workout } from "@/types/domain";
import { partial } from "../helpers/fixtures";

const workouts = partial<Workout[]>([
  {
    id: "w1",
    name: "Push Heavy",
    started: "2024-01-01T10:00:00",
    exercises: [
      {
        name: "Bench Press",
        intensity: "heavy",
        sets: [
          { type: "warmup", weight: 40, reps: 10 },
          { type: "regular", weight: 100, reps: 5 },
          { type: "regular", weight: 100, reps: 4 },
        ],
      },
    ],
  },
  {
    id: "w2",
    name: "Push Light",
    started: "2024-01-04T10:00:00",
    exercises: [
      {
        name: "Bench Press",
        intensity: "light",
        sets: [
          { type: "regular", weight: 70, reps: 10 },
          { type: "regular", weight: 70, reps: 9 },
        ],
      },
    ],
  },
  {
    id: "w3",
    name: "Push Heavy",
    started: "2024-01-08T10:00:00",
    exercises: [
      {
        name: "Bench Press",
        intensity: "heavy",
        sets: [{ type: "regular", weight: 102.5, reps: 3 }],
      },
      {
        name: "Wrist Curl",
        sets: [
          { type: "regular", weight: 20, reps: 12, arm: "left" },
          { type: "regular", weight: 25, reps: 8, arm: "right" },
        ],
      },
    ],
  },
]);

const history = buildExerciseHistory(workouts);
const bench = history.get("Bench Press")!;

describe("buildExerciseHistory", () => {
  it("indexes entries per exercise, newest first", () => {
    expect(bench).toHaveLength(3);
    expect(bench.map((e) => e.workoutId)).toEqual(["w3", "w2", "w1"]);
    expect(bench[0]!.intensity).toBe("heavy");
    expect(history.get("Wrist Curl")![0]!.intensity).toBeNull();
  });

  it("handles workouts without exercises", () => {
    expect(buildExerciseHistory(partial<(Workout | null)[]>([{ id: "w9" }, null])).size).toBe(0);
  });
});

describe("getMaxWeight", () => {
  it("ignores warmup sets and honours exclusions", () => {
    expect(getMaxWeight(bench)).toBe(102.5);
    expect(getMaxWeight(bench, { excludeWorkoutId: "w3" })).toBe(100);
  });

  it("filters by intensity", () => {
    expect(getMaxWeight(bench, { intensity: "light" })).toBe(70);
  });

  it("filters by arm", () => {
    const wrist = history.get("Wrist Curl");
    expect(getMaxWeight(wrist, { arm: "left" })).toBe(20);
    expect(getMaxWeight(wrist, { arm: "right" })).toBe(25);
    expect(getMaxWeight(wrist)).toBe(25);
  });
});

describe("getBestRepsAtWeight", () => {
  it("returns best reps at an exact weight", () => {
    expect(getBestRepsAtWeight(bench, 100)).toBe(5);
    expect(getBestRepsAtWeight(bench, 70)).toBe(10);
    expect(getBestRepsAtWeight(bench, 80)).toBeNull();
    expect(getBestRepsAtWeight(bench, 0)).toBeNull();
  });

  it("scopes to intensity", () => {
    expect(getBestRepsAtWeight(bench, 100, { intensity: "light" })).toBeNull();
  });
});

describe("getBestEstimated1RM", () => {
  it("prefers actual max weight", () => {
    expect(getBestEstimated1RM(bench)).toBe(102.5);
  });

  it("falls back to Epley when no weight is heavier", () => {
    const entries = partial<HistoryEntry[]>([{ workoutId: "w1", sets: [{ weight: 60, reps: 10 }] }]);
    expect(getBestEstimated1RM(entries)).toBe(60);
  });
});

describe("getLastEntry and formatEntrySets", () => {
  it("finds the last light session", () => {
    const entry = getLastEntry(bench, { intensity: "light" });
    expect(entry!.workoutId).toBe("w2");
    expect(formatEntrySets(entry)).toBe("70×10, 70×9");
  });

  it("skips the workout being edited", () => {
    const entry = getLastEntry(bench, { intensity: "heavy", excludeWorkoutId: "w3" });
    expect(entry!.workoutId).toBe("w1");
    expect(formatEntrySets(entry)).toBe("100×5, 100×4");
  });

  it("returns null when nothing matches", () => {
    // Deliberately invalid intensity
    expect(getLastEntry(bench, { intensity: "nope" as Intensity })).toBeNull();
    expect(formatEntrySets(null)).toBe("");
  });

  it("labels single arm sets", () => {
    const entry = getLastEntry(history.get("Wrist Curl"));
    expect(formatEntrySets(entry)).toBe("20×12L, 25×8R");
    expect(formatEntrySets(entry, { arm: "left" })).toBe("20×12L");
  });
});
