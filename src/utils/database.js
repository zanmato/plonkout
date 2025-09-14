import { openDB } from "idb";

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
  const tx = db.transaction("workouts", "readonly");
  const index = tx.objectStore("workouts").index("name_date");

  // Use a key range to find all workouts with the specified name
  const keyRange = IDBKeyRange.bound([workoutName], [workoutName, []]);

  let mostRecentWorkout = null;
  let mostRecentDate = null;

  // Open cursor in reverse order (newest first)
  for await (const cursor of index.iterate(keyRange, "prev")) {
    const workout = cursor.value;

    // Skip if this is the workout we want to exclude
    if (excludeId && workout.id === excludeId) {
      continue;
    }

    // Get the most recent date (ended or started)
    const workoutDate = new Date(workout.ended || workout.started);

    if (!mostRecentWorkout || workoutDate > mostRecentDate) {
      mostRecentWorkout = workout;
      mostRecentDate = workoutDate;
    }

    // Since we're iterating in reverse order by started date,
    // the first valid match should be the most recent
    break;
  }

  return mostRecentWorkout;
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
    // eslint-disable-next-line no-unused-vars
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
    const defaultExercises = [
      // Armwrestling specific exercises
      {
        name: "Wrist Curl",
        muscleGroup: "Forearm",
        singleArm: true,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Pronation Curl",
        muscleGroup: "Forearm",
        singleArm: true,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Supination Curl",
        muscleGroup: "Forearm",
        singleArm: true,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Side Pressure",
        muscleGroup: "Shoulders",
        singleArm: true,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Hook Training",
        muscleGroup: "Forearm",
        singleArm: true,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Top Roll Training",
        muscleGroup: "Forearm",
        singleArm: true,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Cable Hammer Curl",
        muscleGroup: "Biceps",
        singleArm: true,
        type: "strength",
        displayType: "reps",
      },

      // Chest exercises
      {
        name: "Bench Press",
        muscleGroup: "Chest",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Incline Bench Press",
        muscleGroup: "Chest",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Decline Bench Press",
        muscleGroup: "Chest",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Close Grip Bench Press",
        muscleGroup: "Triceps",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Dumbbell Press",
        muscleGroup: "Chest",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Incline Dumbbell Press",
        muscleGroup: "Chest",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Chest Fly",
        muscleGroup: "Chest",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Push-ups",
        muscleGroup: "Chest",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Dips",
        muscleGroup: "Chest",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },

      // Back exercises
      {
        name: "Deadlift",
        muscleGroup: "Back",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Sumo Deadlift",
        muscleGroup: "Back",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Romanian Deadlift",
        muscleGroup: "Back",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Bent Over Row",
        muscleGroup: "Back",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "T-Bar Row",
        muscleGroup: "Back",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Cable Row",
        muscleGroup: "Back",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Lat Pulldown",
        muscleGroup: "Back",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Pull-ups",
        muscleGroup: "Back",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Chin-ups",
        muscleGroup: "Back",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Single Arm Row",
        muscleGroup: "Back",
        singleArm: true,
        type: "strength",
        displayType: "reps",
      },

      // Shoulder exercises
      {
        name: "Shoulder Press",
        muscleGroup: "Shoulders",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Overhead Press",
        muscleGroup: "Shoulders",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Dumbbell Shoulder Press",
        muscleGroup: "Shoulders",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Lateral Raise",
        muscleGroup: "Shoulders",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Front Raise",
        muscleGroup: "Shoulders",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Rear Delt Fly",
        muscleGroup: "Shoulders",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Upright Row",
        muscleGroup: "Shoulders",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Shrugs",
        muscleGroup: "Shoulders",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },

      // Leg exercises
      {
        name: "Squat",
        muscleGroup: "Legs",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Front Squat",
        muscleGroup: "Legs",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Bulgarian Split Squat",
        muscleGroup: "Legs",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Leg Press",
        muscleGroup: "Legs",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Leg Curl",
        muscleGroup: "Legs",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Leg Extension",
        muscleGroup: "Legs",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Calf Raise",
        muscleGroup: "Legs",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Lunges",
        muscleGroup: "Legs",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },

      // Biceps exercises
      {
        name: "Bicep Curl",
        muscleGroup: "Biceps",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Hammer Curl",
        muscleGroup: "Biceps",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Preacher Curl",
        muscleGroup: "Biceps",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Cable Curl",
        muscleGroup: "Biceps",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Concentration Curl",
        muscleGroup: "Biceps",
        singleArm: true,
        type: "strength",
        displayType: "reps",
      },

      // Triceps exercises
      {
        name: "Tricep Extension",
        muscleGroup: "Triceps",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Overhead Tricep Extension",
        muscleGroup: "Triceps",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Tricep Pushdown",
        muscleGroup: "Triceps",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Diamond Push-ups",
        muscleGroup: "Triceps",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },

      // Core/Abs exercises (mix of reps and time-based)
      {
        name: "Plank",
        muscleGroup: "Abs",
        singleArm: false,
        type: "strength",
        displayType: "time",
      },
      {
        name: "Crunches",
        muscleGroup: "Abs",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Russian Twists",
        muscleGroup: "Abs",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Leg Raises",
        muscleGroup: "Abs",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },
      {
        name: "Mountain Climbers",
        muscleGroup: "Abs",
        singleArm: false,
        type: "cardio",
        displayType: "time",
      },
      {
        name: "Dead Bug",
        muscleGroup: "Abs",
        singleArm: false,
        type: "strength",
        displayType: "reps",
      },

      // Cardio exercises (typically time-based)
      {
        name: "Running",
        muscleGroup: "Legs",
        singleArm: false,
        type: "cardio",
        displayType: "time",
      },
      {
        name: "Walking",
        muscleGroup: "Legs",
        singleArm: false,
        type: "cardio",
        displayType: "time",
      },
      {
        name: "Jogging",
        muscleGroup: "Legs",
        singleArm: false,
        type: "cardio",
        displayType: "time",
      },
      {
        name: "Cycling",
        muscleGroup: "Legs",
        singleArm: false,
        type: "cardio",
        displayType: "time",
      },
      {
        name: "Rowing Machine",
        muscleGroup: "Back",
        singleArm: false,
        type: "cardio",
        displayType: "time",
      },
      {
        name: "Elliptical",
        muscleGroup: "Legs",
        singleArm: false,
        type: "cardio",
        displayType: "time",
      },
      {
        name: "Swimming",
        muscleGroup: "Back",
        singleArm: false,
        type: "cardio",
        displayType: "time",
      },
      {
        name: "Jumping Jacks",
        muscleGroup: "Legs",
        singleArm: false,
        type: "cardio",
        displayType: "reps",
      },
      {
        name: "Burpees",
        muscleGroup: "Chest",
        singleArm: false,
        type: "cardio",
        displayType: "reps",
      },
      {
        name: "High Knees",
        muscleGroup: "Legs",
        singleArm: false,
        type: "cardio",
        displayType: "time",
      },
    ];

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
