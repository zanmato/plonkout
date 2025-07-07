import { describe, it, expect } from 'vitest'

describe('WorkoutLog Debugging - Potential Issues', () => {
  it('identifies potential issues with date grouping', () => {
    // Test what happens when dates are very close together within a month
    const workouts = [
      {
        id: 1,
        name: 'Workout 1',
        started: '2024-01-31T23:59:59Z', // End of January
        exercises: []
      },
      {
        id: 2,
        name: 'Workout 2',
        started: '2024-01-01T00:00:01Z', // Beginning of January
        exercises: []
      },
      {
        id: 3,
        name: 'Workout 3', 
        started: '2024-01-15T12:00:00Z', // Middle of January
        exercises: []
      }
    ]

    // Apply the same sorting as WorkoutLog.vue
    const sorted = workouts.sort((a, b) => new Date(b.started) - new Date(a.started))

    // The sorting should be: Jan 31, Jan 15, Jan 1
    expect(sorted[0].id).toBe(1) // Jan 31 (newest)
    expect(sorted[1].id).toBe(3) // Jan 15 
    expect(sorted[2].id).toBe(2) // Jan 1 (oldest)

    console.log('Sorted workout order:')
    sorted.forEach((w, i) => {
      console.log(`${i}: ID ${w.id} - ${w.started}`)
    })
  })

  it('tests the exact grouping logic that might be causing issues', () => {
    // Create a scenario that might expose the sorting problem
    const workouts = [
      {
        id: 1,
        name: 'Should be last in January',
        started: '2024-01-05T08:00:00Z',
        exercises: []
      },
      {
        id: 2,
        name: 'Should be first in January', 
        started: '2024-01-25T16:00:00Z',
        exercises: []
      },
      {
        id: 3,
        name: 'Should be middle in January',
        started: '2024-01-15T12:00:00Z',
        exercises: []
      }
    ]

    // Step 1: Sort workouts (loadWorkouts logic)
    const sortedWorkouts = workouts.sort((a, b) => new Date(b.started) - new Date(a.started))
    
    console.log('After loadWorkouts sorting:')
    sortedWorkouts.forEach((w, i) => {
      console.log(`${i}: ID ${w.id} - ${new Date(w.started).toISOString()}`)
    })

    // Step 2: Group by month (groupedWorkouts logic)
    const groups = {}
    
    sortedWorkouts.forEach((workout) => {
      const date = new Date(workout.started)
      // Simulate the i18n date formatting
      const monthYear = `${date.toLocaleString('en', { month: 'long' })} ${date.getFullYear()}`
      
      if (!groups[monthYear]) {
        groups[monthYear] = {
          monthYear,
          workouts: [],
          sortDate: date,
        }
      }
      
      groups[monthYear].workouts.push(workout)
    })

    // Step 3: Sort groups and workouts within groups
    const groupedWorkouts = Object.values(groups)
      .sort((a, b) => b.sortDate - a.sortDate)
      .map((group) => ({
        ...group,
        workouts: group.workouts.sort(
          (a, b) => new Date(b.started) - new Date(a.started)
        ),
      }))

    expect(groupedWorkouts).toHaveLength(1) // Only January
    const janGroup = groupedWorkouts[0]
    
    console.log('Final grouped workouts for January:')
    janGroup.workouts.forEach((w, i) => {
      console.log(`${i}: ID ${w.id} - ${new Date(w.started).toISOString()}`)
    })

    // The order within January should be: ID 2 (25th), ID 3 (15th), ID 1 (5th)
    expect(janGroup.workouts[0].id).toBe(2) // Jan 25 should be first
    expect(janGroup.workouts[1].id).toBe(3) // Jan 15 should be second  
    expect(janGroup.workouts[2].id).toBe(1) // Jan 5 should be last
  })

  it('reveals the actual problem: sortDate in groups', () => {
    // This test reveals a potential issue with the sortDate assignment
    const workouts = [
      {
        id: 1,
        started: '2024-01-05T08:00:00Z', // Early January
        exercises: []
      },
      {
        id: 2, 
        started: '2024-01-25T16:00:00Z', // Late January
        exercises: []
      }
    ]

    const groups = {}
    
    // Process workouts in the order they come from the database
    // This simulates the forEach in groupedWorkouts
    workouts.forEach((workout) => {
      const date = new Date(workout.started)
      const monthYear = 'January 2024'
      
      if (!groups[monthYear]) {
        groups[monthYear] = {
          monthYear,
          workouts: [],
          sortDate: date, // This uses the FIRST workout's date as sortDate
        }
      }
      
      groups[monthYear].workouts.push(workout)
    })

    const janGroup = groups['January 2024']
    
    // The sortDate is set to the first workout processed (Jan 5)
    // But this might not be the newest workout in the group
    console.log('Group sortDate:', janGroup.sortDate.toISOString())
    console.log('Should be latest date in group:', new Date('2024-01-25T16:00:00Z').toISOString())
    
    // This could cause month groups to be sorted incorrectly
    // if the first workout processed isn't the newest in that month
    expect(janGroup.sortDate).toEqual(new Date('2024-01-05T08:00:00Z'))
    
    // The fix would be to update sortDate to the newest date in the group
    // or calculate it after all workouts are added
  })

  it('proposes a fix for the grouping sortDate issue', () => {
    const workouts = [
      { id: 1, started: '2024-01-05T08:00:00Z', exercises: [] },
      { id: 2, started: '2024-01-25T16:00:00Z', exercises: [] },
      { id: 3, started: '2024-02-10T12:00:00Z', exercises: [] },
      { id: 4, started: '2024-02-01T09:00:00Z', exercises: [] }
    ]

    // Current implementation (potentially buggy)
    const groupsCurrent = {}
    workouts.forEach((workout) => {
      const date = new Date(workout.started)
      const monthYear = `${date.toLocaleString('en', { month: 'long' })} ${date.getFullYear()}`
      
      if (!groupsCurrent[monthYear]) {
        groupsCurrent[monthYear] = {
          monthYear,
          workouts: [],
          sortDate: date, // Uses first workout's date
        }
      }
      
      groupsCurrent[monthYear].workouts.push(workout)
    })

    // Fixed implementation
    const groupsFixed = {}
    workouts.forEach((workout) => {
      const date = new Date(workout.started)
      const monthYear = `${date.toLocaleString('en', { month: 'long' })} ${date.getFullYear()}`
      
      if (!groupsFixed[monthYear]) {
        groupsFixed[monthYear] = {
          monthYear,
          workouts: [],
          sortDate: date,
        }
      } else {
        // Update sortDate to the newest date in this group
        if (date > groupsFixed[monthYear].sortDate) {
          groupsFixed[monthYear].sortDate = date
        }
      }
      
      groupsFixed[monthYear].workouts.push(workout)
    })

    // Check current implementation
    const currentFebGroup = groupsCurrent['February 2024']
    expect(currentFebGroup.sortDate).toEqual(new Date('2024-02-10T12:00:00Z')) // First Feb workout processed

    const currentJanGroup = groupsCurrent['January 2024'] 
    expect(currentJanGroup.sortDate).toEqual(new Date('2024-01-05T08:00:00Z')) // First Jan workout processed

    // Check fixed implementation  
    const fixedFebGroup = groupsFixed['February 2024']
    expect(fixedFebGroup.sortDate).toEqual(new Date('2024-02-10T12:00:00Z')) // Latest Feb workout

    const fixedJanGroup = groupsFixed['January 2024']
    expect(fixedJanGroup.sortDate).toEqual(new Date('2024-01-25T16:00:00Z')) // Latest Jan workout

    console.log('Current Jan sortDate:', currentJanGroup.sortDate.toISOString())
    console.log('Fixed Jan sortDate:', fixedJanGroup.sortDate.toISOString())
  })
})