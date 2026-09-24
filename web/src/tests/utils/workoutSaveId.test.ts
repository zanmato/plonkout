import { describe, it, expect } from 'vitest'
import { saveWorkout } from '@/api/data'
import type { Workout } from '@/types/domain'
import { backend } from '../mocks/backend'

describe('Workout ID Preservation', () => {
  it('preserves ID when saving existing workout', async () => {
    const [existing] = backend.seed({
      workouts: [{
        name: 'Test Workout',
        started: new Date('2024-01-01T10:00:00'),
        ended: new Date('2024-01-01T11:00:00'),
        notes: 'Test notes',
        exercises: []
      }]
    }).workouts

    // The editor round trips the loaded workout through JSON before saving
    const workoutData: Workout = JSON.parse(JSON.stringify({ ...existing, notes: 'Changed' }))

    const saved = await saveWorkout(workoutData)

    expect(saved.id).toBe(existing!.id)
    expect(backend.workouts).toHaveLength(1)
    expect(backend.workout(existing!.id)).toMatchObject({ name: 'Test Workout', notes: 'Changed' })
  })

  it('creates new ID for new workout', async () => {
    const newWorkout: Workout = {
      name: 'New Workout',
      started: new Date('2024-01-01T10:00:00'),
      ended: null,
      notes: '',
      exercises: []
    }

    const workoutData: Workout = JSON.parse(JSON.stringify(newWorkout))

    const saved = await saveWorkout(workoutData)

    expect(saved.id).toMatch(/^[0-9a-f-]{36}$/)
    expect(backend.workouts).toEqual([expect.objectContaining({ id: saved.id, name: 'New Workout' })])
  })
})
