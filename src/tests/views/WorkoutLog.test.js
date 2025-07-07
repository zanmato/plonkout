import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import WorkoutLog from '@/views/WorkoutLog.vue'
import { getWorkouts } from '@/utils/database.js'
import { mockPush } from '../setup.js'

// Mock VirtualList component
vi.mock('vue3-virtual-scroller', () => ({
  VirtualList: {
    name: 'VirtualList',
    template: `
      <div class="virtual-list">
        <div v-for="item in items" :key="item.id">
          <slot :item="item" :index="0"></slot>
        </div>
      </div>
    `,
    props: ['items', 'itemHeight', 'itemKey']
  }
}))

// Mock vue-i18n
const mockT = vi.fn((key, params) => {
  const translations = {
    'workouts.title': 'Workouts',
    'workout.loading': 'Loading...',
    'workouts.noWorkouts': 'No workouts yet',
    'workout.workoutCount': params ? `${params.count} workout${params.count !== 1 ? 's' : ''}` : 'workouts',
    'common.today': 'Today',
    'common.yesterday': 'Yesterday',
    'workout.summary.noExercises': 'No exercises',
    'workout.summary.singleExercise': params ? `${params.totalSets} sets of ${params.exerciseName}` : 'single exercise',
    'workout.summary.multipleExercises': params ? `${params.totalSets} sets of ${params.exerciseCount} exercises` : 'multiple exercises'
  }
  
  return translations[key] || key
})

const mockD = vi.fn((date) => date?.toLocaleDateString() || '')

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: mockT,
    d: mockD
  })
}))

describe('WorkoutLog.vue', () => {
  let wrapper

  const createWrapper = () => {
    return mount(WorkoutLog, {
      global: {
        stubs: {
          VirtualList: {
            template: `
              <div class="virtual-list">
                <div v-for="item in items" :key="item.id" class="virtual-item">
                  <slot :item="item" :index="0"></slot>
                </div>
              </div>
            `,
            props: ['items', 'itemHeight', 'itemKey']
          }
        }
      }
    })
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockPush.mockClear()
  })

  describe('Component Rendering', () => {
    beforeEach(async () => {
      wrapper = createWrapper()
      await nextTick()
    })

    it('renders the workout log header', () => {
      expect(wrapper.find('h1').text()).toBe('Workouts')
      expect(wrapper.find('button').text()).toBe('+')
    })

    it('shows loading state initially', () => {
      expect(wrapper.find('.text-gray-500').text()).toBe('Loading...')
    })

    it('navigates to new workout when add button is clicked', async () => {
      const addButton = wrapper.find('button')
      await addButton.trigger('click')
      
      expect(mockPush).toHaveBeenCalledWith('/workout')
    })
  })

  describe('Data Loading and Display', () => {
    beforeEach(async () => {
      // Mock successful data loading
      getWorkouts.mockResolvedValueOnce([
        {
          id: 1,
          name: 'Test Workout 1',
          started: new Date('2024-01-15T10:00:00'),
          ended: new Date('2024-01-15T11:30:00'),
          exercises: [
            {
              name: 'Wrist Curl',
              sets: [
                { type: 'regular', weight: 20, reps: 10 },
                { type: 'regular', weight: 22, reps: 8 }
              ]
            }
          ]
        },
        {
          id: 2,
          name: 'Test Workout 2',
          started: new Date('2024-01-16T14:00:00'),
          ended: new Date('2024-01-16T15:45:00'),
          exercises: [
            {
              name: 'Hook Training',
              sets: [
                { type: 'warmup', weight: 15, reps: 12 },
                { type: 'regular', weight: 25, reps: 6 }
              ]
            },
            {
              name: 'Side Pressure',
              sets: [
                { type: 'regular', weight: 30, reps: 5 }
              ]
            }
          ]
        }
      ])

      wrapper = createWrapper()
      await nextTick()
      // Wait for data to load
      await new Promise(resolve => setTimeout(resolve, 50))
    })

    it('loads and displays workouts', async () => {
      expect(getWorkouts).toHaveBeenCalled()
      
      // Check that loading state is gone
      expect(wrapper.find('.text-gray-500').exists()).toBe(false)
    })

    it('groups workouts by month', () => {
      const vm = wrapper.vm
      expect(vm.groupedWorkouts).toHaveLength(1) // Both workouts are in January 2024
      expect(vm.groupedWorkouts[0].monthYear).toBe('January 2024')
      expect(vm.groupedWorkouts[0].workouts).toHaveLength(2)
    })

    it('flattens items for virtual scrolling', () => {
      const vm = wrapper.vm
      expect(vm.flattenedItems).toHaveLength(3) // 1 header + 2 workouts
      expect(vm.flattenedItems[0].type).toBe('header')
      expect(vm.flattenedItems[1].type).toBe('workout')
      expect(vm.flattenedItems[2].type).toBe('workout')
    })

    it('calculates workout summaries correctly', () => {
      const vm = wrapper.vm
      const workout1 = vm.flattenedItems[1]
      const workout2 = vm.flattenedItems[2]
      
      // Workout 1: 1 exercise, 2 sets
      expect(vm.getWorkoutSummary(workout1)).toBe('2 sets of Wrist Curl')
      
      // Workout 2: 2 exercises, 2 sets total (1 warmup + 1 regular + 1 regular)
      expect(vm.getWorkoutSummary(workout2)).toBe('2 sets of 2 exercises')
    })
  })

  describe('Empty State', () => {
    beforeEach(async () => {
      getWorkouts.mockResolvedValueOnce([])
      wrapper = createWrapper()
      await nextTick()
      await new Promise(resolve => setTimeout(resolve, 50))
    })

    it('shows empty state when no workouts exist', () => {
      expect(wrapper.text()).toContain('No workouts yet')
      expect(wrapper.find('.text-lg').text()).toBe('💪')
    })
  })

  describe('Date Formatting', () => {
    beforeEach(() => {
      wrapper = createWrapper()
    })

    it('formats dates correctly', () => {
      const vm = wrapper.vm
      const today = new Date()
      const yesterday = new Date()
      yesterday.setDate(yesterday.getDate() - 1)
      const otherDate = new Date('2024-01-15')

      expect(vm.formatWorkoutDate(today)).toBe('Today')
      expect(vm.formatWorkoutDate(yesterday)).toBe('Yesterday')
      expect(vm.formatWorkoutDate(otherDate)).toBe(otherDate.toLocaleDateString())
    })

    it('formats duration correctly', () => {
      const vm = wrapper.vm
      const start = new Date('2024-01-15T10:00:00')
      const end = new Date('2024-01-15T11:30:00')
      
      expect(vm.formatDuration(start, end)).toBe('1h 30m')
      
      const shortEnd = new Date('2024-01-15T10:45:00')
      expect(vm.formatDuration(start, shortEnd)).toBe('45m')
    })
  })

  describe('Virtual Scrolling', () => {
    beforeEach(async () => {
      // Create many workouts to test virtual scrolling
      const manyWorkouts = Array.from({ length: 100 }, (_, i) => ({
        id: i + 1,
        name: `Workout ${i + 1}`,
        started: new Date(`2024-01-${Math.floor(i / 31) + 1}T10:00:00`),
        ended: new Date(`2024-01-${Math.floor(i / 31) + 1}T11:00:00`),
        exercises: [
          {
            name: 'Test Exercise',
            sets: [{ type: 'regular', weight: 20, reps: 10 }]
          }
        ]
      }))
      
      getWorkouts.mockResolvedValueOnce(manyWorkouts)
      wrapper = createWrapper()
      await nextTick()
      await new Promise(resolve => setTimeout(resolve, 50))
    })

    it('estimates item height correctly', () => {
      const vm = wrapper.vm
      
      // Test header height
      const headerItem = { type: 'header' }
      expect(vm.estimateItemHeight(headerItem)).toBe(60)
      
      // Test workout height
      const workoutItem = { type: 'workout' }
      expect(vm.estimateItemHeight(workoutItem)).toBe(120)
      
      // Test default height
      const unknownItem = { type: 'unknown' }
      expect(vm.estimateItemHeight(unknownItem)).toBe(120)
    })
  })

  describe('Error Handling', () => {
    beforeEach(async () => {
      getWorkouts.mockRejectedValueOnce(new Error('Database error'))
      wrapper = createWrapper()
      await nextTick()
      await new Promise(resolve => setTimeout(resolve, 50))
    })

    it('handles loading errors gracefully', () => {
      expect(wrapper.vm.loading).toBe(false)
      expect(wrapper.vm.workouts).toEqual([])
    })
  })

  describe('Navigation', () => {
    beforeEach(async () => {
      getWorkouts.mockResolvedValueOnce([
        {
          id: 1,
          name: 'Test Workout',
          started: new Date(),
          ended: new Date(),
          exercises: []
        }
      ])
      
      wrapper = createWrapper()
      await nextTick()
      await new Promise(resolve => setTimeout(resolve, 50))
    })

    it('navigates to workout edit when workout is clicked', async () => {
      // Find the virtual list and simulate clicking on a workout
      const virtualList = wrapper.find('.virtual-list')
      expect(virtualList.exists()).toBe(true)
      
      // Check if workout click handler exists in component
      const vm = wrapper.vm
      expect(typeof vm.editWorkout).toBe('function')
      
      // Test navigation function directly
      vm.editWorkout(1)
      expect(mockPush).toHaveBeenCalledWith('/workout/1')
    })
  })
})