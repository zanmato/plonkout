/**
 * Workouts, exercises, templates and settings. The function names follow the
 * IndexedDB module this replaced, so the views read the same.
 *
 * Request bodies are built field by field: the server refuses unknown fields,
 * and editor state carries a few of its own.
 */
import * as sdk from "./gen";
import type {
  Exercise,
  ExerciseDraft,
  Id,
  Workout,
  WorkoutExercise,
  WorkoutSet,
  WorkoutTemplate,
} from "@/types/domain";
import type { Result, Template, Workout as ApiWorkout } from "./gen/types.gen";
import { ApiError, unwrap } from "./index";

function iso(date: Date | string): string {
  return new Date(date).toISOString();
}

function cleanSet(set: WorkoutSet): WorkoutSet {
  return {
    type: set.type === "warmup" ? "warmup" : "regular",
    weight: numberOrNull(set.weight),
    distance: numberOrNull(set.distance),
    reps: set.reps === null || set.reps === undefined || Number.isNaN(Number(set.reps)) ? null : Math.round(Number(set.reps)),
    time: set.time ?? "",
    rpe: numberOrNull(set.rpe),
    arm: set.arm ?? "",
    notes: set.notes ?? "",
    ...(set.targetId ? { targetId: set.targetId, targetSeq: set.targetSeq } : {}),
  };
}

function numberOrNull(value: unknown): number | null {
  if (value === null || value === undefined || value === "") return null;
  const n = Number(value);
  return Number.isFinite(n) ? n : null;
}

/** The fields of a logged exercise, from a workout exercise or an exercise list entry. */
export function cleanExercise(exercise: WorkoutExercise): WorkoutExercise {
  return {
    ...(exercise.exerciseId ? { exerciseId: exercise.exerciseId } : {}),
    name: exercise.name,
    muscleGroup: exercise.muscleGroup ?? "",
    type: exercise.type ?? "strength",
    displayType: exercise.displayType ?? "reps",
    singleArm: Boolean(exercise.singleArm),
    intensity: exercise.intensity ?? null,
    ...(exercise.notes ? { notes: exercise.notes } : {}),
    ...(exercise.plannedExerciseId ? { plannedExerciseId: exercise.plannedExerciseId } : {}),
    sets: (exercise.sets ?? []).map(cleanSet),
  };
}

/** Starts a logged exercise from an entry of the exercise list. */
export function toWorkoutExercise(exercise: Exercise, sets: WorkoutSet[] = []): WorkoutExercise {
  return {
    exerciseId: exercise.id,
    name: exercise.name,
    muscleGroup: exercise.muscleGroup,
    type: exercise.type,
    displayType: exercise.displayType,
    singleArm: exercise.singleArm,
    intensity: null,
    sets,
  };
}

function fromApi(workout: ApiWorkout): Workout {
  return { ...workout };
}

// Workouts

export async function getWorkouts(): Promise<Workout[]> {
  return (await unwrap(sdk.listWorkouts())).map(fromApi);
}

export async function getWorkout(id: Id): Promise<Workout | undefined> {
  try {
    return fromApi(await unwrap(sdk.getWorkout({ path: { id } })));
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) return undefined;
    throw error;
  }
}

/** The most recent workout with this name, skipping the one being edited. */
export async function getMostRecentWorkoutByName(
  name: string,
  excludeId: Id | null = null,
): Promise<Workout | null> {
  if (!name) return null;
  try {
    return fromApi(
      await unwrap(
        sdk.getLatestWorkout({ query: { name, ...(excludeId ? { excludeId } : {}) } }),
      ),
    );
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) return null;
    throw error;
  }
}

/**
 * Creates or replaces a workout and answers with what was stored, including
 * its id and new revision. A workout changed elsewhere since it was loaded
 * throws an ApiError with code "stale_revision".
 */
export async function saveWorkout(workout: Workout): Promise<Workout> {
  const body = {
    name: workout.name ?? "",
    started: iso(workout.started),
    ended: workout.ended ? iso(workout.ended) : null,
    notes: workout.notes ?? "",
    exercises: workout.exercises.map(cleanExercise),
  };
  if (workout.id) {
    return fromApi(
      await unwrap(
        sdk.updateWorkout({ path: { id: workout.id }, body: { ...body, revision: workout.revision ?? 1 } }),
      ),
    );
  }
  return fromApi(await unwrap(sdk.createWorkout({ body })));
}

export async function deleteWorkout(id: Id): Promise<void> {
  await unwrap(sdk.deleteWorkout({ path: { id } }));
}

// Exercises

/** The exercise list, without archived exercises. */
export async function getExercises(): Promise<Exercise[]> {
  return (await unwrap(sdk.listExercises())).filter((exercise) => !exercise.archived);
}

export async function saveExercise(exercise: ExerciseDraft): Promise<Exercise> {
  const body = {
    name: exercise.name.trim(),
    muscleGroup: exercise.muscleGroup,
    type: exercise.type,
    displayType: exercise.displayType,
    singleArm: exercise.singleArm,
    archived: exercise.archived ?? false,
  };
  if (exercise.id) {
    return unwrap(sdk.updateExercise({ path: { id: exercise.id }, body }));
  }
  return unwrap(sdk.createExercise({ body }));
}

// Templates

function fromTemplate(template: Template): WorkoutTemplate {
  return { ...template };
}

export async function getWorkoutTemplates(): Promise<WorkoutTemplate[]> {
  return (await unwrap(sdk.listTemplates())).map(fromTemplate);
}

export async function getWorkoutTemplate(id: Id): Promise<WorkoutTemplate | undefined> {
  try {
    return fromTemplate(await unwrap(sdk.getTemplate({ path: { id } })));
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) return undefined;
    throw error;
  }
}

export async function saveWorkoutTemplate(template: WorkoutTemplate): Promise<WorkoutTemplate> {
  const body = {
    name: template.name ?? "",
    notes: template.notes ?? "",
    exercises: template.exercises.map(cleanExercise),
  };
  if (template.id) {
    return fromTemplate(await unwrap(sdk.updateTemplate({ path: { id: template.id }, body })));
  }
  return fromTemplate(await unwrap(sdk.createTemplate({ body })));
}

export async function deleteWorkoutTemplate(id: Id): Promise<void> {
  await unwrap(sdk.deleteTemplate({ path: { id } }));
}

// Settings

// Loaded once per session and kept in step with every save, since the editor
// reads a few settings per exercise.
let settings: Promise<Record<string, unknown>> | null = null;

function loadSettings(): Promise<Record<string, unknown>> {
  if (!settings) {
    settings = unwrap(sdk.listSettings()).catch((error) => {
      settings = null;
      throw error;
    });
  }
  return settings;
}

export async function getSetting<T>(key: string, defaultValue: T): Promise<T>;
export async function getSetting<T = unknown>(key: string): Promise<T | null>;
export async function getSetting<T>(key: string, defaultValue: T | null = null): Promise<T | null> {
  const all = await loadSettings();
  return key in all ? (all[key] as T) : defaultValue;
}

export async function saveSetting(key: string, value: unknown): Promise<void> {
  await unwrap(sdk.putSetting({ path: { key }, body: { value } }));
  const all = await loadSettings();
  all[key] = value;
}

/** Forgets cached settings, e.g. when another user signs in. */
export function clearSettingsCache(): void {
  settings = null;
}

// Import and export

/** Imports the file the local only app exported. */
export function importLegacyExport(file: unknown): Promise<Result> {
  return unwrap(sdk.importLegacyExport({ body: file as Parameters<typeof sdk.importLegacyExport>[0]["body"] }));
}

/** Everything in the account, for a backup file. */
export async function exportAll() {
  const [workouts, templates, exercises, allSettings] = await Promise.all([
    unwrap(sdk.listWorkouts()),
    unwrap(sdk.listTemplates()),
    unwrap(sdk.listExercises()),
    unwrap(sdk.listSettings()),
  ]);
  return {
    format: "plonkout-backup-v2",
    exportDate: new Date().toISOString(),
    workouts,
    templates,
    exercises,
    settings: allSettings,
  };
}
