/**
 * Shapes of the records the app edits. Stored records come from the API, and
 * the editor also holds unsaved ones, which have no id or revision yet.
 */
import type * as api from "@/api/gen/types.gen";

export type Id = string;

export type MuscleGroup =
  | "Forearm"
  | "Chest"
  | "Back"
  | "Shoulders"
  | "Legs"
  | "Biceps"
  | "Triceps"
  | "Abs";

export type ExerciseType = api.Exercise["type"];
export type DisplayType = api.Exercise["displayType"];
export type Intensity = NonNullable<api.WorkoutExercise["intensity"]>;
export type Arm = api.WorkoutSet["arm"];
export type SetType = api.WorkoutSet["type"];

/** Dates are Date objects when fresh and ISO strings from the API. */
export type DateLike = Date | string;

/** An entry of the exercise list. */
export type Exercise = api.Exercise;

/** An exercise to create or update. */
export type ExerciseDraft = api.ExerciseInput & { id?: Id };

export type WorkoutSet = api.WorkoutSet;

export type WorkoutExercise = api.WorkoutExercise;

export interface Workout {
  id?: Id;
  /** The revision last read from the server, sent back on save. */
  revision?: number;
  name: string;
  started: DateLike;
  ended?: DateLike | null;
  notes: string;
  exercises: WorkoutExercise[];
  plannedSessionId?: Id;
  created?: DateLike;
  updated?: DateLike;
}

export interface WorkoutTemplate {
  id?: Id;
  name: string;
  notes: string;
  exercises: WorkoutExercise[];
  created?: DateLike;
  updated?: DateLike;
}

/** A training plan, an ordered queue of sessions. */
export type Plan = api.Plan;

/** One session of a plan, with its planned exercises. */
export type PlannedSession = api.Session;

export type PlannedExercise = api.PlannedExercise;

/** A group of prescribed sets, e.g. "Back-off, 3 × 5 @ 130". */
export type Target = api.Target;

export type QueueEntry = api.QueueEntry;

/** What a session planned against what its workout logged. */
export type Comparison = api.Comparison;

/** A calendar day of the food diary, the user's own, e.g. 2026-10-02. */
export type DayKey = string;

export type Meal = api.Meal["meal"];

/** Nutrients per 100 g. */
export type Nutrients = api.Nutrients;

/** One of the user's own foods. */
export type Food = api.Food;

export type FoodDraft = api.FoodInput;

/** A household measure of a food, e.g. a slice is 10 g. */
export type Portion = api.Portion;

/** A search result, one of the user's foods or Livsmedelsverket's. */
export type FoodMatch = api.Match;

export type FoodSearch = api.SearchResult;

/** A day of the food diary. */
export type Day = api.Day;

export type DayMeal = api.Meal;

/** Something eaten. */
export type FoodEntry = api.Entry;

export type EntryDraft = api.EntryInput;

/** Energy burned by exercise. */
export type Activity = api.Activity;

export type ActivityDraft = api.ActivityInput;

export type NutritionGoal = api.Goal;

export type NutritionGoalDraft = api.GoalInput;

export type DiarySummary = api.Summary;
