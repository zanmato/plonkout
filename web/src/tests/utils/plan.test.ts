import { describe, expect, it } from "vitest";
import {
  complianceOf,
  doneSessions,
  formatRange,
  formatSetTarget,
  formatTarget,
  offArmWeight,
  targetWeightFor,
} from "@/utils/plan";
import type { Comparison, PlannedSession, Target } from "@/types/domain";
import { partial } from "../helpers/fixtures";

const target = (fields: Partial<Target> = {}): Target => ({
  id: "t1",
  setType: "Top set",
  sets: 1,
  reps: 5,
  weight: 145,
  time: "",
  rpeMin: 8,
  rpeMax: 9,
  notes: "",
  ...fields,
});

describe("offArmWeight", () => {
  it("takes the percentage and rounds to a half", () => {
    expect(offArmWeight(145, 82.5)).toBe(119.5);
    expect(offArmWeight(100, 80)).toBe(80);
    expect(offArmWeight(21, 83)).toBe(17.5);
    expect(offArmWeight(20, 86.2)).toBe(17);
  });
});

describe("targetWeightFor", () => {
  const planned = { singleArm: true, offArmPercent: 80 };

  it("gives the dominant arm the listed weight", () => {
    expect(targetWeightFor(target(), planned, "right", "right")).toBe(145);
  });

  it("gives the off arm its share", () => {
    expect(targetWeightFor(target(), planned, "left", "right")).toBe(116);
    expect(targetWeightFor(target(), planned, "right", "left")).toBe(116);
  });

  it("keeps the listed weight without an off arm percentage or arm", () => {
    expect(targetWeightFor(target(), { singleArm: true, offArmPercent: null }, "left", "right")).toBe(145);
    expect(targetWeightFor(target(), planned, "", "right")).toBe(145);
    expect(targetWeightFor(target(), { singleArm: false, offArmPercent: 80 }, "left", "right")).toBe(145);
    expect(targetWeightFor(target(), undefined, "left", "right")).toBe(145);
  });

  it("has no weight when the target has none", () => {
    expect(targetWeightFor(target({ weight: null }), planned, "left", "right")).toBeNull();
  });
});

describe("formatRange", () => {
  it("joins the ends of a range", () => {
    expect(formatRange(8, 9)).toBe("8 to 9");
    expect(formatRange(7.5, 8, "till")).toBe("7.5 till 8");
  });

  it("shows a single value once", () => {
    expect(formatRange(8, 8)).toBe("8");
    expect(formatRange(null, 9)).toBe("9");
    expect(formatRange(null, null)).toBe("");
  });
});

describe("formatTarget", () => {
  it("describes the prescription", () => {
    expect(formatTarget(target(), { unit: "kg" })).toBe("1 × 5 @ 145 kg, RPE 8 to 9");
    expect(formatTarget(target({ sets: 3, reps: 5, weight: 130, rpeMin: 7, rpeMax: null }), { unit: "lbs" })).toBe(
      "3 × 5 @ 130 lbs, RPE 7",
    );
  });

  it("uses the time of a hold and leaves out what is not prescribed", () => {
    expect(
      formatTarget(target({ sets: 2, reps: null, time: "0:30", weight: null, rpeMin: null, rpeMax: null }), {
        unit: "kg",
      }),
    ).toBe("2 × 0:30");
  });

  it("takes an overriding weight", () => {
    expect(formatTarget(target(), { unit: "kg", weight: 119.5, to: "till" })).toBe("1 × 5 @ 119.5 kg, RPE 8 till 9");
  });
});

describe("formatSetTarget", () => {
  it("describes one set", () => {
    expect(formatSetTarget(target(), { unit: "kg" })).toBe("Top set · 145 kg × 5 · RPE 8 to 9");
  });

  it("leaves out missing parts", () => {
    expect(formatSetTarget(target({ weight: null, rpeMin: null, rpeMax: null }), { unit: "kg" })).toBe("Top set · 5");
    expect(formatSetTarget(target({ setType: "", reps: null }), { unit: "kg", weight: 20 })).toBe(
      "20 kg · RPE 8 to 9",
    );
  });
});

describe("complianceOf", () => {
  it("counts the targets met", () => {
    const comparison = partial<Comparison>({
      exercises: [
        { targets: [{ met: true }, { met: false }] },
        { targets: [{ met: true }] },
      ],
      unplanned: [],
    });
    expect(complianceOf(comparison)).toEqual({ met: 2, total: 3 });
  });
});

describe("doneSessions", () => {
  it("keeps completed and skipped sessions, latest in the plan first", () => {
    const sessions = partial<PlannedSession[]>([
      { id: "a", position: 1, status: "completed" },
      { id: "b", position: 2, status: "skipped" },
      { id: "c", position: 3, status: "in_progress" },
      { id: "d", position: 4, status: "pending" },
    ]);
    expect(doneSessions(sessions).map((s) => s.id)).toEqual(["b", "a"]);
  });
});
