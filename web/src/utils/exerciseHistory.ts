/**
 * Pure helpers for looking up an exercise's history across workouts.
 *
 * All lookups run against a pre-built index (see buildExerciseHistory) so
 * that the UI never has to rescan every workout per set.
 */

import type { Arm, ExerciseType, Id, Intensity, Workout, WorkoutSet } from "@/types/domain";

export const INTENSITIES: Intensity[] = ["heavy", "light"];

export interface HistoryEntry {
  workoutId: Id | undefined;
  workoutName: string;
  date: Date;
  intensity: Intensity | null;
  sets: WorkoutSet[];
}

export type ExerciseHistory = Map<string, HistoryEntry[]>;

export interface HistoryScope {
  /** Skip this workout, usually the one being edited */
  excludeWorkoutId?: Id | null;
  /** Only include entries with this intensity. Falsy means any. */
  intensity?: Intensity | null;
  /** Only include sets compatible with this arm */
  arm?: Arm | null;
}

/**
 * Check if two arm settings are compatible for comparison.
 * "both" arm history counts for a single arm, but not the other way around.
 */
export function isArmCompatible(
  currentArm: Arm | null | undefined,
  historicalArm: Arm | null | undefined,
): boolean {
  if (!currentArm || !historicalArm) return true;
  if (currentArm === historicalArm) return true;
  return (
    historicalArm === "both" &&
    (currentArm === "left" || currentArm === "right")
  );
}

/**
 * Whether a set counts as a working (non warmup) set
 */
export function isWorkingSet(set: Pick<WorkoutSet, "type">): boolean {
  return !set.type || set.type === "regular";
}

/**
 * Build an index of exercise name -> history entries, newest first.
 */
export function buildExerciseHistory(
  workouts: ReadonlyArray<Workout | null | undefined> | null | undefined,
): ExerciseHistory {
  const history: ExerciseHistory = new Map();

  for (const workout of workouts || []) {
    if (!workout) continue;
    const date = new Date(workout.started);
    for (const exercise of workout.exercises || []) {
      if (!exercise?.name) continue;
      let entries = history.get(exercise.name);
      if (!entries) {
        entries = [];
        history.set(exercise.name, entries);
      }
      entries.push({
        workoutId: workout.id,
        workoutName: workout.name || "",
        date,
        intensity: exercise.intensity || null,
        sets: exercise.sets || [],
      });
    }
  }

  for (const entries of history.values()) {
    entries.sort((a, b) => b.date.getTime() - a.date.getTime());
  }

  return history;
}

/**
 * Filter history entries by scope.
 */
export function filterEntries(
  entries: HistoryEntry[] | null | undefined,
  scope: HistoryScope = {},
): HistoryEntry[] {
  return (entries || []).filter((entry) => {
    if (scope.excludeWorkoutId && entry.workoutId === scope.excludeWorkoutId) {
      return false;
    }
    if (scope.intensity && entry.intensity !== scope.intensity) {
      return false;
    }
    return true;
  });
}

/**
 * Iterate the working sets of the scoped entries
 */
function eachWorkingSet(
  entries: HistoryEntry[] | null | undefined,
  scope: HistoryScope,
  fn: (set: WorkoutSet, entry: HistoryEntry) => void,
): void {
  for (const entry of filterEntries(entries, scope)) {
    for (const set of entry.sets) {
      if (!isWorkingSet(set)) continue;
      if (!isArmCompatible(scope.arm, set.arm)) continue;
      fn(set, entry);
    }
  }
}

/**
 * Heaviest weight lifted for the exercise
 */
export function getMaxWeight(
  entries: HistoryEntry[] | null | undefined,
  scope: HistoryScope = {},
): number {
  let max = 0;
  eachWorkingSet(entries, scope, (set) => {
    const weight = Number(set.weight) || 0;
    if (weight > max) max = weight;
  });
  return max;
}

/**
 * Best reps performed at an exact weight
 */
export function getBestRepsAtWeight(
  entries: HistoryEntry[] | null | undefined,
  weight: number | null | undefined,
  scope: HistoryScope = {},
): number | null {
  if (!weight || weight <= 0) return null;
  let best = 0;
  eachWorkingSet(entries, scope, (set) => {
    const reps = Number(set.reps) || 0;
    if (Number(set.weight) === Number(weight) && reps > best) best = reps;
  });
  return best > 0 ? best : null;
}

/**
 * Epley one rep max estimate, capped at 12 reps where the formula is reliable
 */
export function calculateEpley1RM(
  weight: number | null | undefined,
  reps: number | null | undefined,
): number {
  if (!weight || !reps || weight <= 0 || reps <= 0) return 0;
  return weight * (1 + Math.min(reps, 12) / 30);
}

/**
 * Best estimated 1RM. Actual max weight is preferred, Epley is the fallback.
 */
export function getBestEstimated1RM(
  entries: HistoryEntry[] | null | undefined,
  scope: HistoryScope = {},
): number {
  let maxWeight = 0;
  let bestEpley = 0;
  eachWorkingSet(entries, scope, (set) => {
    const weight = Number(set.weight) || 0;
    const reps = Number(set.reps) || 0;
    if (!weight || !reps) return;
    if (weight > maxWeight) maxWeight = weight;
    const epley = calculateEpley1RM(weight, reps);
    if (epley > bestEpley) bestEpley = epley;
  });
  return maxWeight > 0 ? maxWeight : bestEpley;
}

/**
 * The most recent history entry within the scope
 */
export function getLastEntry(
  entries: HistoryEntry[] | null | undefined,
  scope: HistoryScope = {},
): HistoryEntry | null {
  return filterEntries(entries, scope)[0] || null;
}

/**
 * Format the working sets of an entry as a compact summary, e.g. "60×10, 60×9"
 */
export function formatEntrySets(
  entry: HistoryEntry | null | undefined,
  { type = "strength", arm = "" }: { type?: ExerciseType; arm?: Arm } = {},
): string {
  if (!entry) return "";
  return entry.sets
    .filter((set) => isWorkingSet(set) && isArmCompatible(arm, set.arm))
    .map((set) => {
      const armSuffix =
        set.arm === "left" ? "L" : set.arm === "right" ? "R" : "";
      if (type === "cardio") {
        const parts: string[] = [];
        if (set.distance) parts.push(String(set.distance));
        if (set.time) parts.push(set.time);
        return parts.join("/") + armSuffix;
      }
      const weight = set.weight ?? 0;
      const amount = set.reps ?? set.time ?? 0;
      return `${weight}×${amount}${armSuffix}`;
    })
    .filter(Boolean)
    .join(", ");
}
