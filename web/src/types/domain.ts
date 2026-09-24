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
