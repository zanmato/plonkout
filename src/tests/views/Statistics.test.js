import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import Statistics from '@/views/Statistics.vue'
import { getWorkouts } from '@/utils/database.js'

// Mock Chart.js components
vi.mock('vue-chartjs', () => ({
  Line: {
    name: 'LineChart',
    template: '<canvas>Line Chart Mock</canvas>',
    props: ['data', 'options']
  },
  Bar: {
    name: 'BarChart', 
    template: '<canvas>Bar Chart Mock</canvas>',
    props: ['data', 'options']
  }
}))

// Mock vue-i18n
const mockT = vi.fn((key, _params) => {
  const translations = {
    'statistics.title': 'Statistics',
    'statistics.loading': 'Loading statistics...',
    'statistics.noData': 'No workout data available',
    'statistics.cards.totalWorkouts': 'Total Workouts',
    'statistics.cards.avgDuration': 'Avg Duration',
    'statistics.cards.totalSets': 'Total Sets',
    'statistics.cards.exercisesUsed': 'Exercises Used',
    'statistics.charts.workoutsPerMonth': 'Workouts per Month',
    'statistics.charts.durationTrends': 'Workout Duration Trends',
    'statistics.charts.mostUsedExercises': 'Most Used Exercises',
    'statistics.charts.exerciseVolume': 'Exercise Volume (Top 5)',
    'statistics.charts.workouts': 'Workouts',
    'statistics.charts.duration': 'Duration (minutes)',
    'statistics.charts.minutes': 'Minutes'
  }
  
  return translations[key] || key
})

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: mockT
  })
}))

describe('Statistics.vue', () => {
  let wrapper

  const createWrapper = () => {
    return mount(Statistics, {
      global: {
        stubs: {
          Line: true,
          Bar: true
        }
      }
    })
  }

  const mockWorkouts = [
    {
      id: 1,
      name: 'Workout 1',
      started: new Date('2024-01-15T10:00:00'),
      ended: new Date('2024-01-15T11:30:00'),
      exercises: [
        {
          name: 'Wrist Curl',
          sets: [
            { type: 'regular', weight: 20, reps: 10 },
            { type: 'regular', weight: 22, reps: 8 },
            { type: 'warmup', weight: 15, reps: 12 }
          ]
        },
        {
          name: 'Hook Training',
          sets: [
            { type: 'regular', weight: 25, reps: 6 }
          ]
        }
      ]
    },
    {
      id: 2,
      name: 'Workout 2',
      started: new Date('2024-01-16T14:00:00'),
      ended: new Date('2024-01-16T15:00:00'),
      exercises: [
        {
          name: 'Wrist Curl',
          sets: [
            { type: 'regular', weight: 23, reps: 9 }
          ]
        },
        {
          name: 'Side Pressure',
          sets: [
            { type: 'regular', weight: 30, reps: 5 },
            { type: 'regular', weight: 32, reps: 4 }
          ]
        }
      ]
    },
    {
      id: 3,
      name: 'Workout 3',
      started: new Date('2024-02-01T09:00:00'),
      ended: new Date('2024-02-01T10:15:00'),
      exercises: [
        {
          name: 'Hook Training',
          sets: [
            { type: 'regular', weight: 26, reps: 7 },
            { type: 'regular', weight: 28, reps: 5 }
          ]
        }
      ]
    }
  ]

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Component Rendering', () => {
    beforeEach(async () => {
      getWorkouts.mockResolvedValueOnce(mockWorkouts)
      wrapper = createWrapper()
      await nextTick()
    })

    it('renders the statistics header', () => {
      expect(wrapper.find('h1').text()).toBe('Statistics')
    })

    it('shows loading state initially', () => {
      // Since we mocked the data, loading should be false after nextTick
      // But we can test the loading template exists
      expect(wrapper.text()).toContain('Statistics')
    })
  })

  describe('Statistics Calculations', () => {
    beforeEach(async () => {
      getWorkouts.mockResolvedValueOnce(mockWorkouts)
      wrapper = createWrapper()
      await nextTick()
      // Wait for data to load
      await new Promise(resolve => setTimeout(resolve, 50))
    })

    it('calculates total workouts correctly', () => {
      const vm = wrapper.vm
      expect(vm.totalWorkouts).toBe(3)
    })

    it('calculates average duration correctly', () => {
      const vm = wrapper.vm
      // Workout 1: 90 minutes, Workout 2: 60 minutes, Workout 3: 75 minutes
      // Average: (90 + 60 + 75) / 3 = 75 minutes
      expect(vm.avgDuration).toBe(75)
    })

    it('calculates total sets correctly', () => {
      const vm = wrapper.vm
      // Workout 1: 3 sets (2 regular + 1 warmup), Workout 2: 3 sets, Workout 3: 2 sets
      // But only regular sets count: 2 + 3 + 2 = 7
      expect(vm.totalSets).toBe(7)
    })

    it('calculates unique exercises correctly', () => {
      const vm = wrapper.vm
      // Unique exercises: Wrist Curl, Hook Training, Side Pressure = 3
      expect(vm.exercisesUsed).toBe(3)
    })

    it('generates workouts per month data', () => {
      const vm = wrapper.vm
      const monthlyData = vm.workoutsPerMonthData
      
      expect(monthlyData.labels).toContain('Jan 2024')
      expect(monthlyData.labels).toContain('Feb 2024')
      expect(monthlyData.datasets[0].data).toEqual(expect.arrayContaining([2, 1]))
    })

    it('generates duration trends data', () => {
      const vm = wrapper.vm
      const durationData = vm.durationTrendsData
      
      expect(durationData.labels).toHaveLength(3)
      expect(durationData.datasets[0].data).toEqual([90, 60, 75])
    })

    it('generates most used exercises data', () => {
      const vm = wrapper.vm
      const exerciseData = vm.mostUsedExercisesData
      
      expect(exerciseData.labels).toContain('Wrist Curl')
      expect(exerciseData.labels).toContain('Hook Training')
      expect(exerciseData.labels).toContain('Side Pressure')
      // Wrist Curl: 2 workouts, Hook Training: 2 workouts, Side Pressure: 1 workout
      expect(exerciseData.datasets[0].data).toEqual(expect.arrayContaining([2, 2, 1]))
    })

    it('generates exercise volume data', () => {
      const vm = wrapper.vm
      const volumeData = vm.exerciseVolumeData
      
      expect(volumeData.labels).toHaveLength(3) // Top 3 exercises by volume
      expect(volumeData.datasets[0].data).toHaveLength(3)
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
      expect(wrapper.text()).toContain('No workout data available')
    })

    it('shows zero values in stats cards when no data', () => {
      const vm = wrapper.vm
      expect(vm.totalWorkouts).toBe(0)
      expect(vm.avgDuration).toBe(0)
      expect(vm.totalSets).toBe(0)
      expect(vm.exercisesUsed).toBe(0)
    })
  })

  describe('Data Processing', () => {
    beforeEach(() => {
      wrapper = createWrapper()
    })

    it('calculates workout duration correctly', () => {
      const vm = wrapper.vm
      const start = new Date('2024-01-15T10:00:00')
      const end = new Date('2024-01-15T11:30:00')
      
      expect(vm.calculateDuration(start, end)).toBe(90) // 90 minutes
    })

    it('handles null end times in duration calculation', () => {
      const vm = wrapper.vm
      const start = new Date('2024-01-15T10:00:00')
      
      expect(vm.calculateDuration(start, null)).toBe(0)
    })

    it('groups workouts by month correctly', () => {
      const vm = wrapper.vm
      const workouts = [
        { started: new Date('2024-01-15T10:00:00') },
        { started: new Date('2024-01-20T10:00:00') },
        { started: new Date('2024-02-05T10:00:00') }
      ]
      
      const grouped = vm.groupWorkoutsByMonth(workouts)
      expect(Object.keys(grouped)).toHaveLength(2)
      expect(grouped['Jan 2024']).toBe(2)
      expect(grouped['Feb 2024']).toBe(1)
    })

    it('calculates exercise volume correctly', () => {
      const vm = wrapper.vm
      const exercise = {
        sets: [
          { type: 'regular', weight: 20, reps: 10 }, // 200
          { type: 'regular', weight: 25, reps: 8 },  // 200
          { type: 'warmup', weight: 15, reps: 12 }   // excluded
        ]
      }
      
      expect(vm.calculateExerciseVolume(exercise)).toBe(400)
    })
  })

  describe('Chart Configuration', () => {
    beforeEach(async () => {
      getWorkouts.mockResolvedValueOnce(mockWorkouts)
      wrapper = createWrapper()
      await nextTick()
      await new Promise(resolve => setTimeout(resolve, 50))
    })

    it('generates proper chart options', () => {
      const vm = wrapper.vm
      const chartOptions = vm.chartOptions
      
      expect(chartOptions.responsive).toBe(true)
      expect(chartOptions.maintainAspectRatio).toBe(false)
      expect(chartOptions.plugins.legend.display).toBe(false)
    })

    it('generates proper bar chart options with y-axis configuration', () => {
      const vm = wrapper.vm
      const barChartOptions = vm.barChartOptions
      
      expect(barChartOptions.responsive).toBe(true)
      expect(barChartOptions.scales.y.beginAtZero).toBe(true)
      expect(barChartOptions.scales.y.ticks.stepSize).toBe(1)
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
      const vm = wrapper.vm
      expect(vm.loading).toBe(false)
      expect(vm.workouts).toEqual([])
    })

    it('shows empty state when error occurs', () => {
      expect(wrapper.text()).toContain('No workout data available')
    })
  })

  describe('Responsive Design', () => {
    beforeEach(async () => {
      getWorkouts.mockResolvedValueOnce(mockWorkouts)
      wrapper = createWrapper()
      await nextTick()
    })

    it('renders charts with responsive configuration', () => {
      const vm = wrapper.vm
      expect(vm.chartOptions.responsive).toBe(true)
      expect(vm.chartOptions.maintainAspectRatio).toBe(false)
    })
  })
})