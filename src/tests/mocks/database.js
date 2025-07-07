import { vi } from 'vitest'

// Mock data
const mockWorkouts = [
  {
    id: 1,
    name: 'Test Workout 1',
    started: new Date('2024-01-15T10:00:00'),
    ended: new Date('2024-01-15T11:30:00'),
    notes: 'First test workout',
    exercises: [
      {
        name: 'Wrist Curl',
        muscleGroup: 'Wrist',
        singleArm: true,
        sets: [
          { type: 'regular', weight: 20, reps: 10, rpe: 'RPE 7', arm: 'right', notes: '' },
          { type: 'regular', weight: 22, reps: 8, rpe: 'RPE 8', arm: 'right', notes: '' }
        ]
      }
    ]
  },
  {
    id: 2,
    name: 'Test Workout 2',
    started: new Date('2024-01-16T14:00:00'),
    ended: new Date('2024-01-16T15:45:00'),
    notes: 'Second test workout',
    exercises: [
      {
        name: 'Hook Training',
        muscleGroup: 'Hand',
        singleArm: true,
        sets: [
          { type: 'warmup', weight: 15, reps: 12, rpe: 'RPE 5', arm: 'left', notes: 'warmup set' },
          { type: 'regular', weight: 25, reps: 6, rpe: 'RPE 9', arm: 'left', notes: '' }
        ]
      }
    ]
  }
]

const mockExercises = [
  { id: 1, name: 'Wrist Curl', muscleGroup: 'Wrist', singleArm: true },
  { id: 2, name: 'Hook Training', muscleGroup: 'Hand', singleArm: true },
  { id: 3, name: 'Side Pressure', muscleGroup: 'Side', singleArm: true },
  { id: 4, name: 'Pronation Curl', muscleGroup: 'Forearm', singleArm: true }
]

const mockSettings = new Map([
  ['language', 'en'],
  ['weightUnit', 'kg'],
  ['theme', 'light']
])

// Mock functions
export const getWorkouts = vi.fn().mockResolvedValue([...mockWorkouts])

export const getWorkout = vi.fn().mockImplementation((id) => {
  const workout = mockWorkouts.find(w => w.id === id)
  return Promise.resolve(workout)
})

export const saveWorkout = vi.fn().mockImplementation((workout) => {
  if (workout.id) {
    const index = mockWorkouts.findIndex(w => w.id === workout.id)
    if (index !== -1) {
      mockWorkouts[index] = { ...workout }
    }
    return Promise.resolve(workout.id)
  } else {
    const newId = Math.max(...mockWorkouts.map(w => w.id)) + 1
    const newWorkout = { ...workout, id: newId, created: new Date(), updated: new Date() }
    mockWorkouts.push(newWorkout)
    return Promise.resolve(newId)
  }
})

export const deleteWorkout = vi.fn().mockImplementation((id) => {
  const index = mockWorkouts.findIndex(w => w.id === id)
  if (index !== -1) {
    mockWorkouts.splice(index, 1)
  }
  return Promise.resolve()
})

export const getExercises = vi.fn().mockResolvedValue([...mockExercises])

export const saveExercise = vi.fn().mockImplementation((exercise) => {
  if (exercise.id) {
    const index = mockExercises.findIndex(e => e.id === exercise.id)
    if (index !== -1) {
      mockExercises[index] = { ...exercise }
    }
    return Promise.resolve(exercise.id)
  } else {
    const newId = Math.max(...mockExercises.map(e => e.id)) + 1
    const newExercise = { ...exercise, id: newId }
    mockExercises.push(newExercise)
    return Promise.resolve(newId)
  }
})

export const getSetting = vi.fn().mockImplementation((key, defaultValue = null) => {
  return Promise.resolve(mockSettings.get(key) || defaultValue)
})

export const saveSetting = vi.fn().mockImplementation((key, value) => {
  mockSettings.set(key, value)
  return Promise.resolve()
})

export const initializeDefaultExercises = vi.fn().mockResolvedValue()

// Helper function to reset mocks
export const resetMocks = () => {
  mockWorkouts.length = 0
  mockWorkouts.push(
    {
      id: 1,
      name: 'Test Workout 1',
      started: new Date('2024-01-15T10:00:00'),
      ended: new Date('2024-01-15T11:30:00'),
      notes: 'First test workout',
      exercises: [
        {
          name: 'Wrist Curl',
          muscleGroup: 'Wrist',
          singleArm: true,
          sets: [
            { type: 'regular', weight: 20, reps: 10, rpe: 'RPE 7', arm: 'right', notes: '' },
            { type: 'regular', weight: 22, reps: 8, rpe: 'RPE 8', arm: 'right', notes: '' }
          ]
        }
      ]
    },
    {
      id: 2,
      name: 'Test Workout 2',
      started: new Date('2024-01-16T14:00:00'),
      ended: new Date('2024-01-16T15:45:00'),
      notes: 'Second test workout',
      exercises: [
        {
          name: 'Hook Training',
          muscleGroup: 'Hand',
          singleArm: true,
          sets: [
            { type: 'warmup', weight: 15, reps: 12, rpe: 'RPE 5', arm: 'left', notes: 'warmup set' },
            { type: 'regular', weight: 25, reps: 6, rpe: 'RPE 9', arm: 'left', notes: '' }
          ]
        }
      ]
    }
  )
  
  mockSettings.clear()
  mockSettings.set('language', 'en')
  mockSettings.set('weightUnit', 'kg')
  mockSettings.set('theme', 'light')
  
  vi.clearAllMocks()
}