import { openDB } from "idb";
import { defaultExercises } from "@/data/defaultExercises.js";

const DB_NAME = "PlonkoutDB";
const DB_VERSION = 3;

/**
 * Initialize the IndexedDB database
 * @returns {Promise<IDBDatabase>} The database instance
 */
async function initDB() {
  return openDB(DB_NAME, DB_VERSION, {
    upgrade(db, oldVersion, newVersion, transaction) {
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
 * @returns {Promise<Array>} Array of workout objects
 */
export async function getWorkouts() {
  const db = await initDB();
  return db.getAllFromIndex("workouts", "date");
}

/**
 * Get the most recent workout by name using efficient cursor-based query
 * @param {string} workoutName - The workout name to search for
 * @param {number} excludeId - Optional workout ID to exclude from results
 * @returns {Promise<Object|null>} The most recent workout object or null
 */
export async function getMostRecentWorkoutByName(workoutName, excludeId = null) {
  if (!workoutName) {
    return null;
  }

  const db = await initDB();
  const index = db.transaction("workouts", "readonly").store.index("name_date");
  const keyRange = IDBKeyRange.bound([workoutName], [workoutName, []]);

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
 * @param {number} id - The workout ID
 * @returns {Promise<Object|undefined>} The workout object or undefined
 */
export async function getWorkout(id) {
  const db = await initDB();
  return db.get("workouts", id);
}

/**
 * Save a workout (create or update)
 * @param {Object} workout - The workout object
 * @returns {Promise<number>} The workout ID
 */
export async function saveWorkout(workout) {
  const db = await initDB();
  if (workout.id) {
    await db.put("workouts", workout);
    return workout.id;
  } else {
    // Remove id field for new workouts to avoid IndexedDB key validation errors
     
    const { id, ...workoutData } = workout;
    return db.add("workouts", {
      ...workoutData,
      created: new Date(),
      updated: new Date(),
    });
  }
}

/**
 * Delete a workout
 * @param {number} id - The workout ID
 * @returns {Promise<void>}
 */
export async function deleteWorkout(id) {
  const db = await initDB();
  return db.delete("workouts", id);
}

/**
 * Get all predefined exercises
 * @returns {Promise<Array>} Array of exercise objects
 */
export async function getExercises() {
  const db = await initDB();
  return db.getAll("exercises");
}

/**
 * Save an exercise (create or update)
 * @param {Object} exercise - The exercise object
 * @returns {Promise<number>} The exercise ID
 */
export async function saveExercise(exercise) {
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
 * @param {string} key - The setting key
 * @param {*} defaultValue - Default value if setting doesn't exist
 * @returns {Promise<*>} The setting value
 */
export async function getSetting(key, defaultValue = null) {
  const db = await initDB();
  const setting = await db.get("settings", key);
  return setting ? setting.value : defaultValue;
}

/**
 * Save a setting
 * @param {string} key - The setting key
 * @param {*} value - The setting value
 * @returns {Promise<void>}
 */
export async function saveSetting(key, value) {
  const db = await initDB();
  return db.put("settings", { key, value });
}

/**
 * Initialize default exercises
 * @returns {Promise<void>}
 */
export async function initializeDefaultExercises() {
  const exercises = await getExercises();
  if (exercises.length === 0) {
    for (const exercise of defaultExercises) {
      await saveExercise(exercise);
    }
  }
}

/**
 * Get all workout templates sorted by creation date (newest first)
 * @returns {Promise<Array>} Array of template objects
 */
export async function getWorkoutTemplates() {
  const db = await initDB();
  return db.getAllFromIndex("workout_templates", "created");
}

/**
 * Get a specific workout template by ID
 * @param {number} id - The template ID
 * @returns {Promise<Object|null>} The template object or null if not found
 */
export async function getWorkoutTemplate(id) {
  const db = await initDB();
  return db.get("workout_templates", id);
}

/**
 * Save a workout template to the database
 * @param {Object} template - The template object to save
 * @returns {Promise<number>} The template ID
 */
export async function saveWorkoutTemplate(template) {
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
 * @param {number} id - The template ID to delete
 * @returns {Promise<void>}
 */
export async function deleteWorkoutTemplate(id) {
  const db = await initDB();
  return db.delete("workout_templates", id);
}
