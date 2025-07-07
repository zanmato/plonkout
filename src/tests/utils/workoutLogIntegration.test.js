import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ref, computed, nextTick } from 'vue'

// Simulate the actual logic from WorkoutLog.vue
describe('WorkoutLog Integration - Sorting Behavior', () => {
  let workouts
  let mockDateFormatter

  beforeEach(() => {
    workouts = ref([])
    
    // Mock the date formatter (d function from i18n)
    mockDateFormatter = vi.fn((date, format) => {
      if (format === 'month') {
        const d = new Date(date)
        const month = d.toLocaleString('en', { month: 'long' })
        const year = d.getFullYear()
        return `${month} ${year}`
      }
      return date.toString()
    })
  })

  it('reproduces the exact WorkoutLog grouping logic', async () => {
    // Set up test data that could expose sorting issues
    const testWorkouts = [
      {
        id: 1,
        name: 'Early January',
        started: '2024-01-05T09:00:00Z',
        ended: '2024-01-05T10:00:00Z',
        exercises: []
      },
      {
        id: 2,
        name: 'Late January',
        started: '2024-01-25T14:00:00Z', 
        ended: '2024-01-25T15:00:00Z',
        exercises: []
      },
      {
        id: 3,
        name: 'Mid January',
        started: '2024-01-15T12:00:00Z',
        ended: '2024-01-15T13:00:00Z', 
        exercises: []
      },
      {
        id: 4,
        name: 'February',
        started: '2024-02-10T10:00:00Z',
        ended: '2024-02-10T11:00:00Z',
        exercises: []
      }
    ]

    // Simulate loadWorkouts function
    const loadWorkouts = async () => {
      // Simulate API call that might return data in different order
      const data = [...testWorkouts] // Copy to avoid mutation
      
      // Apply the exact sorting logic from WorkoutLog.vue
      workouts.value = data.sort((a, b) => new Date(b.started) - new Date(a.started))
    }

    // Simulate the groupedWorkouts computed property
    const groupedWorkouts = computed(() => {
      const groups = {}

      workouts.value.forEach((workout) => {
        const date = new Date(workout.started)
        const monthYear = mockDateFormatter(date, "month")

        if (!groups[monthYear]) {
          groups[monthYear] = {
            monthYear,
            workouts: [],
            sortDate: date,
          }
        }

        groups[monthYear].workouts.push(workout)
      })

      // Sort groups by date (newest first) and workouts within groups by date (newest first)
      return Object.values(groups)
        .sort((a, b) => b.sortDate - a.sortDate)
        .map((group) => ({
          ...group,
          workouts: group.workouts.sort(
            (a, b) => new Date(b.started) - new Date(a.started)
          ),
        }))
    })

    // Simulate the flattenedItems computed property
    const flattenedItems = computed(() => {
      const items = []
      
      groupedWorkouts.value.forEach((group) => {
        // Add month header
        items.push({
          type: 'header',
          monthYear: group.monthYear,
          workoutCount: group.workouts.length
        })
        
        // Add workouts for this month
        group.workouts.forEach((workout) => {
          items.push({
            type: 'workout',
            ...workout
          })
        })
      })
      
      return items
    })

    // Load the workouts
    await loadWorkouts()
    await nextTick() // Wait for reactivity to update

    // Test 1: Check that workouts are sorted correctly in the base array
    expect(workouts.value).toHaveLength(4)
    expect(workouts.value[0].id).toBe(4) // Feb 10 (newest)
    expect(workouts.value[1].id).toBe(2) // Jan 25 
    expect(workouts.value[2].id).toBe(3) // Jan 15
    expect(workouts.value[3].id).toBe(1) // Jan 5 (oldest)

    // Test 2: Check grouped workouts
    const groups = groupedWorkouts.value
    expect(groups).toHaveLength(2) // February and January

    // February should be first (newest month)
    expect(groups[0].monthYear).toBe('February 2024')
    expect(groups[0].workouts).toHaveLength(1)
    expect(groups[0].workouts[0].id).toBe(4)

    // January should be second  
    expect(groups[1].monthYear).toBe('January 2024')
    expect(groups[1].workouts).toHaveLength(3)
    
    // Within January, workouts should be sorted newest first
    expect(groups[1].workouts[0].id).toBe(2) // Jan 25 (newest in January)
    expect(groups[1].workouts[1].id).toBe(3) // Jan 15
    expect(groups[1].workouts[2].id).toBe(1) // Jan 5 (oldest in January)

    // Test 3: Check flattened items for virtual scroller
    const flattened = flattenedItems.value
    expect(flattened).toHaveLength(6) // 2 headers + 4 workouts

    // Should be: Feb header, Feb workout, Jan header, Jan workouts (25th, 15th, 5th)
    expect(flattened[0].type).toBe('header')
    expect(flattened[0].monthYear).toBe('February 2024')
    
    expect(flattened[1].type).toBe('workout')
    expect(flattened[1].id).toBe(4) // Feb workout
    
    expect(flattened[2].type).toBe('header') 
    expect(flattened[2].monthYear).toBe('January 2024')
    
    expect(flattened[3].type).toBe('workout')
    expect(flattened[3].id).toBe(2) // Jan 25 (newest in Jan)
    
    expect(flattened[4].type).toBe('workout')
    expect(flattened[4].id).toBe(3) // Jan 15
    
    expect(flattened[5].type).toBe('workout')
    expect(flattened[5].id).toBe(1) // Jan 5 (oldest in Jan)
  })

  it('tests potential reactivity issues with sorting', async () => {
    // This test simulates adding workouts one by one to see if sorting remains correct
    
    const groupedWorkouts = computed(() => {
      const groups = {}
      
      workouts.value.forEach((workout) => {
        const date = new Date(workout.started)
        const monthYear = mockDateFormatter(date, "month")
        
        if (!groups[monthYear]) {
          groups[monthYear] = {
            monthYear,
            workouts: [],
            sortDate: date,
          }
        }
        
        groups[monthYear].workouts.push(workout)
      })
      
      return Object.values(groups)
        .sort((a, b) => b.sortDate - a.sortDate)
        .map((group) => ({
          ...group,
          workouts: group.workouts.sort(
            (a, b) => new Date(b.started) - new Date(a.started)
          ),
        }))
    })

    // Start with empty and add workouts
    expect(workouts.value).toHaveLength(0)
    
    // Add first workout
    workouts.value = [{
      id: 1,
      name: 'First',
      started: '2024-01-15T10:00:00Z',
      exercises: []
    }]
    await nextTick()
    
    let groups = groupedWorkouts.value
    expect(groups).toHaveLength(1)
    expect(groups[0].workouts[0].id).toBe(1)
    
    // Add an older workout
    workouts.value = [
      ...workouts.value,
      {
        id: 2,
        name: 'Older', 
        started: '2024-01-10T10:00:00Z',
        exercises: []
      }
    ].sort((a, b) => new Date(b.started) - new Date(a.started))
    await nextTick()
    
    groups = groupedWorkouts.value
    expect(groups).toHaveLength(1)
    expect(groups[0].workouts).toHaveLength(2)
    expect(groups[0].workouts[0].id).toBe(1) // 15th should be first
    expect(groups[0].workouts[1].id).toBe(2) // 10th should be second
    
    // Add a newer workout
    workouts.value = [
      ...workouts.value,
      {
        id: 3,
        name: 'Newer',
        started: '2024-01-20T10:00:00Z', 
        exercises: []
      }
    ].sort((a, b) => new Date(b.started) - new Date(a.started))
    await nextTick()
    
    groups = groupedWorkouts.value
    expect(groups).toHaveLength(1)
    expect(groups[0].workouts).toHaveLength(3)
    expect(groups[0].workouts[0].id).toBe(3) // 20th should be first now
    expect(groups[0].workouts[1].id).toBe(1) // 15th should be second
    expect(groups[0].workouts[2].id).toBe(2) // 10th should be last
  })

  it('verifies sorting works with various date formats and timezones', () => {
    // Test different date formats that might come from the database
    const testWorkouts = [
      {
        id: 1,
        started: '2024-01-15T10:00:00.000Z', // With milliseconds
        exercises: []
      },
      {
        id: 2, 
        started: '2024-01-20T10:00:00Z', // Without milliseconds
        exercises: []
      },
      {
        id: 3,
        started: new Date('2024-01-10T10:00:00Z'), // Date object
        exercises: []
      }
    ]

    // Apply sorting
    const sorted = testWorkouts.sort((a, b) => new Date(b.started) - new Date(a.started))

    expect(sorted[0].id).toBe(2) // Jan 20
    expect(sorted[1].id).toBe(1) // Jan 15  
    expect(sorted[2].id).toBe(3) // Jan 10
  })
})