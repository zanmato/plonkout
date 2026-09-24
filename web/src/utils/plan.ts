/**
 * Pure helpers for training plans: the weight a set is planned at, target
 * descriptions and how well a session went.
 */
import type {
  Arm,
  Comparison,
  PlannedExercise,
  PlannedSession,
  Target,
} from "@/types/domain";

export type DominantArm = "left" | "right";

/**
 * The weight the non dominant arm works with, rounded to a half, which is
 * what plates and stacks allow. Must match OffArmWeight on the server.
 */
export function offArmWeight(weight: number, percent: number): number {
  return Math.round(((weight * percent) / 100) * 2) / 2;
}

/**
 * The weight a set of a target is planned at. Targets list the dominant arm's
 * weight, a single arm exercise's off arm set gets its share of it.
 */
export function targetWeightFor(
  target: Target,
  planned: Pick<PlannedExercise, "singleArm" | "offArmPercent"> | undefined,
  arm: Arm,
  dominantArm: DominantArm,
): number | null {
  if (target.weight === null) return null;
  const offArm = planned?.singleArm && arm && arm !== "both" && arm !== dominantArm;
  if (offArm && planned.offArmPercent !== null) {
    return offArmWeight(target.weight, planned.offArmPercent);
  }
  return target.weight;
}

const number = (value: number) => String(Number(value.toFixed(2)));

/** "8 to 9", "8" or "" for an RPE range. */
export function formatRange(min: number | null, max: number | null, to = "to"): string {
  if (min !== null && max !== null && min !== max) return `${number(min)} ${to} ${number(max)}`;
  const only = min ?? max;
  return only === null ? "" : number(only);
}

/** Reps or the time of a hold, whichever the target prescribes. */
function amount(target: Target): string {
  if (target.reps !== null) return String(target.reps);
  return target.time;
}

export interface FormatOptions {
  unit: string;
  /** The word between the ends of a range, "to" in English. */
  to?: string;
  /** Overrides the target's weight, e.g. for an off arm set. */
  weight?: number | null;
}

/** A target's prescription, e.g. "3 × 5 @ 130 kg, RPE 7 to 8". */
export function formatTarget(target: Target, { unit, to = "to", weight = target.weight }: FormatOptions): string {
  const reps = amount(target);
  let text = reps ? `${target.sets} × ${reps}` : `${target.sets} ×`;
  if (weight !== null) text += ` @ ${number(weight)} ${unit}`;
  const rpe = formatRange(target.rpeMin, target.rpeMax, to);
  if (rpe) text += `, RPE ${rpe}`;
  return text;
}

/** What one set of a target asks for, e.g. "Top set · 145 kg × 5 · RPE 8 to 9". */
export function formatSetTarget(target: Target, { unit, to = "to", weight = target.weight }: FormatOptions): string {
  const reps = amount(target);
  const load = [weight !== null ? `${number(weight)} ${unit}` : "", reps].filter(Boolean).join(" × ");
  const rpe = formatRange(target.rpeMin, target.rpeMax, to);
  return [target.setType, load, rpe ? `RPE ${rpe}` : ""].filter(Boolean).join(" · ");
}

export interface Compliance {
  met: number;
  total: number;
}

/** How many of a session's targets were met. */
export function complianceOf(comparison: Comparison): Compliance {
  const targets = comparison.exercises.flatMap((exercise) => exercise.targets);
  return { met: targets.filter((target) => target.met).length, total: targets.length };
}

/**
 * Sessions that are done with, newest first. Skipped sessions have no date,
 * so the plan's order stands in for when they happened.
 */
export function doneSessions(sessions: PlannedSession[]): PlannedSession[] {
  return sessions
    .filter((session) => session.status === "completed" || session.status === "skipped")
    .sort((a, b) => b.position - a.position);
}
