/**
 * Block Periodization Utility
 * Provides functions for calculating progressive overload recommendations
 * based on block periodization principles.
 */

import { getSetting, saveSetting } from "./database.js";

// Rep ranges for each week of the block
const REP_RANGES = {
  1: { min: 8, max: 10 }, // 75%
  2: { min: 6, max: 8 }, // 80%
  3: { min: 5, max: 6 }, // 85%
  4: { min: 4, max: 5 }, // 90%
  5: { min: 3, max: 4 }, // 95%
  6: { min: 1, max: 3 }, // 100%+ (max test)
};

// Default global settings
const DEFAULT_SETTINGS = {
  weeksPerBlock: 5,
  workoutsPerWeek: 1,
  startPercentage: 75,
  progressionPerWeek: 5,
};

/**
 * Calculate estimated 1RM using Epley formula
 * @param {number} weight - Weight lifted
 * @param {number} reps - Reps performed
 * @returns {number} Estimated 1RM
 */
export function calculateEpley1RM(weight, reps) {
  if (!weight || !reps || weight <= 0 || reps <= 0) return 0;
  // Formula is less accurate for high reps, cap at 12
  const cappedReps = Math.min(reps, 12);
  return weight * (1 + cappedReps / 30);
}

/**
 * Check if two arm settings are compatible for comparison
 * @param {string} currentArm - Current arm setting
 * @param {string} historicalArm - Historical arm setting to compare
 * @returns {boolean} Whether the arms are compatible for comparison
 */
export function isArmCompatible(currentArm, historicalArm) {
  // If either arm is not specified, they're compatible
  if (!currentArm || !historicalArm) {
    return true;
  }

  // Exact match
  if (currentArm === historicalArm) {
    return true;
  }

  // 'both' is compatible with individual arms
  if (
    historicalArm === "both" &&
    (currentArm === "left" || currentArm === "right")
  ) {
    return true;
  }

  return false;
}

/**
 * Get the maximum weight lifted for an exercise
 * @param {Array} workouts - All workouts
 * @param {string} exerciseName - Exercise to analyze
 * @param {string} arm - Arm filter for single-arm exercises
 * @param {number} excludeWorkoutId - Workout ID to exclude (current workout)
 * @returns {number} Maximum weight lifted
 */
export function getMaxWeight(workouts, exerciseName, arm, excludeWorkoutId) {
  let maxWeight = 0;

  workouts.forEach((workout) => {
    if (excludeWorkoutId && workout.id === excludeWorkoutId) return;

    workout.exercises?.forEach((exercise) => {
      if (exercise.name === exerciseName) {
        exercise.sets?.forEach((set) => {
          if (set.weight && set.weight > maxWeight && set.type === "regular") {
            if (isArmCompatible(arm, set.arm)) {
              maxWeight = set.weight;
            }
          }
        });
      }
    });
  });

  return maxWeight;
}

/**
 * Get the best estimated 1RM from all historical sets
 * Uses max weight as primary, Epley estimate as fallback
 * @param {Array} workouts - All workouts
 * @param {string} exerciseName - Exercise to analyze
 * @param {string} arm - Arm filter for single-arm exercises
 * @param {number} excludeWorkoutId - Workout ID to exclude
 * @returns {number} Best estimated 1RM
 */
export function getBestEstimated1RM(
  workouts,
  exerciseName,
  arm,
  excludeWorkoutId,
) {
  let maxWeight = 0;
  let bestEpley1RM = 0;

  workouts.forEach((workout) => {
    if (excludeWorkoutId && workout.id === excludeWorkoutId) return;

    workout.exercises?.forEach((exercise) => {
      if (exercise.name === exerciseName) {
        exercise.sets?.forEach((set) => {
          if (set.weight && set.type === "regular" && set.reps) {
            if (isArmCompatible(arm, set.arm)) {
              // Track max weight
              if (set.weight > maxWeight) {
                maxWeight = set.weight;
              }
              // Track best Epley estimate
              const estimated1RM = calculateEpley1RM(set.weight, set.reps);
              if (estimated1RM > bestEpley1RM) {
                bestEpley1RM = estimated1RM;
              }
            }
          }
        });
      }
    });
  });

  // Prefer actual max weight, fallback to Epley estimate
  return maxWeight > 0 ? maxWeight : bestEpley1RM;
}

/**
 * Get global block periodization settings
 * @returns {Promise<Object>} Global settings
 */
export async function getGlobalSettings() {
  return {
    weeksPerBlock: await getSetting(
      "blockPeriodization_weeksPerBlock",
      DEFAULT_SETTINGS.weeksPerBlock,
    ),
    workoutsPerWeek: await getSetting(
      "blockPeriodization_workoutsPerWeek",
      DEFAULT_SETTINGS.workoutsPerWeek,
    ),
    startPercentage: await getSetting(
      "blockPeriodization_startPercentage",
      DEFAULT_SETTINGS.startPercentage,
    ),
    progressionPerWeek: await getSetting(
      "blockPeriodization_progressionPerWeek",
      DEFAULT_SETTINGS.progressionPerWeek,
    ),
  };
}

/**
 * Save global block periodization settings
 * @param {Object} settings - Settings to save
 */
export async function saveGlobalSettings(settings) {
  if (settings.weeksPerBlock !== undefined) {
    await saveSetting(
      "blockPeriodization_weeksPerBlock",
      settings.weeksPerBlock,
    );
  }
  if (settings.workoutsPerWeek !== undefined) {
    await saveSetting(
      "blockPeriodization_workoutsPerWeek",
      settings.workoutsPerWeek,
    );
  }
  if (settings.startPercentage !== undefined) {
    await saveSetting(
      "blockPeriodization_startPercentage",
      settings.startPercentage,
    );
  }
  if (settings.progressionPerWeek !== undefined) {
    await saveSetting(
      "blockPeriodization_progressionPerWeek",
      settings.progressionPerWeek,
    );
  }
}

/**
 * Get block state for a specific exercise
 * @param {string} exerciseName - Exercise name
 * @returns {Promise<Object|null>} Block state or null if no active block
 */
export async function getExerciseBlockState(exerciseName) {
  const allBlocks = await getSetting("blockPeriodization_exercises", {});
  return allBlocks[exerciseName] || null;
}

/**
 * Get effective settings for an exercise (merges global + per-exercise overrides)
 * @param {string} exerciseName - Exercise name
 * @returns {Promise<Object>} Effective settings
 */
export async function getEffectiveSettings(exerciseName) {
  const global = await getGlobalSettings();
  const exerciseState = await getExerciseBlockState(exerciseName);

  return {
    weeksPerBlock: exerciseState?.weeksPerBlock ?? global.weeksPerBlock,
    workoutsPerWeek: exerciseState?.workoutsPerWeek ?? global.workoutsPerWeek,
    startPercentage: exerciseState?.startPercentage ?? global.startPercentage,
    progressionPerWeek:
      exerciseState?.progressionPerWeek ?? global.progressionPerWeek,
  };
}

/**
 * Calculate current week of block
 * @param {Object} blockState - Block state
 * @param {number} workoutsPerWeek - Workouts per week setting
 * @returns {number} Current week (1-indexed)
 */
export function getCurrentBlockWeek(blockState, workoutsPerWeek) {
  if (!blockState) return 0;
  if (blockState.blockWorkoutCount <= 0) return 1;
  return Math.ceil(blockState.blockWorkoutCount / workoutsPerWeek);
}

/**
 * Calculate current percentage based on week
 * @param {number} week - Current week
 * @param {number} startPercentage - Starting percentage
 * @param {number} progressionPerWeek - Weekly progression
 * @returns {number} Current percentage
 */
export function getCurrentPercentage(
  week,
  startPercentage,
  progressionPerWeek,
) {
  return startPercentage + (week - 1) * progressionPerWeek;
}

/**
 * Get recommended weight for current week
 * @param {number} estimated1RM - Base 1RM
 * @param {number} week - Current week (1-indexed)
 * @param {number} startPercentage - Starting % (e.g., 75)
 * @param {number} progressionPerWeek - Weekly % increase (e.g., 5)
 * @returns {number} Recommended weight (rounded to nearest 0.5)
 */
export function getRecommendedWeight(
  estimated1RM,
  week,
  startPercentage,
  progressionPerWeek,
) {
  const percentage = getCurrentPercentage(
    week,
    startPercentage,
    progressionPerWeek,
  );
  const weight = (estimated1RM * percentage) / 100;
  // Round to nearest 0.5
  return Math.round(weight * 2) / 2;
}

/**
 * Get recommended rep range for a week
 * @param {number} week - Current week
 * @param {number} weeksPerBlock - Total weeks in block
 * @returns {Object} Rep range { min, max }
 */
export function getRecommendedReps(week, _weeksPerBlock) {
  // Map week to our predefined ranges, capped at week 6
  const effectiveWeek = Math.min(week, 6);
  return REP_RANGES[effectiveWeek] || REP_RANGES[1];
}

/**
 * Start a new block for an exercise
 * @param {string} exerciseName - Exercise name
 * @param {number} estimated1RM - Starting 1RM
 * @param {Object} settings - Optional per-exercise settings overrides
 * @param {number} settings.startWeek - Week to start at (default 1)
 */
export async function startNewBlock(exerciseName, estimated1RM, settings = {}) {
  const effectiveSettings = await getEffectiveSettings(exerciseName);
  const weeksPerBlock =
    settings.weeksPerBlock ?? effectiveSettings.weeksPerBlock;
  const workoutsPerWeek =
    settings.workoutsPerWeek ?? effectiveSettings.workoutsPerWeek;
  const progressionPerWeek =
    settings.progressionPerWeek ?? effectiveSettings.progressionPerWeek;
  const startWeek = settings.startWeek ?? 1;

  // Calculate target max
  const totalProgression =
    (progressionPerWeek * weeksPerBlock * estimated1RM) / 100;
  const targetMax = estimated1RM + totalProgression;

  // Calculate initial workout count so getCurrentBlockWeek returns startWeek
  // Week 1 = 1 workout count, Week 3 = 3 workout count (for workoutsPerWeek=1)
  const initialWorkoutCount = startWeek * workoutsPerWeek;

  const allBlocks = await getSetting("blockPeriodization_exercises", {});
  allBlocks[exerciseName] = {
    blockStartDate: new Date().toISOString(),
    blockWorkoutCount: initialWorkoutCount,
    startingEstimated1RM: estimated1RM,
    targetMax: targetMax,
    lastWorkoutDate: null,
    // Store per-exercise overrides if provided
    ...(settings.weeksPerBlock !== undefined && {
      weeksPerBlock: settings.weeksPerBlock,
    }),
    ...(settings.workoutsPerWeek !== undefined && {
      workoutsPerWeek: settings.workoutsPerWeek,
    }),
    ...(settings.startPercentage !== undefined && {
      startPercentage: settings.startPercentage,
    }),
    ...(settings.progressionPerWeek !== undefined && {
      progressionPerWeek: settings.progressionPerWeek,
    }),
  };

  await saveSetting("blockPeriodization_exercises", allBlocks);
  return allBlocks[exerciseName];
}

/**
 * Update per-exercise settings
 * @param {string} exerciseName - Exercise name
 * @param {Object} settings - Settings to update (null to reset to global)
 */
export async function updateExerciseSettings(exerciseName, settings) {
  const allBlocks = await getSetting("blockPeriodization_exercises", {});

  if (!allBlocks[exerciseName]) {
    // Create a new block state if none exists
    allBlocks[exerciseName] = {
      blockStartDate: null,
      blockWorkoutCount: 0,
      startingEstimated1RM: null,
      targetMax: null,
      lastWorkoutDate: null,
    };
  }

  // Update or clear settings
  if (settings.weeksPerBlock !== undefined) {
    allBlocks[exerciseName].weeksPerBlock = settings.weeksPerBlock;
  }
  if (settings.workoutsPerWeek !== undefined) {
    allBlocks[exerciseName].workoutsPerWeek = settings.workoutsPerWeek;
  }
  if (settings.startPercentage !== undefined) {
    allBlocks[exerciseName].startPercentage = settings.startPercentage;
  }
  if (settings.progressionPerWeek !== undefined) {
    allBlocks[exerciseName].progressionPerWeek = settings.progressionPerWeek;
  }

  await saveSetting("blockPeriodization_exercises", allBlocks);
}

/**
 * Increment workout count for an exercise (called on workout save)
 * @param {string} exerciseName - Exercise name
 */
export async function incrementBlockWorkout(exerciseName) {
  const blockState = await getExerciseBlockState(exerciseName);
  if (!blockState || !blockState.blockStartDate) return; // No active block

  const allBlocks = await getSetting("blockPeriodization_exercises", {});
  allBlocks[exerciseName] = {
    ...blockState,
    blockWorkoutCount: (blockState.blockWorkoutCount || 0) + 1,
    lastWorkoutDate: new Date().toISOString(),
  };

  await saveSetting("blockPeriodization_exercises", allBlocks);
}

/**
 * Check if a block is complete
 * @param {Object} blockState - Block state
 * @param {number} weeksPerBlock - Weeks per block
 * @param {number} workoutsPerWeek - Workouts per week
 * @returns {boolean} Whether the block is complete
 */
export function isBlockComplete(blockState, weeksPerBlock, workoutsPerWeek) {
  if (!blockState) return false;
  const totalWorkouts = weeksPerBlock * workoutsPerWeek;
  return blockState.blockWorkoutCount >= totalWorkouts;
}

/**
 * Reset all block data
 */
export async function resetAllBlocks() {
  await saveSetting("blockPeriodization_exercises", {});
}

/**
 * Get block data for display
 * @param {string} exerciseName - Exercise name
 * @param {Array} workouts - All workouts for 1RM calculation
 * @param {string} arm - Arm filter
 * @returns {Promise<Object>} Block display data
 */
export async function getBlockDisplayData(exerciseName, workouts, arm) {
  const settings = await getEffectiveSettings(exerciseName);
  const blockState = await getExerciseBlockState(exerciseName);
  const estimated1RM = getBestEstimated1RM(workouts, exerciseName, arm);

  const hasActiveBlock = blockState && blockState.blockStartDate;
  const currentWeek = hasActiveBlock
    ? getCurrentBlockWeek(blockState, settings.workoutsPerWeek)
    : 0;
  const isComplete = hasActiveBlock
    ? isBlockComplete(
        blockState,
        settings.weeksPerBlock,
        settings.workoutsPerWeek,
      )
    : false;

  const recommendedWeight =
    hasActiveBlock && !isComplete
      ? getRecommendedWeight(
          blockState.startingEstimated1RM,
          currentWeek,
          settings.startPercentage,
          settings.progressionPerWeek,
        )
      : 0;

  const currentPercentage =
    hasActiveBlock && !isComplete
      ? getCurrentPercentage(
          currentWeek,
          settings.startPercentage,
          settings.progressionPerWeek,
        )
      : 0;

  const recommendedReps =
    hasActiveBlock && !isComplete
      ? getRecommendedReps(currentWeek, settings.weeksPerBlock)
      : null;

  return {
    hasActiveBlock,
    blockState,
    settings,
    estimated1RM,
    currentWeek,
    isComplete,
    recommendedWeight,
    currentPercentage,
    recommendedReps,
    totalWeeks: settings.weeksPerBlock,
    totalWorkouts: settings.weeksPerBlock * settings.workoutsPerWeek,
  };
}
