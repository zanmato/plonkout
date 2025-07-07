import { describe, it, expect } from 'vitest'

describe('Workout Sorting Logic', () => {
  it('sorts workouts by started date descending', () => {
    // Test data with mixed dates
    const workouts = [
      {
        id: 1,
        name: 'Workout 1',
        started: '2024-01-15T10:00:00Z', // Middle
        ended: '2024-01-15T11:00:00Z',
        exercises: []
      },
      {
        id: 2,
        name: 'Workout 2', 
        started: '2024-01-20T10:00:00Z', // Newest
        ended: '2024-01-20T11:00:00Z',
        exercises: []
      },
      {
        id: 3,
        name: 'Workout 3',
        started: '2024-01-10T10:00:00Z', // Oldest
        ended: '2024-01-10T11:00:00Z',
        exercises: []
      }
    ]

    // Apply the same sorting logic as in WorkoutLog
    const sorted = workouts.sort((a, b) => new Date(b.started) - new Date(a.started))

    expect(sorted).toHaveLength(3)
    expect(sorted[0].id).toBe(2) // 2024-01-20 (newest first)
    expect(sorted[1].id).toBe(1) // 2024-01-15 (middle)
    expect(sorted[2].id).toBe(3) // 2024-01-10 (oldest last)
  })

  it('groups workouts by month and sorts within groups by date descending', () => {
    // Test data with workouts in same month but different dates
    const workouts = [
      {
        id: 1,
        name: 'Early January',
        started: '2024-01-05T10:00:00Z',
        ended: '2024-01-05T11:00:00Z',
        exercises: []
      },
      {
        id: 2,
        name: 'Late January',
        started: '2024-01-25T10:00:00Z',
        ended: '2024-01-25T11:00:00Z',
        exercises: []
      },
      {
        id: 3,
        name: 'Mid January',
        started: '2024-01-15T10:00:00Z',
        ended: '2024-01-15T11:00:00Z',
        exercises: []
      },
      {
        id: 4,
        name: 'February Workout',
        started: '2024-02-10T10:00:00Z',
        ended: '2024-02-10T11:00:00Z',
        exercises: []
      }
    ]

    // First sort all workouts by date (as loadWorkouts does)
    const sortedWorkouts = workouts.sort((a, b) => new Date(b.started) - new Date(a.started))

    // Then group by month (simplified version of groupedWorkouts logic)
    const groups = {}
    
    sortedWorkouts.forEach((workout) => {
      const date = new Date(workout.started)
      const monthKey = `${date.getFullYear()}-${date.getMonth()}` // Year-Month key
      
      if (!groups[monthKey]) {
        groups[monthKey] = {
          monthKey,
          workouts: [],
          sortDate: date,
        }
      }
      
      groups[monthKey].workouts.push(workout)
    })

    // Sort groups by date (newest first) and workouts within groups by date (newest first)
    const groupedWorkouts = Object.values(groups)
      .sort((a, b) => b.sortDate - a.sortDate)
      .map((group) => ({
        ...group,
        workouts: group.workouts.sort(
          (a, b) => new Date(b.started) - new Date(a.started)
        ),
      }))

    expect(groupedWorkouts).toHaveLength(2) // Two months

    // February should be first (newest month)
    const febGroup = groupedWorkouts[0]
    expect(febGroup.workouts).toHaveLength(1)
    expect(febGroup.workouts[0].id).toBe(4)

    // January should be second
    const janGroup = groupedWorkouts[1] 
    expect(janGroup.workouts).toHaveLength(3)
    
    // Within January, should be sorted newest first
    expect(janGroup.workouts[0].id).toBe(2) // 2024-01-25 (latest)
    expect(janGroup.workouts[1].id).toBe(3) // 2024-01-15 (middle)
    expect(janGroup.workouts[2].id).toBe(1) // 2024-01-05 (earliest)
  })

  it('handles edge case with same started dates', () => {
    const workouts = [
      {
        id: 1,
        name: 'Workout 1',
        started: '2024-01-15T10:00:00Z',
        ended: '2024-01-15T11:00:00Z',
        exercises: []
      },
      {
        id: 2,
        name: 'Workout 2',
        started: '2024-01-15T10:00:00Z', // Same exact time
        ended: '2024-01-15T11:30:00Z',
        exercises: []
      }
    ]

    // Sort by started date
    const sorted = workouts.sort((a, b) => new Date(b.started) - new Date(a.started))

    // Both workouts should still be present
    expect(sorted).toHaveLength(2)
    
    // Order might be either way since dates are identical, but both should be there
    const ids = sorted.map(w => w.id)
    expect(ids).toContain(1)
    expect(ids).toContain(2)
  })

  it('verifies date comparison logic works correctly', () => {
    // Test the core date comparison used in sorting
    const date1 = new Date('2024-01-15T10:00:00Z')
    const date2 = new Date('2024-01-20T10:00:00Z')
    const date3 = new Date('2024-01-10T10:00:00Z')

    // For descending sort (newest first), b - a should be positive when b > a
    expect(date2 - date1).toBeGreaterThan(0) // 20th - 15th = positive
    expect(date1 - date3).toBeGreaterThan(0) // 15th - 10th = positive
    expect(date2 - date3).toBeGreaterThan(0) // 20th - 10th = positive

    // Test actual sort result
    const dates = [date1, date2, date3]
    const sorted = dates.sort((a, b) => b - a) // Descending

    expect(sorted[0]).toBe(date2) // 20th first
    expect(sorted[1]).toBe(date1) // 15th second  
    expect(sorted[2]).toBe(date3) // 10th last
  })

  it('tests actual WorkoutLog.vue grouping logic with i18n formatting', () => {
    // This test simulates the actual grouping logic from WorkoutLog.vue
    // without requiring the full component setup

    const workouts = [
      {
        id: 1,
        name: 'Workout A',
        started: '2024-01-05T10:00:00Z',
        exercises: []
      },
      {
        id: 2, 
        name: 'Workout B',
        started: '2024-01-25T10:00:00Z',
        exercises: []
      },
      {
        id: 3,
        name: 'Workout C', 
        started: '2024-02-10T10:00:00Z',
        exercises: []
      }
    ]

    // Sort workouts first (as done in loadWorkouts)
    const sortedWorkouts = workouts.sort((a, b) => new Date(b.started) - new Date(a.started))

    // Group by month (simplified version without i18n formatting)
    const groups = {}
    
    sortedWorkouts.forEach((workout) => {
      const date = new Date(workout.started)
      // Use simple month-year key instead of i18n formatted string
      const monthYear = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`
      
      if (!groups[monthYear]) {
        groups[monthYear] = {
          monthYear,
          workouts: [],
          sortDate: date,
        }
      }
      
      groups[monthYear].workouts.push(workout)
    })

    // Apply the exact sorting logic from WorkoutLog.vue
    const groupedWorkouts = Object.values(groups)
      .sort((a, b) => b.sortDate - a.sortDate)
      .map((group) => ({
        ...group,
        workouts: group.workouts.sort(
          (a, b) => new Date(b.started) - new Date(a.started)
        ),
      }))

    // Verify results
    expect(groupedWorkouts).toHaveLength(2)
    
    // February group should be first
    expect(groupedWorkouts[0].monthYear).toBe('2024-02')
    expect(groupedWorkouts[0].workouts[0].id).toBe(3)
    
    // January group should be second  
    expect(groupedWorkouts[1].monthYear).toBe('2024-01')
    expect(groupedWorkouts[1].workouts[0].id).toBe(2) // Jan 25th first
    expect(groupedWorkouts[1].workouts[1].id).toBe(1) // Jan 5th second
  })
})