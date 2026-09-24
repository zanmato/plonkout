import { openDB, type DBSchema, type IDBPDatabase } from "idb";
import { defaultExercises } from "@/data/defaultExercises";
import type { Exercise, Id, Workout, WorkoutTemplate } from "@/types/domain";

interface PlonkoutSchema extends DBSchema {
  workouts: {
    key: Id;
    value: Workout;
    indexes: { date: Date; name_date: [string, Date] };
  };
  exercises: {
    key: Id;
    value: Exercise;
    indexes: { name: string; muscleGroup: string };
  };
  settings: {
    key: string;
    value: { key: string; value: unknown };
  };
  workout_templates: {
    key: Id;
    value: WorkoutTemplate;
    indexes: { created: Date; name: string };
  };
}

const DB_NAME = "PlonkoutDB";
const DB_VERSION = 3;

/**
 * Initialize the IndexedDB database
 */
async function initDB(): Promise<IDBPDatabase<PlonkoutSchema>> {
  return openDB<PlonkoutSchema>(DB_NAME, DB_VERSION, {
    upgrade(db, _oldVersion, _newVersion, transaction) {
      // Workouts store
      if (!db.objectStoreNames.contains("workouts")) {
        const workoutStore = db.createObjectStore("workouts", {
          keyPath: "id",
          autoIncrement: true,
        });
        workoutStore.createIndex("date", "started");
        workoutStore.createIndex("name_date", ["name", "started"]);
      } else {
        // Add the compound index if it doesn't exist (for existing databases)
        const workoutStore = transaction.objectStore("workouts");
        if (!workoutStore.indexNames.contains("name_date")) {
          workoutStore.createIndex("name_date", ["name", "started"]);
        }
      }

      // Exercises store (predefined exercises)
      if (!db.objectStoreNames.contains("exercises")) {
        const exerciseStore = db.createObjectStore("exercises", {
          keyPath: "id",
          autoIncrement: true,
        });
        exerciseStore.createIndex("name", "name");
        exerciseStore.createIndex("muscleGroup", "muscleGroup");
      }

      // Settings store
      if (!db.objectStoreNames.contains("settings")) {
        db.createObjectStore("settings", {
          keyPath: "key",
        });
      }

      // Workout templates store
      if (!db.objectStoreNames.contains("workout_templates")) {
        const templateStore = db.createObjectStore("workout_templates", {
          keyPath: "id",
          autoIncrement: true,
        });
        templateStore.createIndex("created", "created");
        templateStore.createIndex("name", "name");
      }
    },
  });
}

/**
 * Get all workouts sorted by date (newest first)
 */
export async function getWorkouts(): Promise<Workout[]> {
  const db = await initDB();
  return db.getAllFromIndex("workouts", "date");
}

/**
 * Get the most recent workout by name using efficient cursor-based query
 */
export async function getMostRecentWorkoutByName(
  workoutName: string,
  excludeId: Id | null = null,
): Promise<Workout | null> {
  if (!workoutName) {
    return null;
  }

  const db = await initDB();
  const index = db.transaction("workouts", "readonly").store.index("name_date");
  const keyRange = IDBKeyRange.bound([workoutName], [workoutName, []]) as IDBKeyRange;

  // Iterate newest first and return the first workout that is not excluded
  for await (const cursor of index.iterate(keyRange, "prev")) {
    if (!excludeId || cursor.value.id !== excludeId) {
      return cursor.value;
    }
  }

  return null;
}

/**
 * Get a single workout by ID
 */
export async function getWorkout(id: Id): Promise<Workout | undefined> {
  const db = await initDB();
  return db.get("workouts", id);
}

/**
 * Save a workout (create or update)
 */
export async function saveWorkout(workout: Workout): Promise<Id> {
  const db = await initDB();
  if (workout.id) {
    await db.put("workouts", workout);
    return workout.id;
  } else {
    // Remove id field for new workouts to avoid IndexedDB key validation errors
    const { id: _id, ...workoutData } = workout;
    return db.add("workouts", {
      ...workoutData,
      created: new Date(),
      updated: new Date(),
    });
  }
}

/**
 * Delete a workout
 */
export async function deleteWorkout(id: Id): Promise<void> {
  const db = await initDB();
  await db.delete("workouts", id);
}

/**
 * Get all predefined exercises
 */
export async function getExercises(): Promise<Exercise[]> {
  const db = await initDB();
  return db.getAll("exercises");
}

/**
 * Save an exercise (create or update)
 */
export async function saveExercise(exercise: Exercise): Promise<Id> {
  const db = await initDB();
  if (exercise.id) {
    await db.put("exercises", exercise);
    return exercise.id;
  } else {
    return db.add("exercises", exercise);
  }
}

/**
 * Get a setting value
 */
export async function getSetting<T>(key: string, defaultValue: T): Promise<T>;
export async function getSetting<T = unknown>(key: string): Promise<T | null>;
export async function getSetting<T>(key: string, defaultValue: T | null = null): Promise<T | null> {
  const db = await initDB();
  const setting = await db.get("settings", key);
  return setting ? (setting.value as T) : defaultValue;
}

/**
 * Save a setting
 */
export async function saveSetting(key: string, value: unknown): Promise<void> {
  const db = await initDB();
  await db.put("settings", { key, value });
}

/**
 * Initialize default exercises
 */
export async function initializeDefaultExercises(): Promise<void> {
  const exercises = await getExercises();
  if (exercises.length === 0) {
    for (const exercise of defaultExercises) {
      await saveExercise(exercise);
    }
  }
}

/**
 * Get all workout templates sorted by creation date (newest first)
 */
export async function getWorkoutTemplates(): Promise<WorkoutTemplate[]> {
  const db = await initDB();
  return db.getAllFromIndex("workout_templates", "created");
}

/**
 * Get a specific workout template by ID
 */
export async function getWorkoutTemplate(id: Id): Promise<WorkoutTemplate | undefined> {
  const db = await initDB();
  return db.get("workout_templates", id);
}

/**
 * Save a workout template to the database
 */
export async function saveWorkoutTemplate(template: WorkoutTemplate): Promise<Id> {
  const db = await initDB();
  const templateData = {
    ...template,
    created: template.created || new Date(),
    updated: new Date(),
  };

  if (template.id) {
    await db.put("workout_templates", templateData);
    return template.id;
  } else {
    return db.add("workout_templates", templateData);
  }
}

/**
 * Delete a workout template from the database
 */
export async function deleteWorkoutTemplate(id: Id): Promise<void> {
  const db = await initDB();
  await db.delete("workout_templates", id);
}
