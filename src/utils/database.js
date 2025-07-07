import { openDB } from 'idb'

const DB_NAME = 'PlonkoutDB'
const DB_VERSION = 2

/**
 * Initialize the IndexedDB database
 * @returns {Promise<IDBDatabase>} The database instance
 */
async function initDB() {
  return openDB(DB_NAME, DB_VERSION, {
    upgrade(db) {
      // Workouts store
      if (!db.objectStoreNames.contains('workouts')) {
        const workoutStore = db.createObjectStore('workouts', {
          keyPath: 'id',
          autoIncrement: true
        })
        workoutStore.createIndex('date', 'started')
      }

      // Exercises store (predefined exercises)
      if (!db.objectStoreNames.contains('exercises')) {
        const exerciseStore = db.createObjectStore('exercises', {
          keyPath: 'id',
          autoIncrement: true
        })
        exerciseStore.createIndex('name', 'name')
        exerciseStore.createIndex('muscleGroup', 'muscleGroup')
      }

      // Settings store
      if (!db.objectStoreNames.contains('settings')) {
        db.createObjectStore('settings', {
          keyPath: 'key'
        })
      }

      // Workout templates store
      if (!db.objectStoreNames.contains('workout_templates')) {
        const templateStore = db.createObjectStore('workout_templates', {
          keyPath: 'id',
          autoIncrement: true
        })
        templateStore.createIndex('created', 'created')
        templateStore.createIndex('name', 'name')
      }
    }
  })
}

/**
 * Get all workouts sorted by date (newest first)
 * @returns {Promise<Array>} Array of workout objects
 */
export async function getWorkouts() {
  const db = await initDB()
  return db.getAllFromIndex('workouts', 'date')
}

/**
 * Get a single workout by ID
 * @param {number} id - The workout ID
 * @returns {Promise<Object|undefined>} The workout object or undefined
 */
export async function getWorkout(id) {
  const db = await initDB()
  return db.get('workouts', id)
}

/**
 * Save a workout (create or update)
 * @param {Object} workout - The workout object
 * @returns {Promise<number>} The workout ID
 */
export async function saveWorkout(workout) {
  const db = await initDB()
  if (workout.id) {
    await db.put('workouts', workout)
    return workout.id
  } else {
    return db.add('workouts', {
      ...workout,
      created: new Date(),
      updated: new Date()
    })
  }
}

/**
 * Delete a workout
 * @param {number} id - The workout ID
 * @returns {Promise<void>}
 */
export async function deleteWorkout(id) {
  const db = await initDB()
  return db.delete('workouts', id)
}

/**
 * Get all predefined exercises
 * @returns {Promise<Array>} Array of exercise objects
 */
export async function getExercises() {
  const db = await initDB()
  return db.getAll('exercises')
}

/**
 * Save an exercise (create or update)
 * @param {Object} exercise - The exercise object
 * @returns {Promise<number>} The exercise ID
 */
export async function saveExercise(exercise) {
  const db = await initDB()
  if (exercise.id) {
    await db.put('exercises', exercise)
    return exercise.id
  } else {
    return db.add('exercises', exercise)
  }
}

/**
 * Get a setting value
 * @param {string} key - The setting key
 * @param {*} defaultValue - Default value if setting doesn't exist
 * @returns {Promise<*>} The setting value
 */
export async function getSetting(key, defaultValue = null) {
  const db = await initDB()
  const setting = await db.get('settings', key)
  return setting ? setting.value : defaultValue
}

/**
 * Save a setting
 * @param {string} key - The setting key
 * @param {*} value - The setting value
 * @returns {Promise<void>}
 */
export async function saveSetting(key, value) {
  const db = await initDB()
  return db.put('settings', { key, value })
}

/**
 * Initialize default exercises
 * @returns {Promise<void>}
 */
export async function initializeDefaultExercises() {
  const exercises = await getExercises()
  if (exercises.length === 0) {
    const defaultExercises = [
      // Armwrestling specific exercises
      { name: 'Wrist Curl', muscleGroup: 'Forearm', singleArm: true },
      { name: 'Pronation Curl', muscleGroup: 'Forearm', singleArm: true },
      { name: 'Supination Curl', muscleGroup: 'Forearm', singleArm: true },
      { name: 'Side Pressure', muscleGroup: 'Shoulders', singleArm: true },
      { name: 'Hook Training', muscleGroup: 'Forearm', singleArm: true },
      { name: 'Top Roll Training', muscleGroup: 'Forearm', singleArm: true },
      { name: 'Cable Hammer Curl', muscleGroup: 'Biceps', singleArm: true },
      
      // Chest exercises
      { name: 'Bench Press', muscleGroup: 'Chest', singleArm: false },
      { name: 'Incline Bench Press', muscleGroup: 'Chest', singleArm: false },
      { name: 'Decline Bench Press', muscleGroup: 'Chest', singleArm: false },
      { name: 'Close Grip Bench Press', muscleGroup: 'Triceps', singleArm: false },
      { name: 'Dumbbell Press', muscleGroup: 'Chest', singleArm: false },
      { name: 'Incline Dumbbell Press', muscleGroup: 'Chest', singleArm: false },
      { name: 'Chest Fly', muscleGroup: 'Chest', singleArm: false },
      { name: 'Push-ups', muscleGroup: 'Chest', singleArm: false },
      { name: 'Dips', muscleGroup: 'Chest', singleArm: false },
      
      // Back exercises
      { name: 'Deadlift', muscleGroup: 'Back', singleArm: false },
      { name: 'Sumo Deadlift', muscleGroup: 'Back', singleArm: false },
      { name: 'Romanian Deadlift', muscleGroup: 'Back', singleArm: false },
      { name: 'Bent Over Row', muscleGroup: 'Back', singleArm: false },
      { name: 'T-Bar Row', muscleGroup: 'Back', singleArm: false },
      { name: 'Cable Row', muscleGroup: 'Back', singleArm: false },
      { name: 'Lat Pulldown', muscleGroup: 'Back', singleArm: false },
      { name: 'Pull-ups', muscleGroup: 'Back', singleArm: false },
      { name: 'Chin-ups', muscleGroup: 'Back', singleArm: false },
      { name: 'Single Arm Row', muscleGroup: 'Back', singleArm: true },
      
      // Shoulder exercises
      { name: 'Shoulder Press', muscleGroup: 'Shoulders', singleArm: false },
      { name: 'Overhead Press', muscleGroup: 'Shoulders', singleArm: false },
      { name: 'Dumbbell Shoulder Press', muscleGroup: 'Shoulders', singleArm: false },
      { name: 'Lateral Raise', muscleGroup: 'Shoulders', singleArm: false },
      { name: 'Front Raise', muscleGroup: 'Shoulders', singleArm: false },
      { name: 'Rear Delt Fly', muscleGroup: 'Shoulders', singleArm: false },
      { name: 'Upright Row', muscleGroup: 'Shoulders', singleArm: false },
      { name: 'Shrugs', muscleGroup: 'Shoulders', singleArm: false },
      
      // Leg exercises
      { name: 'Squat', muscleGroup: 'Legs', singleArm: false },
      { name: 'Front Squat', muscleGroup: 'Legs', singleArm: false },
      { name: 'Bulgarian Split Squat', muscleGroup: 'Legs', singleArm: false },
      { name: 'Leg Press', muscleGroup: 'Legs', singleArm: false },
      { name: 'Leg Curl', muscleGroup: 'Legs', singleArm: false },
      { name: 'Leg Extension', muscleGroup: 'Legs', singleArm: false },
      { name: 'Calf Raise', muscleGroup: 'Legs', singleArm: false },
      { name: 'Lunges', muscleGroup: 'Legs', singleArm: false },
      
      // Biceps exercises
      { name: 'Bicep Curl', muscleGroup: 'Biceps', singleArm: false },
      { name: 'Hammer Curl', muscleGroup: 'Biceps', singleArm: false },
      { name: 'Preacher Curl', muscleGroup: 'Biceps', singleArm: false },
      { name: 'Cable Curl', muscleGroup: 'Biceps', singleArm: false },
      { name: 'Concentration Curl', muscleGroup: 'Biceps', singleArm: true },
      
      // Triceps exercises
      { name: 'Tricep Extension', muscleGroup: 'Triceps', singleArm: false },
      { name: 'Overhead Tricep Extension', muscleGroup: 'Triceps', singleArm: false },
      { name: 'Tricep Pushdown', muscleGroup: 'Triceps', singleArm: false },
      { name: 'Diamond Push-ups', muscleGroup: 'Triceps', singleArm: false },
      
      // Core/Abs exercises
      { name: 'Plank', muscleGroup: 'Abs', singleArm: false },
      { name: 'Crunches', muscleGroup: 'Abs', singleArm: false },
      { name: 'Russian Twists', muscleGroup: 'Abs', singleArm: false },
      { name: 'Leg Raises', muscleGroup: 'Abs', singleArm: false },
      { name: 'Mountain Climbers', muscleGroup: 'Abs', singleArm: false },
      { name: 'Dead Bug', muscleGroup: 'Abs', singleArm: false },
      
      // Cardio exercises
      { name: 'Running', muscleGroup: 'Cardio', singleArm: false },
      { name: 'Walking', muscleGroup: 'Cardio', singleArm: false },
      { name: 'Jogging', muscleGroup: 'Cardio', singleArm: false },
      { name: 'Cycling', muscleGroup: 'Cardio', singleArm: false },
      { name: 'Rowing Machine', muscleGroup: 'Cardio', singleArm: false },
      { name: 'Elliptical', muscleGroup: 'Cardio', singleArm: false },
      { name: 'Swimming', muscleGroup: 'Cardio', singleArm: false },
      { name: 'Jumping Jacks', muscleGroup: 'Cardio', singleArm: false },
      { name: 'Burpees', muscleGroup: 'Cardio', singleArm: false },
      { name: 'High Knees', muscleGroup: 'Cardio', singleArm: false },
    ]
    
    for (const exercise of defaultExercises) {
      await saveExercise(exercise)
    }
  }
}

/**
 * Get all workout templates sorted by creation date (newest first)
 * @returns {Promise<Array>} Array of template objects
 */
export async function getWorkoutTemplates() {
  const db = await initDB()
  return db.getAllFromIndex('workout_templates', 'created')
}

/**
 * Get a specific workout template by ID
 * @param {number} id - The template ID
 * @returns {Promise<Object|null>} The template object or null if not found
 */
export async function getWorkoutTemplate(id) {
  const db = await initDB()
  return db.get('workout_templates', id)
}

/**
 * Save a workout template to the database
 * @param {Object} template - The template object to save
 * @returns {Promise<number>} The template ID
 */
export async function saveWorkoutTemplate(template) {
  const db = await initDB()
  const templateData = {
    ...template,
    created: template.created || new Date(),
    updated: new Date()
  }
  
  if (template.id) {
    await db.put('workout_templates', templateData)
    return template.id
  } else {
    return db.add('workout_templates', templateData)
  }
}

/**
 * Delete a workout template from the database
 * @param {number} id - The template ID to delete
 * @returns {Promise<void>}
 */
export async function deleteWorkoutTemplate(id) {
  const db = await initDB()
  return db.delete('workout_templates', id)
}