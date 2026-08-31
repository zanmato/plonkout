/**
 * Pure helpers for looking up an exercise's history across workouts.
 *
 * All lookups run against a pre-built index (see buildExerciseHistory) so
 * that the UI never has to rescan every workout per set.
 */

export const INTENSITIES = ["heavy", "light"];

/**
 * Check if two arm settings are compatible for comparison.
 * "both" arm history counts for a single arm, but not the other way around.
 * @param {string} currentArm
 * @param {string} historicalArm
 * @returns {boolean}
 */
export function isArmCompatible(currentArm, historicalArm) {
  if (!currentArm || !historicalArm) return true;
  if (currentArm === historicalArm) return true;
  return (
    historicalArm === "both" &&
    (currentArm === "left" || currentArm === "right")
  );
}

/**
 * Whether a set counts as a working (non warmup) set
 * @param {Object} set
 * @returns {boolean}
 */
export function isWorkingSet(set) {
  return !set.type || set.type === "regular";
}

/**
 * Build an index of exercise name -> history entries, newest first.
 * @param {Array} workouts - All workouts
 * @returns {Map<string, Array<{workoutId:number, workoutName:string, date:Date, intensity:string|null, sets:Array}>>}
 */
export function buildExerciseHistory(workouts) {
  const history = new Map();

  for (const workout of workouts || []) {
    if (!workout) continue;
    const date = new Date(workout.started);
    for (const exercise of workout.exercises || []) {
      if (!exercise?.name) continue;
      if (!history.has(exercise.name)) history.set(exercise.name, []);
      history.get(exercise.name).push({
        workoutId: workout.id,
        workoutName: workout.name || "",
        date,
        intensity: exercise.intensity || null,
        sets: exercise.sets || [],
      });
    }
  }

  for (const entries of history.values()) {
    entries.sort((a, b) => b.date - a.date);
  }

  return history;
}

/**
 * Filter history entries by scope.
 * @param {Array} entries
 * @param {Object} scope
 * @param {number} [scope.excludeWorkoutId] - Skip this workout (usually the one being edited)
 * @param {string|null} [scope.intensity] - Only include entries with this intensity. Falsy means any.
 * @returns {Array}
 */
export function filterEntries(entries, scope = {}) {
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
 * @param {Array} entries
 * @param {Object} scope - See filterEntries, plus optional arm
 * @param {(set: Object, entry: Object) => void} fn
 */
function eachWorkingSet(entries, scope, fn) {
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
 * @param {Array} entries
 * @param {Object} scope
 * @returns {number}
 */
export function getMaxWeight(entries, scope = {}) {
  let max = 0;
  eachWorkingSet(entries, scope, (set) => {
    const weight = Number(set.weight) || 0;
    if (weight > max) max = weight;
  });
  return max;
}

/**
 * Best reps performed at an exact weight
 * @param {Array} entries
 * @param {number} weight
 * @param {Object} scope
 * @returns {number|null}
 */
export function getBestRepsAtWeight(entries, weight, scope = {}) {
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
 * @param {number} weight
 * @param {number} reps
 * @returns {number}
 */
export function calculateEpley1RM(weight, reps) {
  if (!weight || !reps || weight <= 0 || reps <= 0) return 0;
  return weight * (1 + Math.min(reps, 12) / 30);
}

/**
 * Best estimated 1RM. Actual max weight is preferred, Epley is the fallback.
 * @param {Array} entries
 * @param {Object} scope
 * @returns {number}
 */
export function getBestEstimated1RM(entries, scope = {}) {
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
 * @param {Array} entries
 * @param {Object} scope
 * @returns {Object|null}
 */
export function getLastEntry(entries, scope = {}) {
  return filterEntries(entries, scope)[0] || null;
}

/**
 * Format the working sets of an entry as a compact summary, e.g. "60×10, 60×9"
 * @param {Object} entry
 * @param {Object} options
 * @param {"strength"|"cardio"} [options.type]
 * @param {string} [options.arm] - Only include sets compatible with this arm
 * @returns {string}
 */
export function formatEntrySets(entry, { type = "strength", arm = "" } = {}) {
  if (!entry) return "";
  return entry.sets
    .filter((set) => isWorkingSet(set) && isArmCompatible(arm, set.arm))
    .map((set) => {
      const armSuffix =
        set.arm === "left" ? "L" : set.arm === "right" ? "R" : "";
      if (type === "cardio") {
        const parts = [];
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
