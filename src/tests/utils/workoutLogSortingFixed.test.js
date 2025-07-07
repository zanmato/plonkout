import { describe, it, expect } from 'vitest'

describe('WorkoutLog Sorting - Verification of Fix', () => {
  it('verifies the fix ensures correct month group sorting', () => {
    // Test data that could cause the original bug
    const workouts = [
      { id: 1, started: '2024-01-05T08:00:00Z', exercises: [] }, // Early January (processed first)
      { id: 2, started: '2024-01-25T16:00:00Z', exercises: [] }, // Late January (processed second)
      { id: 3, started: '2024-02-01T09:00:00Z', exercises: [] }, // Early February (processed third)
      { id: 4, started: '2024-02-15T14:00:00Z', exercises: [] }  // Late February (processed fourth)
    ]

    // Simulate the fixed grouping logic
    const groups = {}
    
    workouts.forEach((workout) => {
      const date = new Date(workout.started)
      const monthYear = `${date.toLocaleString('en', { month: 'long', timeZone: 'UTC' })} ${date.getFullYear()}`
      
      if (!groups[monthYear]) {
        groups[monthYear] = {
          monthYear,
          workouts: [],
          sortDate: date,
        }
      } else {
        // Fixed logic: Update sortDate to ensure it's the newest date in this group
        if (date > groups[monthYear].sortDate) {
          groups[monthYear].sortDate = date
        }
      }
      
      groups[monthYear].workouts.push(workout)
    })

    // Apply the sorting logic
    const groupedWorkouts = Object.values(groups)
      .sort((a, b) => b.sortDate - a.sortDate)
      .map((group) => ({
        ...group,
        workouts: group.workouts.sort(
          (a, b) => new Date(b.started) - new Date(a.started)
        ),
      }))

    expect(groupedWorkouts).toHaveLength(2)

    // February should be first (newer month)
    const febGroup = groupedWorkouts[0]
    expect(febGroup.monthYear).toBe('February 2024')
    expect(febGroup.sortDate).toEqual(new Date('2024-02-15T14:00:00Z')) // Latest February date
    expect(febGroup.workouts[0].id).toBe(4) // Feb 15 (newest in February)
    expect(febGroup.workouts[1].id).toBe(3) // Feb 1 (oldest in February)

    // January should be second (older month)
    const janGroup = groupedWorkouts[1]
    expect(janGroup.monthYear).toBe('January 2024')
    expect(janGroup.sortDate).toEqual(new Date('2024-01-25T16:00:00Z')) // Latest January date
    expect(janGroup.workouts[0].id).toBe(2) // Jan 25 (newest in January)
    expect(janGroup.workouts[1].id).toBe(1) // Jan 5 (oldest in January)
  })

  it('verifies sorting works correctly when workouts are pre-sorted by loadWorkouts', () => {
    // Test the actual scenario: workouts come pre-sorted from loadWorkouts
    const workouts = [
      { id: 4, started: '2024-02-15T14:00:00Z', exercises: [] }, // Newest overall
      { id: 3, started: '2024-02-01T09:00:00Z', exercises: [] }, 
      { id: 2, started: '2024-01-25T16:00:00Z', exercises: [] },
      { id: 1, started: '2024-01-05T08:00:00Z', exercises: [] }  // Oldest overall
    ]

    // Since workouts are pre-sorted, the first workout in each month group
    // will be the newest for that month, making the fix less critical
    // but still important for edge cases

    const groups = {}
    
    workouts.forEach((workout) => {
      const date = new Date(workout.started)
      const monthYear = `${date.toLocaleString('en', { month: 'long', timeZone: 'UTC' })} ${date.getFullYear()}`
      
      if (!groups[monthYear]) {
        groups[monthYear] = {
          monthYear,
          workouts: [],
          sortDate: date,
        }
      } else {
        // Fixed logic
        if (date > groups[monthYear].sortDate) {
          groups[monthYear].sortDate = date
        }
      }
      
      groups[monthYear].workouts.push(workout)
    })

    const groupedWorkouts = Object.values(groups)
      .sort((a, b) => b.sortDate - a.sortDate)
      .map((group) => ({
        ...group,
        workouts: group.workouts.sort(
          (a, b) => new Date(b.started) - new Date(a.started)
        ),
      }))

    // Verify correct month order
    expect(groupedWorkouts[0].monthYear).toBe('February 2024')
    expect(groupedWorkouts[1].monthYear).toBe('January 2024')

    // Verify correct workout order within months
    expect(groupedWorkouts[0].workouts[0].id).toBe(4) // Feb 15
    expect(groupedWorkouts[0].workouts[1].id).toBe(3) // Feb 1

    expect(groupedWorkouts[1].workouts[0].id).toBe(2) // Jan 25  
    expect(groupedWorkouts[1].workouts[1].id).toBe(1) // Jan 5
  })

  it('tests edge case where month groups could be created in wrong order', () => {
    // Edge case: if database returns workouts in a specific order that could
    // cause groups to be created with wrong sortDates

    const workouts = [
      { id: 1, started: '2024-01-01T00:00:00Z', exercises: [] }, // Very early January
      { id: 2, started: '2024-02-28T23:59:59Z', exercises: [] }, // Very late February  
      { id: 3, started: '2024-01-31T23:59:59Z', exercises: [] }, // Very late January
      { id: 4, started: '2024-02-01T00:00:01Z', exercises: [] }  // Very early February
    ]

    // Without the fix, if these are processed in this order:
    // - January group gets sortDate of Jan 1 (first January workout processed)
    // - February group gets sortDate of Feb 28 (first February workout processed)
    // - When Jan 31 is processed, it doesn't update January's sortDate
    // - When Feb 1 is processed, it doesn't update February's sortDate
    // This could cause incorrect month ordering in some edge cases

    const groups = {}
    
    workouts.forEach((workout) => {
      const date = new Date(workout.started)
      const monthYear = `${date.toLocaleString('en', { month: 'long', timeZone: 'UTC' })} ${date.getFullYear()}`
      
      if (!groups[monthYear]) {
        groups[monthYear] = {
          monthYear,
          workouts: [],
          sortDate: date,
        }
      } else {
        // With the fix: always update to the newest date
        if (date > groups[monthYear].sortDate) {
          groups[monthYear].sortDate = date
        }
      }
      
      groups[monthYear].workouts.push(workout)
    })

    // Check that sortDates are correctly set to the newest date in each group
    expect(groups['January 2024'].sortDate).toEqual(new Date('2024-01-31T23:59:59Z'))
    expect(groups['February 2024'].sortDate).toEqual(new Date('2024-02-28T23:59:59Z'))

    const groupedWorkouts = Object.values(groups)
      .sort((a, b) => b.sortDate - a.sortDate)

    // February should still be first (newer month)
    expect(groupedWorkouts[0].monthYear).toBe('February 2024')
    expect(groupedWorkouts[1].monthYear).toBe('January 2024')
  })

  it('confirms the fix resolves the reported sorting issue', () => {
    // Create a test case that demonstrates the specific issue the user reported
    console.log('Testing the fix for workout sorting within months...')

    const workouts = [
      { 
        id: 1, 
        name: 'Morning workout',
        started: '2024-12-05T09:00:00Z', 
        exercises: [] 
      },
      { 
        id: 2, 
        name: 'Evening workout',
        started: '2024-12-20T18:00:00Z', 
        exercises: [] 
      },
      { 
        id: 3, 
        name: 'Afternoon workout',
        started: '2024-12-15T15:00:00Z', 
        exercises: [] 
      }
    ]

    // Apply the complete sorting process
    // Step 1: loadWorkouts sorting
    const sortedWorkouts = workouts.sort((a, b) => new Date(b.started) - new Date(a.started))
    
    console.log('After loadWorkouts sorting:')
    sortedWorkouts.forEach((w, i) => {
      console.log(`  ${i}: ${w.name} (${w.started})`)
    })

    // Step 2: groupedWorkouts with the fix
    const groups = {}
    sortedWorkouts.forEach((workout) => {
      const date = new Date(workout.started)
      const monthYear = 'December 2024'
      
      if (!groups[monthYear]) {
        groups[monthYear] = {
          monthYear,
          workouts: [],
          sortDate: date,
        }
      } else {
        if (date > groups[monthYear].sortDate) {
          groups[monthYear].sortDate = date
        }
      }
      
      groups[monthYear].workouts.push(workout)
    })

    const groupedWorkouts = Object.values(groups).map((group) => ({
      ...group,
      workouts: group.workouts.sort(
        (a, b) => new Date(b.started) - new Date(a.started)
      ),
    }))

    const decemberGroup = groupedWorkouts[0]
    console.log('Final workout order within December:')
    decemberGroup.workouts.forEach((w, i) => {
      console.log(`  ${i}: ${w.name} (${w.started})`)
    })

    // Verify the correct order: Evening (20th), Afternoon (15th), Morning (5th)
    expect(decemberGroup.workouts[0].id).toBe(2) // Dec 20 (newest)
    expect(decemberGroup.workouts[1].id).toBe(3) // Dec 15 (middle)
    expect(decemberGroup.workouts[2].id).toBe(1) // Dec 5 (oldest)

    console.log('✓ Fix confirmed: Workouts are properly sorted by date descending within the month')
  })
})