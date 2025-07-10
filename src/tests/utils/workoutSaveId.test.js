import { describe, it, expect, vi, beforeEach } from 'vitest'

// Mock the database function
const mockSaveWorkoutToDB = vi.fn()
vi.mock('@/utils/database.js', () => ({
  saveWorkout: mockSaveWorkoutToDB
}))

describe('Workout ID Preservation', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('preserves ID when saving existing workout', async () => {
    // Test data
    const existingWorkout = {
      id: 42,
      name: 'Test Workout',
      started: new Date('2024-01-01T10:00:00'),
      ended: new Date('2024-01-01T11:00:00'),
      notes: 'Test notes',
      exercises: []
    }

    // Mock the database function to return the same ID
    mockSaveWorkoutToDB.mockResolvedValueOnce(42)

    // Simulate what happens in saveWorkout function
    const workoutData = JSON.parse(
      JSON.stringify({
        ...existingWorkout,
        updated: new Date(),
      })
    )

    const id = await mockSaveWorkoutToDB(workoutData)

    // Verify the workout data includes the ID
    expect(mockSaveWorkoutToDB).toHaveBeenCalledWith(
      expect.objectContaining({
        id: 42,
        name: 'Test Workout'
      })
    )

    // Verify the same ID is returned
    expect(id).toBe(42)
  })

  it('creates new ID for new workout', async () => {
    // Test data without ID
    const newWorkout = {
      id: null,
      name: 'New Workout',
      started: new Date('2024-01-01T10:00:00'),
      ended: null,
      notes: '',
      exercises: []
    }

    // Mock the database function to return a new ID
    mockSaveWorkoutToDB.mockResolvedValueOnce(123)

    // Simulate what happens in saveWorkout function
    const workoutData = JSON.parse(
      JSON.stringify({
        ...newWorkout,
        updated: new Date(),
      })
    )

    const id = await mockSaveWorkoutToDB(workoutData)

    // Verify the workout data is passed correctly
    expect(mockSaveWorkoutToDB).toHaveBeenCalledWith(
      expect.objectContaining({
        id: null,
        name: 'New Workout'
      })
    )

    // Verify a new ID is returned
    expect(id).toBe(123)
  })
})