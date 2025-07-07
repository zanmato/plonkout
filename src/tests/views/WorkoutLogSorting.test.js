import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import WorkoutLog from '@/views/WorkoutLog.vue'

// Mock the database module
vi.mock('@/utils/database.js', () => ({
  getWorkouts: vi.fn(),
  initializeDefaultExercises: vi.fn().mockResolvedValue()
}))

// Mock vue-router
vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn()
  })
}))

// Mock virtual scroller components
vi.mock('vue3-virtual-scroller', () => ({
  DynamicScroller: {
    name: 'DynamicScroller',
    template: '<div><slot v-for="(item, index) in items" :key="index" :item="item" :index="index" :active="true" /></div>',
    props: ['items', 'minItemSize']
  },
  DynamicScrollerItem: {
    name: 'DynamicScrollerItem',
    template: '<div><slot /></div>',
    props: ['item', 'active', 'sizeDepedencies', 'dataIndex']
  }
}))

// Mock NeoHeader and NeoButton components
vi.mock('@/components/NeoHeader.vue', () => ({
  default: {
    name: 'NeoHeader',
    template: '<div><slot name="right" /></div>',
    props: ['title']
  }
}))

vi.mock('@/components/NeoButton.vue', () => ({
  default: {
    name: 'NeoButton',
    template: '<button><slot name="icon" /><slot /></button>',
    props: ['variant', 'size', 'class']
  }
}))

describe('WorkoutLog.vue - Date Sorting', () => {
  let i18n
  let mockGetWorkouts

  beforeEach(async () => {
    // Setup i18n
    i18n = createI18n({
      locale: 'en',
      messages: {
        en: {
          workout: {
            title: 'Workouts',
            loading: 'Loading...',
            loadError: 'Error loading workouts'
          },
          workouts: {
            noWorkouts: 'No workouts found'
          },
          common: {
            duration: {
              hours: 'h',
              minutes: 'm'
            }
          }
        }
      }
    })

    // Setup database mock
    const { getWorkouts } = await import('@/utils/database.js')
    mockGetWorkouts = getWorkouts
  })

  it('sorts workouts by started date descending within loadWorkouts function', async () => {
    // Create test data with mixed dates
    const testWorkouts = [
      {
        id: 1,
        name: 'Workout 1',
        started: '2024-01-15T10:00:00Z', // Older
        ended: '2024-01-15T11:00:00Z',
        exercises: []
      },
      {
        id: 2,
        name: 'Workout 2', 
        started: '2024-01-20T10:00:00Z', // Newer
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

    mockGetWorkouts.mockResolvedValue(testWorkouts)

    const wrapper = mount(WorkoutLog, {
      global: {
        plugins: [i18n]
      }
    })

    // Wait for component to load workouts
    await wrapper.vm.$nextTick()
    await new Promise(resolve => setTimeout(resolve, 0))

    // Check that workouts are sorted by started date descending
    const sortedWorkouts = wrapper.vm.workouts
    
    expect(sortedWorkouts).toHaveLength(3)
    expect(sortedWorkouts[0].id).toBe(2) // 2024-01-20 (newest)
    expect(sortedWorkouts[1].id).toBe(1) // 2024-01-15 (middle)
    expect(sortedWorkouts[2].id).toBe(3) // 2024-01-10 (oldest)
  })

  it('sorts workouts within month groups by started date descending', async () => {
    // Create test data with workouts in the same month but different dates
    const testWorkouts = [
      {
        id: 1,
        name: 'Workout 1',
        started: '2024-01-05T10:00:00Z', // Earlier in month
        ended: '2024-01-05T11:00:00Z',
        exercises: []
      },
      {
        id: 2,
        name: 'Workout 2',
        started: '2024-01-25T10:00:00Z', // Later in month
        ended: '2024-01-25T11:00:00Z',
        exercises: []
      },
      {
        id: 3,
        name: 'Workout 3',
        started: '2024-01-15T10:00:00Z', // Middle of month
        ended: '2024-01-15T11:00:00Z',
        exercises: []
      },
      {
        id: 4,
        name: 'Workout 4',
        started: '2024-02-10T10:00:00Z', // Different month
        ended: '2024-02-10T11:00:00Z',
        exercises: []
      }
    ]

    mockGetWorkouts.mockResolvedValue(testWorkouts)

    const wrapper = mount(WorkoutLog, {
      global: {
        plugins: [i18n]
      }
    })

    // Wait for component to load workouts
    await wrapper.vm.$nextTick()
    await new Promise(resolve => setTimeout(resolve, 0))

    // Check grouped workouts
    const grouped = wrapper.vm.groupedWorkouts
    
    expect(grouped).toHaveLength(2) // Two months: January and February

    // Find January group (should be second due to month sorting)
    const januaryGroup = grouped.find(group => {
      const date = new Date(group.workouts[0].started)
      return date.getMonth() === 0 // January is month 0
    })
    
    expect(januaryGroup).toBeDefined()
    expect(januaryGroup.workouts).toHaveLength(3)
    
    // Check that workouts within January are sorted by date descending
    const janWorkouts = januaryGroup.workouts
    expect(janWorkouts[0].id).toBe(2) // 2024-01-25 (latest)
    expect(janWorkouts[1].id).toBe(3) // 2024-01-15 (middle)
    expect(janWorkouts[2].id).toBe(1) // 2024-01-05 (earliest)
  })

  it('sorts month groups by date descending (newest month first)', async () => {
    // Create test data with workouts in different months
    const testWorkouts = [
      {
        id: 1,
        name: 'Workout 1',
        started: '2024-01-15T10:00:00Z', // January
        ended: '2024-01-15T11:00:00Z',
        exercises: []
      },
      {
        id: 2,
        name: 'Workout 2',
        started: '2024-03-10T10:00:00Z', // March (newest)
        ended: '2024-03-10T11:00:00Z',
        exercises: []
      },
      {
        id: 3,
        name: 'Workout 3',
        started: '2024-02-20T10:00:00Z', // February
        ended: '2024-02-20T11:00:00Z',
        exercises: []
      }
    ]

    mockGetWorkouts.mockResolvedValue(testWorkouts)

    const wrapper = mount(WorkoutLog, {
      global: {
        plugins: [i18n]
      }
    })

    // Wait for component to load workouts
    await wrapper.vm.$nextTick()
    await new Promise(resolve => setTimeout(resolve, 0))

    // Check grouped workouts
    const grouped = wrapper.vm.groupedWorkouts
    
    expect(grouped).toHaveLength(3) // Three months

    // Check that months are sorted newest first
    const firstGroup = grouped[0]
    const secondGroup = grouped[1] 
    const thirdGroup = grouped[2]

    expect(new Date(firstGroup.workouts[0].started).getMonth()).toBe(2) // March (month 2)
    expect(new Date(secondGroup.workouts[0].started).getMonth()).toBe(1) // February (month 1)
    expect(new Date(thirdGroup.workouts[0].started).getMonth()).toBe(0) // January (month 0)
  })

  it('handles workouts with same started date correctly', async () => {
    // Create test data with identical started dates
    const testWorkouts = [
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
        started: '2024-01-15T10:00:00Z', // Same date
        ended: '2024-01-15T11:30:00Z',
        exercises: []
      }
    ]

    mockGetWorkouts.mockResolvedValue(testWorkouts)

    const wrapper = mount(WorkoutLog, {
      global: {
        plugins: [i18n]
      }
    })

    // Wait for component to load workouts
    await wrapper.vm.$nextTick()
    await new Promise(resolve => setTimeout(resolve, 0))

    // Check that both workouts are present and handled correctly
    const grouped = wrapper.vm.groupedWorkouts
    expect(grouped).toHaveLength(1)
    expect(grouped[0].workouts).toHaveLength(2)
    
    // Both workouts should be in the same group
    const workoutIds = grouped[0].workouts.map(w => w.id)
    expect(workoutIds).toContain(1)
    expect(workoutIds).toContain(2)
  })

  it('verifies flattened items maintain proper workout order', async () => {
    // Create test data spanning multiple months
    const testWorkouts = [
      {
        id: 1,
        name: 'January Early',
        started: '2024-01-05T10:00:00Z',
        ended: '2024-01-05T11:00:00Z',
        exercises: []
      },
      {
        id: 2,
        name: 'January Late', 
        started: '2024-01-25T10:00:00Z',
        ended: '2024-01-25T11:00:00Z',
        exercises: []
      },
      {
        id: 3,
        name: 'February Workout',
        started: '2024-02-10T10:00:00Z',
        ended: '2024-02-10T11:00:00Z',
        exercises: []
      }
    ]

    mockGetWorkouts.mockResolvedValue(testWorkouts)

    const wrapper = mount(WorkoutLog, {
      global: {
        plugins: [i18n]
      }
    })

    // Wait for component to load workouts
    await wrapper.vm.$nextTick()
    await new Promise(resolve => setTimeout(resolve, 0))

    // Check flattened items for virtual scroller
    const flattened = wrapper.vm.flattenedItems
    
    // Should have: Feb header, Feb workout, Jan header, Jan late workout, Jan early workout
    expect(flattened).toHaveLength(5)
    
    // First should be February header
    expect(flattened[0].type).toBe('header')
    
    // Second should be February workout
    expect(flattened[1].type).toBe('workout')
    expect(flattened[1].id).toBe(3)
    
    // Third should be January header
    expect(flattened[2].type).toBe('header')
    
    // Fourth should be January late workout (newest in January)
    expect(flattened[3].type).toBe('workout') 
    expect(flattened[3].id).toBe(2)
    
    // Fifth should be January early workout (oldest in January)
    expect(flattened[4].type).toBe('workout')
    expect(flattened[4].id).toBe(1)
  })
})