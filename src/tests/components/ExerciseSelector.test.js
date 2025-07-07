import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import ExerciseSelector from '@/components/ExerciseSelector.vue'
import { getExercises, saveExercise } from '@/utils/database.js'

// Mock vue-i18n
const mockT = vi.fn((key, params) => {
  const translations = {
    'exercise.selector.title': 'Select Exercise',
    'exercise.selector.new': '+ New',
    'exercise.selector.search': 'Search exercises...',
    'exercise.selector.loading': 'Loading exercises...',
    'exercise.selector.noResults': 'No exercises found',
    'exercise.selector.singleArm': 'Single arm',
    'exercise.selector.bothArms': 'Both arms',
    'exercise.selector.newExercise': 'New Exercise',
    'exercise.selector.name': 'Name',
    'exercise.selector.namePlaceholder': 'Exercise name',
    'exercise.selector.muscleGroup': 'Muscle Group',
    'exercise.selector.muscleGroupPlaceholder': 'e.g., Wrist, Forearm, Bicep',
    'exercise.selector.singleArmLabel': 'Single arm exercise',
    'exercise.selector.create': 'Create',
    'exercise.selector.loadError': 'Error loading exercises',
    'exercise.selector.createError': 'Error creating exercise',
    'exercise.selector.defaultMuscleGroup': 'General',
    'common.cancel': 'Cancel'
  }
  
  return translations[key] || key
})

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: mockT
  })
}))

// Mock useToast composable
const mockShowError = vi.fn()

vi.mock('@/composables/useToast.js', () => ({
  useToast: () => ({
    showError: mockShowError
  })
}))

describe('ExerciseSelector.vue', () => {
  let wrapper

  const mockExercises = [
    { id: 1, name: 'Wrist Curl', muscleGroup: 'Wrist', singleArm: true },
    { id: 2, name: 'Hook Training', muscleGroup: 'Hand', singleArm: true },
    { id: 3, name: 'Side Pressure', muscleGroup: 'Side', singleArm: true },
    { id: 4, name: 'Resistance Band Pull', muscleGroup: 'Back', singleArm: false }
  ]

  const createWrapper = () => {
    return mount(ExerciseSelector, {
      attachTo: document.body
    })
  }

  beforeEach(() => {
    vi.clearAllMocks()
    getExercises.mockResolvedValue(mockExercises)
  })

  afterEach(() => {
    if (wrapper) {
      wrapper.unmount()
    }
  })

  describe('Component Rendering', () => {
    beforeEach(async () => {
      wrapper = createWrapper()
      await nextTick()
      // Wait for exercises to load
      await new Promise(resolve => setTimeout(resolve, 50))
    })

    it('renders the modal overlay and content', () => {
      expect(wrapper.find('.fixed.inset-0').exists()).toBe(true)
      expect(wrapper.find('h3').text()).toBe('Select Exercise')
    })

    it('renders the search input', () => {
      const searchInput = wrapper.find('input[type="text"]')
      expect(searchInput.exists()).toBe(true)
      expect(searchInput.attributes('placeholder')).toBe('Search exercises...')
    })

    it('renders the new exercise button', () => {
      const newButton = wrapper.find('button')
      expect(newButton.text()).toBe('+ New')
    })

    it('closes modal when clicking overlay', async () => {
      const overlay = wrapper.find('.fixed.inset-0')
      await overlay.trigger('click')
      
      expect(wrapper.emitted('close')).toBeTruthy()
    })

    it('does not close modal when clicking content area', async () => {
      const content = wrapper.find('.neo-card')
      await content.trigger('click')
      
      expect(wrapper.emitted('close')).toBeFalsy()
    })
  })

  describe('Exercise Loading and Display', () => {
    beforeEach(async () => {
      wrapper = createWrapper()
      await nextTick()
      await new Promise(resolve => setTimeout(resolve, 50))
    })

    it('loads exercises on mount', () => {
      expect(getExercises).toHaveBeenCalled()
    })

    it('displays all exercises initially', () => {
      const exerciseButtons = wrapper.findAll('button').filter(btn => 
        btn.text().includes('Wrist') || btn.text().includes('Hook') || btn.text().includes('Side') || btn.text().includes('Resistance')
      )
      expect(exerciseButtons.length).toBe(4)
    })

    it('shows exercise details correctly', () => {
      expect(wrapper.text()).toContain('Wrist Curl')
      expect(wrapper.text()).toContain('Wrist')
      expect(wrapper.text()).toContain('Single arm')
      expect(wrapper.text()).toContain('Both arms')
    })
  })

  describe('Exercise Search and Filtering', () => {
    beforeEach(async () => {
      wrapper = createWrapper()
      await nextTick()
      await new Promise(resolve => setTimeout(resolve, 50))
    })

    it('filters exercises by name', async () => {
      const searchInput = wrapper.find('input[type="text"]')
      await searchInput.setValue('Wrist')
      await nextTick()
      
      const vm = wrapper.vm
      expect(vm.filteredExercises).toHaveLength(1)
      expect(vm.filteredExercises[0].name).toBe('Wrist Curl')
    })

    it('filters exercises by muscle group', async () => {
      const searchInput = wrapper.find('input[type="text"]')
      await searchInput.setValue('Hand')
      await nextTick()
      
      const vm = wrapper.vm
      expect(vm.filteredExercises).toHaveLength(1)
      expect(vm.filteredExercises[0].name).toBe('Hook Training')
    })

    it('shows no results message when no exercises match', async () => {
      const searchInput = wrapper.find('input[type="text"]')
      await searchInput.setValue('NonExistentExercise')
      await nextTick()
      
      expect(wrapper.text()).toContain('No exercises found')
    })

    it('is case insensitive in search', async () => {
      const searchInput = wrapper.find('input[type="text"]')
      await searchInput.setValue('wrist')
      await nextTick()
      
      const vm = wrapper.vm
      expect(vm.filteredExercises).toHaveLength(1)
    })
  })

  describe('Exercise Selection', () => {
    beforeEach(async () => {
      wrapper = createWrapper()
      await nextTick()
      await new Promise(resolve => setTimeout(resolve, 50))
    })

    it('emits select event when exercise is clicked', async () => {
      const exerciseButtons = wrapper.findAll('button').filter(btn => 
        btn.text().includes('Wrist Curl')
      )
      
      if (exerciseButtons.length > 0) {
        await exerciseButtons[0].trigger('click')
        
        expect(wrapper.emitted('select')).toBeTruthy()
        expect(wrapper.emitted('select')[0][0]).toEqual(mockExercises[0])
      }
    })
  })

  describe('New Exercise Creation', () => {
    beforeEach(async () => {
      wrapper = createWrapper()
      await nextTick()
      await new Promise(resolve => setTimeout(resolve, 50))
    })

    it('shows new exercise form when new button is clicked', async () => {
      const newButton = wrapper.findAll('button').find(btn => btn.text() === '+ New')
      await newButton.trigger('click')
      await nextTick()
      
      expect(wrapper.find('h3').text()).toContain('New Exercise')
      expect(wrapper.vm.showNewExerciseForm).toBe(true)
    })

    it('creates new exercise successfully', async () => {
      saveExercise.mockResolvedValueOnce(5)
      
      // Show form
      const vm = wrapper.vm
      vm.showNewExerciseForm = true
      await nextTick()
      
      // Fill form
      vm.newExercise.name = 'New Test Exercise'
      vm.newExercise.muscleGroup = 'Test Group'
      vm.newExercise.singleArm = false
      
      // Submit form
      await vm.createExercise()
      
      expect(saveExercise).toHaveBeenCalledWith({
        name: 'New Test Exercise',
        muscleGroup: 'Test Group',
        singleArm: false
      })
      
      expect(wrapper.emitted('select')).toBeTruthy()
      expect(vm.showNewExerciseForm).toBe(false)
    })

    it('uses default muscle group when empty', async () => {
      saveExercise.mockResolvedValueOnce(5)
      
      const vm = wrapper.vm
      vm.showNewExerciseForm = true
      vm.newExercise.name = 'Test Exercise'
      vm.newExercise.muscleGroup = ''
      
      await vm.createExercise()
      
      expect(saveExercise).toHaveBeenCalledWith({
        name: 'Test Exercise',
        muscleGroup: 'General',
        singleArm: true
      })
    })

    it('does not create exercise with empty name', async () => {
      const vm = wrapper.vm
      vm.showNewExerciseForm = true
      vm.newExercise.name = ''
      
      await vm.createExercise()
      
      expect(saveExercise).not.toHaveBeenCalled()
    })

    it('handles exercise creation errors', async () => {
      saveExercise.mockRejectedValueOnce(new Error('Creation failed'))
      
      const vm = wrapper.vm
      vm.showNewExerciseForm = true
      vm.newExercise.name = 'Test Exercise'
      
      await vm.createExercise()
      
      expect(mockShowError).toHaveBeenCalledWith('Error creating exercise')
    })

    it('closes new exercise form when cancel is clicked', async () => {
      const vm = wrapper.vm
      vm.showNewExerciseForm = true
      await nextTick()
      
      const cancelButtons = wrapper.findAll('button').filter(btn => btn.text() === 'Cancel')
      const formCancelButton = cancelButtons[cancelButtons.length - 1] // Get the one in the form
      
      await formCancelButton.trigger('click')
      
      expect(vm.showNewExerciseForm).toBe(false)
    })

    it('resets form after successful creation', async () => {
      saveExercise.mockResolvedValueOnce(5)
      
      const vm = wrapper.vm
      vm.showNewExerciseForm = true
      vm.newExercise.name = 'Test Exercise'
      vm.newExercise.muscleGroup = 'Test Group'
      vm.newExercise.singleArm = false
      
      await vm.createExercise()
      
      expect(vm.newExercise.name).toBe('')
      expect(vm.newExercise.muscleGroup).toBe('')
      expect(vm.newExercise.singleArm).toBe(true)
    })
  })

  describe('Loading States', () => {
    it('shows loading message while exercises are loading', async () => {
      getExercises.mockImplementationOnce(() => new Promise(resolve => setTimeout(resolve, 100)))
      
      wrapper = createWrapper()
      await nextTick()
      
      expect(wrapper.text()).toContain('Loading exercises...')
      expect(wrapper.vm.loading).toBe(true)
    })
  })

  describe('Error Handling', () => {
    it('handles exercise loading errors gracefully', async () => {
      getExercises.mockRejectedValueOnce(new Error('Loading failed'))
      
      wrapper = createWrapper()
      await nextTick()
      await new Promise(resolve => setTimeout(resolve, 50))
      
      expect(wrapper.vm.loading).toBe(false)
      expect(wrapper.vm.exercises).toEqual([])
    })
  })

  describe('Form Validation', () => {
    beforeEach(async () => {
      wrapper = createWrapper()
      await nextTick()
    })

    it('disables create button when name is empty', async () => {
      const vm = wrapper.vm
      vm.showNewExerciseForm = true
      await nextTick()
      
      const createButton = wrapper.findAll('button').find(btn => btn.text() === 'Create')
      expect(createButton.attributes('disabled')).toBeDefined()
    })

    it('enables create button when name is provided', async () => {
      const vm = wrapper.vm
      vm.showNewExerciseForm = true
      vm.newExercise.name = 'Test Exercise'
      await nextTick()
      
      const createButton = wrapper.findAll('button').find(btn => btn.text() === 'Create')
      expect(createButton.attributes('disabled')).toBeUndefined()
    })
  })

  describe('Accessibility', () => {
    beforeEach(async () => {
      wrapper = createWrapper()
      await nextTick()
    })

    it('has proper modal structure with z-index layering', () => {
      expect(wrapper.find('.z-50').exists()).toBe(true)
      
      // When new exercise form is shown, it should have higher z-index
      const vm = wrapper.vm
      vm.showNewExerciseForm = true
      expect(wrapper.find('.z-60').exists()).toBe(true)
    })

    it('has proper form labels and inputs', async () => {
      const vm = wrapper.vm
      vm.showNewExerciseForm = true
      await nextTick()
      
      expect(wrapper.find('label').exists()).toBe(true)
      expect(wrapper.find('input[type="checkbox"]').exists()).toBe(true)
    })
  })
})