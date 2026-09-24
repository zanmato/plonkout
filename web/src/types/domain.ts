/**
 * Shapes of the records the app stores and edits.
 *
 * Ids are an alias so the switch from IndexedDB autoincrement numbers to
 * server uuids is a one line change.
 */
export type Id = number;

export type MuscleGroup =
  | "Forearm"
  | "Chest"
  | "Back"
  | "Shoulders"
  | "Legs"
  | "Biceps"
  | "Triceps"
  | "Abs";

export type ExerciseType = "strength" | "cardio";
export type DisplayType = "reps" | "time";
export type Intensity = "heavy" | "light";
export type Arm = "" | "left" | "right" | "both";
export type SetType = "regular" | "warmup";

/** Dates are Date objects when fresh and ISO strings after a JSON round trip. */
export type DateLike = Date | string;

export interface Exercise {
  id?: Id;
  name: string;
  muscleGroup: MuscleGroup | string;
  singleArm: boolean;
  type: ExerciseType;
  displayType: DisplayType;
}

export interface WorkoutSet {
  type: SetType;
  weight: number | null;
  distance: number | null;
  reps: number | null;
  time: string;
  rpe: number | null;
  arm: Arm;
  notes: string;
}

export interface WorkoutExercise extends Exercise {
  intensity?: Intensity | null;
  sets: WorkoutSet[];
}

export interface Workout {
  id?: Id;
  name: string;
  started: DateLike;
  ended?: DateLike | null;
  notes: string;
  exercises: WorkoutExercise[];
  created?: DateLike;
  updated?: DateLike;
}

export type WorkoutTemplate = Workout;

export interface Setting<T = unknown> {
  key: string;
  value: T;
}
