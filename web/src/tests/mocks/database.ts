import { vi } from 'vitest'
import type * as db from '@/utils/database'
import type { Exercise, Workout, WorkoutSet } from '@/types/domain'

const set = (fields: Partial<WorkoutSet>): WorkoutSet => ({
  type: 'regular',
  weight: null,
  distance: null,
  reps: null,
  time: '',
  rpe: null,
  arm: '',
  notes: '',
  ...fields
})

const createMockWorkouts = (): Workout[] => [
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
        type: 'strength',
        displayType: 'reps',
        sets: [
          set({ type: 'regular', weight: 20, reps: 10, rpe: 7, arm: 'right', notes: '' }),
          set({ type: 'regular', weight: 22, reps: 8, rpe: 8, arm: 'right', notes: '' })
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
        type: 'strength',
        displayType: 'reps',
        sets: [
          set({ type: 'warmup', weight: 15, reps: 12, rpe: 5, arm: 'left', notes: 'warmup set' }),
          set({ type: 'regular', weight: 25, reps: 6, rpe: 9, arm: 'left', notes: '' })
        ]
      }
    ]
  }
]

// Mock data
const mockWorkouts: Workout[] = createMockWorkouts()

const mockExercises: Exercise[] = [
  { id: 1, name: 'Wrist Curl', muscleGroup: 'Wrist', singleArm: true, type: 'strength', displayType: 'reps' },
  { id: 2, name: 'Hook Training', muscleGroup: 'Hand', singleArm: true, type: 'strength', displayType: 'reps' },
  { id: 3, name: 'Side Pressure', muscleGroup: 'Side', singleArm: true, type: 'strength', displayType: 'reps' },
  { id: 4, name: 'Pronation Curl', muscleGroup: 'Forearm', singleArm: true, type: 'strength', displayType: 'reps' }
]

const mockSettings = new Map<string, unknown>([
  ['language', 'en'],
  ['weightUnit', 'kg'],
  ['theme', 'light']
])

// Mock functions
export const getWorkouts = vi.fn<typeof db.getWorkouts>().mockResolvedValue([...mockWorkouts])

export const getWorkout = vi.fn<typeof db.getWorkout>().mockImplementation((id) => {
  const workout = mockWorkouts.find(w => w.id === id)
  return Promise.resolve(workout)
})

export const saveWorkout = vi.fn<typeof db.saveWorkout>().mockImplementation((workout) => {
  if (workout.id) {
    const index = mockWorkouts.findIndex(w => w.id === workout.id)
    if (index !== -1) {
      mockWorkouts[index] = { ...workout }
    }
    return Promise.resolve(workout.id)
  } else {
    const newId = Math.max(...mockWorkouts.map(w => w.id ?? 0)) + 1
    const newWorkout = { ...workout, id: newId, created: new Date(), updated: new Date() }
    mockWorkouts.push(newWorkout)
    return Promise.resolve(newId)
  }
})

export const deleteWorkout = vi.fn<typeof db.deleteWorkout>().mockImplementation((id) => {
  const index = mockWorkouts.findIndex(w => w.id === id)
  if (index !== -1) {
    mockWorkouts.splice(index, 1)
  }
  return Promise.resolve()
})

export const getExercises = vi.fn<typeof db.getExercises>().mockResolvedValue([...mockExercises])

export const saveExercise = vi.fn<typeof db.saveExercise>().mockImplementation((exercise) => {
  if (exercise.id) {
    const index = mockExercises.findIndex(e => e.id === exercise.id)
    if (index !== -1) {
      mockExercises[index] = { ...exercise }
    }
    return Promise.resolve(exercise.id)
  } else {
    const newId = Math.max(...mockExercises.map(e => e.id ?? 0)) + 1
    const newExercise = { ...exercise, id: newId }
    mockExercises.push(newExercise)
    return Promise.resolve(newId)
  }
})

// Cast needed because the real getSetting is overloaded
export const getSetting = vi.fn<typeof db.getSetting>().mockImplementation(
  ((key: string, defaultValue: unknown = null) => {
    return Promise.resolve(mockSettings.get(key) || defaultValue)
  }) as typeof db.getSetting
)

export const saveSetting = vi.fn<typeof db.saveSetting>().mockImplementation((key, value) => {
  mockSettings.set(key, value)
  return Promise.resolve()
})

export const initializeDefaultExercises = vi.fn<typeof db.initializeDefaultExercises>().mockResolvedValue()

// Helper function to reset mocks
export const resetMocks = () => {
  mockWorkouts.length = 0
  mockWorkouts.push(...createMockWorkouts())

  mockSettings.clear()
  mockSettings.set('language', 'en')
  mockSettings.set('weightUnit', 'kg')
  mockSettings.set('theme', 'light')

  vi.clearAllMocks()
}
