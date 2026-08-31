import { ref, computed } from "vue";
import { getWorkouts, getSetting } from "@/utils/database.js";
import {
  buildExerciseHistory,
  getMaxWeight,
  getBestRepsAtWeight,
  getLastEntry,
} from "@/utils/exerciseHistory.js";

/**
 * Compare scope setting values
 * "intensity": compare against workouts logged with the same intensity tag
 * "all": compare against every workout
 */
export const COMPARE_SCOPES = ["intensity", "all"];

/**
 * Exercise history lookups for the workout editor.
 *
 * Records and % of max always use the full history, since a PR is a PR.
 * "Previous best" and "last time" honour the compare scope setting so a light
 * day is compared against previous light days.
 *
 * @param {import("vue").Ref<Object>} workout - The workout being edited
 */
export function useExerciseHistory(workout) {
  const workouts = ref([]);
  const compareScope = ref("intensity");

  const history = computed(() => buildExerciseHistory(workouts.value));

  async function loadHistory() {
    try {
      const [data, scope] = await Promise.all([
        getWorkouts(),
        getSetting("compareScope", "intensity"),
      ]);
      workouts.value = data;
      compareScope.value = scope;
    } catch (error) {
      console.error("Error loading exercise history:", error);
    }
  }

  function entriesFor(exerciseName) {
    return history.value.get(exerciseName) || [];
  }

  /**
   * Scope used for comparisons. Only narrows by intensity when the
   * exercise has one and the setting asks for it.
   */
  function comparisonScope(exercise, arm) {
    return {
      arm,
      excludeWorkoutId: workout.value.id,
      intensity:
        compareScope.value === "intensity" ? exercise.intensity || null : null,
    };
  }

  /** Best reps at this weight in the current (unsaved) workout */
  function currentWorkoutBestReps(exerciseName, weight, arm) {
    return getBestRepsAtWeight(
      [{ workoutId: null, sets: currentSets(exerciseName) }],
      weight,
      { arm },
    );
  }

  function currentSets(exerciseName) {
    return workout.value.exercises
      .filter((ex) => ex.name === exerciseName)
      .flatMap((ex) => ex.sets || []);
  }

  /** Historical max, excluding the workout being edited */
  function historicalMax(exerciseName, arm) {
    return getMaxWeight(entriesFor(exerciseName), {
      arm,
      excludeWorkoutId: workout.value.id,
    });
  }

  /**
   * Weight as a percentage of the all-time max (including this workout)
   * @returns {string}
   */
  function getMaxPercentage(exerciseName, weight, arm) {
    if (!weight || weight <= 0) return "-";
    const max = Math.max(
      historicalMax(exerciseName, arm),
      getMaxWeight([{ workoutId: null, sets: currentSets(exerciseName) }], {
        arm,
      }),
    );
    return max > 0 ? `${Math.round((weight / max) * 100)}%` : "-";
  }

  /**
   * Previous best reps at this weight, within the compare scope
   * @returns {number|null}
   */
  function getPreviousReps(exercise, weight, arm) {
    return getBestRepsAtWeight(
      entriesFor(exercise.name),
      weight,
      comparisonScope(exercise, arm),
    );
  }

  /**
   * Highest reps ever at this weight, including the current workout
   * @returns {number|null}
   */
  function getHighestReps(exerciseName, weight, arm) {
    const historical = getBestRepsAtWeight(entriesFor(exerciseName), weight, {
      arm,
      excludeWorkoutId: workout.value.id,
    });
    const current = currentWorkoutBestReps(exerciseName, weight, arm);
    const best = Math.max(historical || 0, current || 0);
    return best > 0 ? best : null;
  }

  /** True when this set beats every previous set at this weight (all history) */
  function isRepRecord(exerciseName, weight, reps, arm) {
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
  function isWeightRecord(exerciseName, weight, arm) {
    if (!weight || weight <= 0) return false;
    return weight > historicalMax(exerciseName, arm);
  }

  /**
   * The most recent previous performance of this exercise within the compare scope
   * @returns {Object|null} History entry
   */
  function getLastTime(exercise) {
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
