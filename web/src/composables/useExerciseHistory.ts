import { ref, computed, type Ref } from "vue";
import { getWorkouts, getSetting } from "@/api/data";
import {
  buildExerciseHistory,
  getMaxWeight,
  getBestRepsAtWeight,
  getLastEntry,
  type HistoryEntry,
  type HistoryScope,
} from "@/utils/exerciseHistory";
import type { Arm, Workout, WorkoutExercise, WorkoutSet } from "@/types/domain";

/**
 * Compare scope setting values
 * "intensity": compare against workouts logged with the same intensity tag
 * "all": compare against every workout
 */
export const COMPARE_SCOPES = ["intensity", "all"] as const;
export type CompareScope = (typeof COMPARE_SCOPES)[number];

type ArmArg = Arm | null | undefined;

/**
 * Exercise history lookups for the workout editor.
 *
 * Records and % of max always use the full history, since a PR is a PR.
 * "Previous best" and "last time" honour the compare scope setting so a light
 * day is compared against previous light days.
 */
export function useExerciseHistory(workout: Ref<Workout>) {
  const workouts = ref<Workout[]>([]);
  const compareScope = ref<CompareScope>("intensity");

  const history = computed(() => buildExerciseHistory(workouts.value));

  async function loadHistory() {
    try {
      const [data, scope] = await Promise.all([
        getWorkouts(),
        getSetting<CompareScope>("compareScope", "intensity"),
      ]);
      workouts.value = data;
      compareScope.value = scope;
    } catch (error) {
      console.error("Error loading exercise history:", error);
    }
  }

  function entriesFor(exerciseName: string): HistoryEntry[] {
    return history.value.get(exerciseName) || [];
  }

  /**
   * Scope used for comparisons. Only narrows by intensity when the
   * exercise has one and the setting asks for it.
   */
  function comparisonScope(exercise: WorkoutExercise, arm?: ArmArg): HistoryScope {
    return {
      arm,
      excludeWorkoutId: workout.value.id,
      intensity:
        compareScope.value === "intensity" ? exercise.intensity || null : null,
    };
  }

  /** Best reps at this weight in the current (unsaved) workout */
  function currentWorkoutBestReps(exerciseName: string, weight: number | null, arm: ArmArg) {
    return getBestRepsAtWeight(
      [currentEntry(exerciseName)],
      weight,
      { arm },
    );
  }

  function currentSets(exerciseName: string): WorkoutSet[] {
    return workout.value.exercises
      .filter((ex) => ex.name === exerciseName)
      .flatMap((ex) => ex.sets || []);
  }

  /** Historical max, excluding the workout being edited */
  /** The unsaved sets of the current workout, shaped as a history entry */
  function currentEntry(exerciseName: string): HistoryEntry {
    return {
      workoutId: undefined,
      workoutName: "",
      date: new Date(),
      intensity: null,
      sets: currentSets(exerciseName),
    };
  }

  function historicalMax(exerciseName: string, arm: ArmArg) {
    return getMaxWeight(entriesFor(exerciseName), {
      arm,
      excludeWorkoutId: workout.value.id,
    });
  }

  /**
   * Weight as a percentage of the all-time max (including this workout)
   */
  function getMaxPercentage(exerciseName: string, weight: number | null, arm: ArmArg): string {
    if (!weight || weight <= 0) return "-";
    const max = Math.max(
      historicalMax(exerciseName, arm),
      getMaxWeight([currentEntry(exerciseName)], { arm }),
    );
    return max > 0 ? `${Math.round((weight / max) * 100)}%` : "-";
  }

  /**
   * Previous best reps at this weight, within the compare scope
   */
  function getPreviousReps(exercise: WorkoutExercise, weight: number | null, arm: ArmArg) {
    return getBestRepsAtWeight(
      entriesFor(exercise.name),
      weight,
      comparisonScope(exercise, arm),
    );
  }

  /**
   * Highest reps ever at this weight, including the current workout
   */
  function getHighestReps(exerciseName: string, weight: number | null, arm: ArmArg) {
    const historical = getBestRepsAtWeight(entriesFor(exerciseName), weight, {
      arm,
      excludeWorkoutId: workout.value.id,
    });
    const current = currentWorkoutBestReps(exerciseName, weight, arm);
    const best = Math.max(historical || 0, current || 0);
    return best > 0 ? best : null;
  }

  /** True when this set beats every previous set at this weight (all history) */
  function isRepRecord(
    exerciseName: string,
    weight: number | null,
    reps: number | null,
    arm: ArmArg,
  ): boolean {
    if (!weight || !reps || weight <= 0 || reps <= 0) return false;
    const previous = getBestRepsAtWeight(entriesFor(exerciseName), weight, {
      arm,
      excludeWorkoutId: workout.value.id,
    });
    if (previous !== null && reps <= previous) return false;
    // Only star the best set for this weight within the current workout
    return reps === currentWorkoutBestReps(exerciseName, weight, arm);
  }

  /** True when this weight is heavier than anything in history */
  function isWeightRecord(exerciseName: string, weight: number | null, arm: ArmArg): boolean {
    if (!weight || weight <= 0) return false;
    return weight > historicalMax(exerciseName, arm);
  }

  /**
   * The most recent previous performance of this exercise within the compare scope
   */
  function getLastTime(exercise: WorkoutExercise): HistoryEntry | null {
    return getLastEntry(entriesFor(exercise.name), comparisonScope(exercise));
  }

  return {
    workouts,
    history,
    compareScope,
    loadHistory,
    getMaxPercentage,
    getPreviousReps,
    getHighestReps,
    isRepRecord,
    isWeightRecord,
    getLastTime,
  };
}
