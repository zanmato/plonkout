import { vi } from 'vitest'
import { config } from '@vue/test-utils'

// Mock database module
vi.mock('@/utils/database.js', async () => {
  const mocks = await import('./mocks/database.js')
  return mocks
})

// Mock PrimeVue Toast
vi.mock('primevue/usetoast', () => ({
  useToast: () => ({
    add: vi.fn(),
    removeGroup: vi.fn(),
    removeAllGroups: vi.fn()
  })
}))

// Mock vue-router
const mockPush = vi.fn()
const mockReplace = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: mockPush,
    replace: mockReplace,
    go: vi.fn(),
    back: vi.fn(),
    forward: vi.fn()
  }),
  useRoute: () => ({
    params: {},
    query: {},
    path: '/',
    name: 'test'
  })
}))

// Mock global confirm and alert functions
global.confirm = vi.fn(() => true)
global.alert = vi.fn()

// Global test utilities
config.global.mocks = {
  $t: (key, params) => {
    const translations = {
      'workouts.title': 'Workouts',
      'workout.loading': 'Loading...',
      'workouts.noWorkouts': 'No workouts yet',
      'statistics.title': 'Statistics',
      'statistics.loading': 'Loading statistics...',
      'statistics.noData': 'No workout data available',
      'settings.title': 'Settings'
    }
    
    const translated = translations[key] || key
    if (params) {
      return translated.replace(/\{(\w+)\}/g, (match, paramKey) => params[paramKey] || match)
    }
    return translated
  },
  $d: (date) => date?.toLocaleDateString() || ''
}

// Export router mocks for test use
export { mockPush, mockReplace }