/**
 * Block Periodization Utility
 * Provides functions for calculating progressive overload recommendations
 * based on block periodization principles.
 */

import { getSetting, saveSetting } from "./database";
import {
  buildExerciseHistory,
  getMaxWeight as historyMaxWeight,
  getBestEstimated1RM as historyBestEstimated1RM,
} from "./exerciseHistory";
import type { Arm, Id, Workout } from "@/types/domain";

export interface BlockSettings {
  weeksPerBlock: number;
  workoutsPerWeek: number;
  startPercentage: number;
  progressionPerWeek: number;
}

/** Per exercise overrides, null clears an override and falls back to global. */
export type BlockSettingOverrides = {
  [K in keyof BlockSettings]?: number | null;
};

export interface BlockState extends BlockSettingOverrides {
  blockStartDate: string | null;
  blockWorkoutCount: number;
  startingEstimated1RM: number | null;
  targetMax: number | null;
  lastWorkoutDate: string | null;
}

export type BlockStates = Record<string, BlockState>;

export interface RepRange {
  min: number;
  max: number;
}

const EXERCISES_KEY = "blockPeriodization_exercises";

// Rep ranges for each week of the block
const REP_RANGES: Record<number, RepRange> = {
  1: { min: 8, max: 10 }, // 75%
  2: { min: 6, max: 8 }, // 80%
  3: { min: 5, max: 6 }, // 85%
  4: { min: 4, max: 5 }, // 90%
  5: { min: 3, max: 4 }, // 95%
  6: { min: 1, max: 3 }, // 100%+ (max test)
};

// Default global settings
const DEFAULT_SETTINGS: BlockSettings = {
  weeksPerBlock: 5,
  workoutsPerWeek: 1,
  startPercentage: 75,
  progressionPerWeek: 5,
};

export {
  calculateEpley1RM,
  isArmCompatible,
} from "./exerciseHistory";

/**
 * Get the maximum weight lifted for an exercise
 */
export function getMaxWeight(
  workouts: Workout[],
  exerciseName: string,
  arm?: Arm | null,
  excludeWorkoutId?: Id | null,
): number {
  const entries = buildExerciseHistory(workouts).get(exerciseName);
  return historyMaxWeight(entries, { arm, excludeWorkoutId });
}

/**
 * Get the best estimated 1RM from all historical sets
 * Uses max weight as primary, Epley estimate as fallback
 */
export function getBestEstimated1RM(
  workouts: Workout[],
  exerciseName: string,
  arm?: Arm | null,
  excludeWorkoutId?: Id | null,
): number {
  const entries = buildExerciseHistory(workouts).get(exerciseName);
  return historyBestEstimated1RM(entries, { arm, excludeWorkoutId });
}

/**
 * Get global block periodization settings
 */
export async function getGlobalSettings(): Promise<BlockSettings> {
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
 */
export async function saveGlobalSettings(settings: Partial<BlockSettings>): Promise<void> {
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
 */
export async function getExerciseBlockState(exerciseName: string): Promise<BlockState | null> {
  const allBlocks = await getBlockStates();
  return allBlocks[exerciseName] || null;
}

/**
 * Get effective settings for an exercise (merges global + per-exercise overrides)
 */
export async function getEffectiveSettings(exerciseName: string): Promise<BlockSettings> {
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
 */
export function getCurrentBlockWeek(
  blockState: BlockState | null,
  workoutsPerWeek: number,
): number {
  if (!blockState) return 0;
  if (blockState.blockWorkoutCount <= 0) return 1;
  return Math.ceil(blockState.blockWorkoutCount / workoutsPerWeek);
}

/**
 * Calculate current percentage based on week
 */
export function getCurrentPercentage(
  week: number,
  startPercentage: number,
  progressionPerWeek: number,
): number {
  return startPercentage + (week - 1) * progressionPerWeek;
}

/**
 * Get recommended weight for current week
 */
export function getRecommendedWeight(
  estimated1RM: number,
  week: number,
  startPercentage: number,
  progressionPerWeek: number,
): number {
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
 */
export function getRecommendedReps(week: number, _weeksPerBlock?: number): RepRange {
  // Map week to our predefined ranges, capped at week 6
  const effectiveWeek = Math.min(week, 6);
  return REP_RANGES[effectiveWeek] ?? { min: 8, max: 10 };
}

/**
 * Start a new block for an exercise
 */
export async function startNewBlock(
  exerciseName: string,
  estimated1RM: number,
  settings: Partial<BlockSettings> & { startWeek?: number } = {},
): Promise<BlockState> {
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

  const allBlocks = await getBlockStates();
  const state: BlockState = {
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
  allBlocks[exerciseName] = state;

  await saveSetting(EXERCISES_KEY, allBlocks);
  return state;
}

/**
 * Update per-exercise settings
 */
export async function updateExerciseSettings(
  exerciseName: string,
  settings: BlockSettingOverrides,
): Promise<void> {
  const allBlocks = await getBlockStates();

  // Create a new block state if none exists
  const state: BlockState = allBlocks[exerciseName] ?? {
      blockStartDate: null,
      blockWorkoutCount: 0,
      startingEstimated1RM: null,
      targetMax: null,
      lastWorkoutDate: null,
    };
  allBlocks[exerciseName] = state;

  // Update or clear settings
  if (settings.weeksPerBlock !== undefined) {
    state.weeksPerBlock = settings.weeksPerBlock;
  }
  if (settings.workoutsPerWeek !== undefined) {
    state.workoutsPerWeek = settings.workoutsPerWeek;
  }
  if (settings.startPercentage !== undefined) {
    state.startPercentage = settings.startPercentage;
  }
  if (settings.progressionPerWeek !== undefined) {
    state.progressionPerWeek = settings.progressionPerWeek;
  }

  await saveSetting(EXERCISES_KEY, allBlocks);
}

/**
 * Increment workout count for an exercise (called on workout save)
 */
export async function incrementBlockWorkout(exerciseName: string): Promise<void> {
  const blockState = await getExerciseBlockState(exerciseName);
  if (!blockState || !blockState.blockStartDate) return; // No active block

  const allBlocks = await getBlockStates();
  allBlocks[exerciseName] = {
    ...blockState,
    blockWorkoutCount: (blockState.blockWorkoutCount || 0) + 1,
    lastWorkoutDate: new Date().toISOString(),
  };

  await saveSetting(EXERCISES_KEY, allBlocks);
}

/**
 * Check if a block is complete
 */
export function isBlockComplete(
  blockState: BlockState | null,
  weeksPerBlock: number,
  workoutsPerWeek: number,
): boolean {
  if (!blockState) return false;
  const totalWorkouts = weeksPerBlock * workoutsPerWeek;
  return blockState.blockWorkoutCount >= totalWorkouts;
}

/**
 * Reset all block data
 */
export async function resetAllBlocks(): Promise<void> {
  await saveSetting(EXERCISES_KEY, {});
}

async function getBlockStates(): Promise<BlockStates> {
  return getSetting<BlockStates>(EXERCISES_KEY, {});
}

/**
 * Get block data for display
 */
export async function getBlockDisplayData(
  exerciseName: string,
  workouts: Workout[],
  arm?: Arm | null,
) {
  const settings = await getEffectiveSettings(exerciseName);
  const blockState = await getExerciseBlockState(exerciseName);
  const estimated1RM = getBestEstimated1RM(workouts, exerciseName, arm);

  const activeBlock = blockState?.blockStartDate ? blockState : null;
  const hasActiveBlock = activeBlock !== null;
  const currentWeek = hasActiveBlock
    ? getCurrentBlockWeek(activeBlock, settings.workoutsPerWeek)
    : 0;
  const isComplete = hasActiveBlock
    ? isBlockComplete(
        activeBlock,
        settings.weeksPerBlock,
        settings.workoutsPerWeek,
      )
    : false;

  const recommendedWeight =
    activeBlock && !isComplete
      ? getRecommendedWeight(
          activeBlock.startingEstimated1RM ?? 0,
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
