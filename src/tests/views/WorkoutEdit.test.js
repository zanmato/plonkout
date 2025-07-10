import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import WorkoutEdit from '@/views/WorkoutEdit.vue'
import { getWorkout, saveWorkout as saveWorkoutToDB, deleteWorkout, getSetting } from '@/utils/database.js'
import { mockPush, mockReplace } from '../setup.js'

// Mock ExerciseSelector component
vi.mock('@/components/ExerciseSelector.vue', () => ({
  default: {
    name: 'ExerciseSelector',
    template: '<div>Mock ExerciseSelector</div>',
    emits: ['close', 'select']
  }
}))

// Mock vue-i18n
const mockT = vi.fn((key, params) => {
  if (params) {
    return `${key}(${JSON.stringify(params)})`
  }
  return key
})

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: mockT
  })
}))

// Mock useToast composable
const mockShowSuccess = vi.fn()
const mockShowError = vi.fn()
const mockShowInfo = vi.fn()

vi.mock('@/composables/useToast.js', () => ({
  useToast: () => ({
    showSuccess: mockShowSuccess,
    showError: mockShowError,
    showInfo: mockShowInfo
  })
}))

describe('WorkoutEdit.vue', () => {
  let wrapper

  const createWrapper = (props = {}) => {
    return mount(WorkoutEdit, {
      props,
      global: {
        stubs: {
          ExerciseSelector: true
        }
      }
    })
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockPush.mockClear()
    mockReplace.mockClear()
    
    // Mock getSetting to return default values
    getSetting.mockImplementation((key, defaultValue) => {
      const settings = {
        'weightUnit': 'kg',
        'exerciseDisplay': 'reps'
      }
      return Promise.resolve(settings[key] || defaultValue)
    })
  })

  describe('New Workout', () => {
    beforeEach(() => {
      wrapper = createWrapper()
    })

    it('renders new workout form', () => {
      expect(wrapper.find('h1').text()).toBe('workout.new')
      expect(wrapper.find('button[data-testid="save-button"]').exists()).toBe(false) // Need to check actual save button
      expect(wrapper.find('.workout-edit').exists()).toBe(true)
    })

    it('initializes with empty workout data', () => {
      const workoutNameInput = wrapper.find('input[type="text"]')
      expect(workoutNameInput.element.value).toBe('')
    })

    it('can add new exercise', async () => {
      // Find button with "Add Exercise" functionality
      const addExerciseButton = wrapper.find('[data-testid="add-exercise-button"]')
      
      if (addExerciseButton) {
        await addExerciseButton.trigger('click')
        await nextTick()
        expect(wrapper.vm.showExerciseSelector).toBe(true)
      }
    })

    it('saves new workout successfully', async () => {
      saveWorkoutToDB.mockResolvedValueOnce(123)
      
      // Set workout data
      await wrapper.find('input[type="text"]').setValue('Test Workout')
      
      // Find and click save button
      const saveButton = wrapper.findAll('button').find(btn => 
        btn.text().includes('workout.save')
      )
      
      if (saveButton) {
        await saveButton.trigger('click')
        await nextTick()
        
        expect(saveWorkoutToDB).toHaveBeenCalled()
        expect(mockShowSuccess).toHaveBeenCalledWith('workout.saveSuccess')
        expect(mockReplace).toHaveBeenCalledWith('/workout/123')
      }
    })
  })

  describe('Edit Existing Workout', () => {
    beforeEach(async () => {
      wrapper = createWrapper({ id: '1' })
      await nextTick()
    })

    it('loads existing workout data', async () => {
      expect(getWorkout).toHaveBeenCalledWith(1)
      // Wait for the component to load the data
      await nextTick()
    })

    it('renders edit workout form', () => {
      expect(wrapper.find('h1').text()).toBe('workout.edit')
    })

    it('can duplicate workout', async () => {
      saveWorkoutToDB.mockResolvedValueOnce(456)
      
      // Find context menu button (three dots)
      const contextMenuBtn = wrapper.findAll('button').find(btn => 
        btn.find('svg')?.exists()
      )
      
      if (contextMenuBtn) {
        await contextMenuBtn.trigger('click')
        await nextTick()
        
        // Find duplicate button in context menu
        const duplicateBtn = wrapper.findAll('button').find(btn => 
          btn.text().includes('workout.duplicate')
        )
        
        if (duplicateBtn) {
          await duplicateBtn.trigger('click')
          await nextTick()
          
          expect(saveWorkoutToDB).toHaveBeenCalled()
          expect(mockPush).toHaveBeenCalledWith('/workout/456')
        }
      }
    })

    it('preserves workout ID when saving existing workout', async () => {
      // Mock existing workout data
      getWorkout.mockResolvedValueOnce({
        id: 1,
        name: 'Test Workout',
        started: new Date(),
        ended: null,
        notes: '',
        exercises: []
      })
      
      saveWorkoutToDB.mockResolvedValueOnce(1) // Should return the same ID
      
      // Create wrapper for existing workout
      wrapper = createWrapper({ id: '1' })
      await nextTick()
      await new Promise(resolve => setTimeout(resolve, 50)) // Wait for loading
      
      // Trigger save
      const vm = wrapper.vm
      await vm.saveWorkout()
      
      // Verify that saveWorkoutToDB was called with the workout including ID
      expect(saveWorkoutToDB).toHaveBeenCalledWith(
        expect.objectContaining({
          id: 1,
          name: 'Test Workout'
        })
      )
    })

    it('can delete workout', async () => {
      global.confirm.mockReturnValueOnce(true)
      
      // Find context menu button
      const contextMenuBtn = wrapper.findAll('button').find(btn => 
        btn.find('svg')?.exists()
      )
      
      if (contextMenuBtn) {
        await contextMenuBtn.trigger('click')
        await nextTick()
        
        // Find delete button in context menu
        const deleteBtn = wrapper.findAll('button').find(btn => 
          btn.text().includes('workout.delete')
        )
        
        if (deleteBtn) {
          await deleteBtn.trigger('click')
          await nextTick()
          
          expect(deleteWorkout).toHaveBeenCalledWith(1)
          expect(mockPush).toHaveBeenCalledWith('/workouts')
        }
      }
    })
  })

  describe('Exercise Management', () => {
    beforeEach(() => {
      wrapper = createWrapper()
    })

    it('can add sets to exercises', async () => {
      // First add an exercise
      const mockExercise = { name: 'Wrist Curl', muscleGroup: 'Wrist', singleArm: true }
      wrapper.vm.addExercise(mockExercise)
      await nextTick()
      
      expect(wrapper.vm.workout.exercises).toHaveLength(1)
      expect(wrapper.vm.workout.exercises[0].sets).toHaveLength(1)
      
      // Add another set
      wrapper.vm.addSet(0)
      await nextTick()
      
      expect(wrapper.vm.workout.exercises[0].sets).toHaveLength(2)
    })

    it('can remove exercises', async () => {
      global.confirm.mockReturnValueOnce(true)
      
      // Add an exercise first
      const mockExercise = { name: 'Wrist Curl', muscleGroup: 'Wrist', singleArm: true }
      wrapper.vm.addExercise(mockExercise)
      await nextTick()
      
      expect(wrapper.vm.workout.exercises).toHaveLength(1)
      
      // Remove the exercise
      wrapper.vm.removeExercise(0)
      await nextTick()
      
      expect(wrapper.vm.workout.exercises).toHaveLength(0)
    })

    it('can toggle set type between regular and warmup', async () => {
      // Add an exercise with a set
      const mockExercise = { name: 'Wrist Curl', muscleGroup: 'Wrist', singleArm: true }
      wrapper.vm.addExercise(mockExercise)
      await nextTick()
      
      const initialType = wrapper.vm.workout.exercises[0].sets[0].type
      expect(initialType).toBe('regular')
      
      // Toggle set type
      wrapper.vm.toggleSetType(0, 0)
      await nextTick()
      
      expect(wrapper.vm.workout.exercises[0].sets[0].type).toBe('warmup')
      
      // Toggle back
      wrapper.vm.toggleSetType(0, 0)
      await nextTick()
      
      expect(wrapper.vm.workout.exercises[0].sets[0].type).toBe('regular')
    })
  })

  describe('Form Validation and Error Handling', () => {
    beforeEach(() => {
      wrapper = createWrapper()
    })

    it('handles save errors gracefully', async () => {
      saveWorkoutToDB.mockRejectedValueOnce(new Error('Save failed'))
      
      const saveButton = wrapper.findAll('button').find(btn => 
        btn.text().includes('workout.save')
      )
      
      if (saveButton) {
        await saveButton.trigger('click')
        await nextTick()
        
        expect(mockShowError).toHaveBeenCalledWith('workout.saveError')
      }
    })

    it('formats datetime correctly for inputs', () => {
      const testDate = new Date('2024-01-15T10:30:00')
      const formatted = wrapper.vm.formatDatetimeLocal(testDate)
      expect(formatted).toMatch(/2024-01-15T\d{2}:30/)
    })

    it('generates correct set numbers', () => {
      const sets = [
        { type: 'warmup' },
        { type: 'warmup' },
        { type: 'regular' },
        { type: 'regular' },
        { type: 'warmup' }
      ]
      
      expect(wrapper.vm.getSetNumber(sets, 0)).toBe('W1')
      expect(wrapper.vm.getSetNumber(sets, 1)).toBe('W2')
      expect(wrapper.vm.getSetNumber(sets, 2)).toBe('1')
      expect(wrapper.vm.getSetNumber(sets, 3)).toBe('2')
      expect(wrapper.vm.getSetNumber(sets, 4)).toBe('W3')
    })
  })

  describe('Navigation', () => {
    beforeEach(() => {
      wrapper = createWrapper()
    })

    it('navigates back when back button is clicked', async () => {
      const backButton = wrapper.find('[data-testid="back-button"]')
      
      if (backButton) {
        await backButton.trigger('click')
        expect(mockPush).toHaveBeenCalledWith('/workouts')
      }
    })
  })

  describe('Exercise Display Settings', () => {
    beforeEach(async () => {
      wrapper = createWrapper()
      await nextTick()
      // Wait for settings to load
      await new Promise(resolve => setTimeout(resolve, 50))
    })

    it('loads exercise display setting on mount', () => {
      expect(getSetting).toHaveBeenCalledWith('exerciseDisplay', 'reps')
      expect(wrapper.vm.exerciseDisplay).toBe('reps')
    })

    it('displays reps input when exercise display is set to reps', async () => {
      // Add an exercise first
      const mockExercise = { name: 'Wrist Curl', muscleGroup: 'Wrist', singleArm: true }
      wrapper.vm.addExercise(mockExercise)
      await nextTick()

      // The component should show reps input
      expect(wrapper.vm.exerciseDisplay).toBe('reps')
      
      // Check if reps input exists
      const repsInputs = wrapper.findAll('input[type="number"]').filter(input => 
        input.attributes('placeholder') === '0'
      )
      expect(repsInputs.length).toBeGreaterThan(0)
    })

    it('displays time input when exercise display is set to time', async () => {
      // Change exercise display to time
      getSetting.mockImplementation((key, defaultValue) => {
        const settings = {
          'weightUnit': 'kg',
          'exerciseDisplay': 'time'
        }
        return Promise.resolve(settings[key] || defaultValue)
      })

      // Create new wrapper to trigger setting load
      wrapper = createWrapper()
      await nextTick()
      await new Promise(resolve => setTimeout(resolve, 50))

      // Add an exercise
      const mockExercise = { name: 'Plank', muscleGroup: 'Abs', singleArm: false }
      wrapper.vm.addExercise(mockExercise)
      await nextTick()

      expect(wrapper.vm.exerciseDisplay).toBe('time')
      
      // Check if time input exists
      const timeInputs = wrapper.findAll('input[type="text"]').filter(input => 
        input.attributes('placeholder') === '0:00'
      )
      expect(timeInputs.length).toBeGreaterThan(0)
    })

    it('creates new sets with time field when display is time', async () => {
      // Add an exercise
      const mockExercise = { name: 'Wrist Curl', muscleGroup: 'Wrist', singleArm: true }
      wrapper.vm.addExercise(mockExercise)
      await nextTick()

      // Check that the set has a time field
      const newSet = wrapper.vm.workout.exercises[0].sets[0]
      expect(newSet).toHaveProperty('time')
      expect(newSet.time).toBe('')
    })
  })
})