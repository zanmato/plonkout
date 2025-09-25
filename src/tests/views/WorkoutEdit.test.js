import { describe, it, expect, beforeEach, vi } from "vitest";
import { mount } from "@vue/test-utils";
import { nextTick } from "vue";
import WorkoutEdit from "@/views/WorkoutEdit.vue";
import {
  getWorkout,
  saveWorkout as saveWorkoutToDB,
  deleteWorkout,
  getSetting,
} from "@/utils/database.js";
import { mockPush, mockReplace } from "../setup.js";

// Mock useToast composable
const mockShowSuccess = vi.fn();
const mockShowError = vi.fn();
const mockShowInfo = vi.fn();

vi.mock("@/composables/useToast.js", () => ({
  useToast: () => ({
    showSuccess: mockShowSuccess,
    showError: mockShowError,
    showInfo: mockShowInfo,
  }),
}));

// Mock all the necessary components
vi.mock("@/components/NeoButton.vue", () => ({
  default: {
    name: "NeoButton",
    template: '<button @click="$emit(\'click\')"><slot><slot name="icon" /></slot></button>',
    props: ["variant", "size", "fullWidth"],
    emits: ["click"],
  },
}));

vi.mock("@/components/NeoPanel.vue", () => ({
  default: {
    name: "NeoPanel",
    template: "<div><slot /></div>",
    props: ["padding", "class"],
  },
}));

vi.mock("@/components/NeoHeader.vue", () => ({
  default: {
    name: "NeoHeader",
    template: '<div><slot name="left" /><slot name="middle" /><slot name="right" /></div>',
    props: ["title"],
  },
}));

vi.mock("@/components/DestructiveButton.vue", () => ({
  default: {
    name: "DestructiveButton",
    template: '<button @click="$emit(\'confirm\')"><slot><slot name="icon" /></slot></button>',
    props: ["confirmText", "fullWidth", "size", "iconOnly"],
    emits: ["confirm"],
  },
}));

vi.mock("@/components/StrengthSetEditor.vue", () => ({
  default: {
    name: "StrengthSetEditor",
    template: "<div>StrengthSetEditor</div>",
    props: ["set", "exercise", "setNumber", "weightUnit", "isWeightRecord", "isRepRecord", "previousReps", "maxPercentage", "rpeOptions"],
    emits: ["toggleSetType", "update:weight", "update:reps", "update:time", "update:rpe", "update:arm", "update:notes"],
  },
}));

vi.mock("@/components/CardioSetEditor.vue", () => ({
  default: {
    name: "CardioSetEditor",
    template: "<div>CardioSetEditor</div>",
    props: ["set", "exercise", "setNumber", "distanceUnit", "rpeOptions"],
    emits: ["toggleSetType", "update:distance", "update:time", "update:rpe", "update:notes"],
  },
}));

describe("WorkoutEdit.vue", () => {
  let wrapper;

  const createWrapper = (props = {}) => {
    return mount(WorkoutEdit, {
      props,
      global: {
        stubs: {
          ExerciseSelector: true,
        },
      },
    });
  };

  beforeEach(() => {
    vi.clearAllMocks();
    mockPush.mockClear();
    mockReplace.mockClear();

    // Mock getSetting to return default values
    getSetting.mockImplementation((key, defaultValue) => {
      const settings = {
        weightUnit: "kg",
        distanceUnit: "km",
        exerciseDisplay: "reps",
      };
      return Promise.resolve(settings[key] || defaultValue);
    });
  });

  describe("New Workout", () => {
    beforeEach(() => {
      wrapper = createWrapper();
    });

    it("renders new workout form", () => {
      expect(wrapper.find("h1").text()).toBe("New Workout");
      expect(wrapper.find(".workout-edit").exists()).toBe(true);
    });

    it("initializes with empty workout data", () => {
      const workoutNameInput = wrapper.find('input[type="text"]');
      expect(workoutNameInput.element.value).toBe("");
    });

    it("can add new exercise", async () => {
      // Find button with "Add Exercise" functionality
      const addExerciseButton = wrapper.find(
        '[data-testid="add-exercise-button"]'
      );

      if (addExerciseButton) {
        await addExerciseButton.trigger("click");
        await nextTick();
        expect(wrapper.vm.showExerciseSelector).toBe(true);
      }
    });

    it("saves new workout successfully", async () => {
      saveWorkoutToDB.mockResolvedValueOnce(123);

      // Set workout data
      await wrapper.find('input[type="text"]').setValue("Test Workout");

      // Find and click save button
      const saveButton = wrapper
        .findAll("button")
        .find((btn) => btn.text().includes("workout.save"));

      if (saveButton) {
        await saveButton.trigger("click");
        await nextTick();

        expect(saveWorkoutToDB).toHaveBeenCalled();
        expect(mockShowSuccess).toHaveBeenCalledWith("Workout saved successfully");
        expect(mockReplace).toHaveBeenCalledWith({ name: 'workout-edit', params: { id: '123' } });
      }
    });
  });

  describe("Edit Existing Workout", () => {
    beforeEach(async () => {
      wrapper = createWrapper({ id: "1" });
      await nextTick();
    });

    it("loads existing workout data", async () => {
      expect(getWorkout).toHaveBeenCalledWith(1);
      // Wait for the component to load the data
      await nextTick();
    });

    it("renders edit workout form", () => {
      expect(wrapper.find("h1").text()).toBe("Edit Workout");
    });

    it("can duplicate workout", async () => {
      saveWorkoutToDB.mockResolvedValueOnce(456);

      // Find context menu button (three dots)
      const contextMenuBtn = wrapper
        .findAll("button")
        .find((btn) => btn.find("svg")?.exists());

      if (contextMenuBtn) {
        await contextMenuBtn.trigger("click");
        await nextTick();

        // Find duplicate button in context menu
        const duplicateBtn = wrapper
          .findAll("button")
          .find((btn) => btn.text().includes("workout.duplicate"));

        if (duplicateBtn) {
          await duplicateBtn.trigger("click");
          await nextTick();

          expect(saveWorkoutToDB).toHaveBeenCalled();
          expect(mockPush).toHaveBeenCalledWith({ name: 'workout-edit', params: { id: '456' } });
        }
      }
    });

    it("preserves workout ID when saving existing workout", async () => {
      // Mock existing workout data
      getWorkout.mockResolvedValueOnce({
        id: 1,
        name: "Test Workout",
        started: new Date(),
        ended: null,
        notes: "",
        exercises: [],
      });

      saveWorkoutToDB.mockResolvedValueOnce(1); // Should return the same ID

      // Create wrapper for existing workout
      wrapper = createWrapper({ id: "1" });
      await nextTick();
      await new Promise((resolve) => setTimeout(resolve, 50)); // Wait for loading

      // Trigger save
      const vm = wrapper.vm;
      await vm.saveWorkout();

      // Verify that saveWorkoutToDB was called with the workout including ID
      expect(saveWorkoutToDB).toHaveBeenCalledWith(
        expect.objectContaining({
          id: 1,
          name: "Test Workout",
        })
      );
    });

    it("can delete workout", async () => {
      global.confirm.mockReturnValueOnce(true);

      // Find context menu button
      const contextMenuBtn = wrapper
        .findAll("button")
        .find((btn) => btn.find("svg")?.exists());

      if (contextMenuBtn) {
        await contextMenuBtn.trigger("click");
        await nextTick();

        // Find delete button in context menu
        const deleteBtn = wrapper
          .findAll("button")
          .find((btn) => btn.text().includes("workout.delete"));

        if (deleteBtn) {
          await deleteBtn.trigger("click");
          await nextTick();

          expect(deleteWorkout).toHaveBeenCalledWith(1);
          expect(mockPush).toHaveBeenCalledWith({ name: 'log' });
        }
      }
    });
  });

  describe("Exercise Management", () => {
    beforeEach(() => {
      wrapper = createWrapper();
    });

    it("can add sets to exercises", async () => {
      // First add an exercise
      const mockExercise = {
        name: "Wrist Curl",
        muscleGroup: "Wrist",
        singleArm: true,
      };
      wrapper.vm.addExercise(mockExercise);
      await nextTick();

      expect(wrapper.vm.workout.exercises).toHaveLength(1);
      expect(wrapper.vm.workout.exercises[0].sets).toHaveLength(1);

      // Add another set
      wrapper.vm.addSet(0);
      await nextTick();

      expect(wrapper.vm.workout.exercises[0].sets).toHaveLength(2);
    });

    it("can remove exercises", async () => {
      global.confirm.mockReturnValueOnce(true);

      // Add an exercise first
      const mockExercise = {
        name: "Wrist Curl",
        muscleGroup: "Wrist",
        singleArm: true,
      };
      wrapper.vm.addExercise(mockExercise);
      await nextTick();

      expect(wrapper.vm.workout.exercises).toHaveLength(1);

      // Remove the exercise
      wrapper.vm.removeExercise(0);
      await nextTick();

      expect(wrapper.vm.workout.exercises).toHaveLength(0);
    });

    it("can toggle set type between regular and warmup", async () => {
      // Add an exercise with a set
      const mockExercise = {
        name: "Wrist Curl",
        muscleGroup: "Wrist",
        singleArm: true,
      };
      wrapper.vm.addExercise(mockExercise);
      await nextTick();

      const initialType = wrapper.vm.workout.exercises[0].sets[0].type;
      expect(initialType).toBe("regular");

      // Toggle set type
      wrapper.vm.toggleSetType(0, 0);
      await nextTick();

      expect(wrapper.vm.workout.exercises[0].sets[0].type).toBe("warmup");

      // Toggle back
      wrapper.vm.toggleSetType(0, 0);
      await nextTick();

      expect(wrapper.vm.workout.exercises[0].sets[0].type).toBe("regular");
    });
  });

  describe("Form Validation and Error Handling", () => {
    beforeEach(() => {
      wrapper = createWrapper();
    });

    it("handles save errors gracefully", async () => {
      saveWorkoutToDB.mockRejectedValueOnce(new Error("Save failed"));

      const saveButton = wrapper
        .findAll("button")
        .find((btn) => btn.text().includes("workout.save"));

      if (saveButton) {
        await saveButton.trigger("click");
        await nextTick();

        expect(mockShowError).toHaveBeenCalledWith("Error saving workout");
      }
    });

    it("formats datetime correctly for inputs", () => {
      const testDate = new Date("2024-01-15T10:30:00");
      const formatted = wrapper.vm.formatDatetimeLocal(testDate);
      expect(formatted).toMatch(/2024-01-15T\d{2}:30/);
    });

    it("generates correct set numbers", () => {
      const sets = [
        { type: "warmup" },
        { type: "warmup" },
        { type: "regular" },
        { type: "regular" },
        { type: "warmup" },
      ];

      const mockExercise = { singleArm: false };
      expect(wrapper.vm.getSetNumber(sets, 0, mockExercise, sets[0])).toBe("W1");
      expect(wrapper.vm.getSetNumber(sets, 1, mockExercise, sets[1])).toBe("W2");
      expect(wrapper.vm.getSetNumber(sets, 2, mockExercise, sets[2])).toBe("1");
      expect(wrapper.vm.getSetNumber(sets, 3, mockExercise, sets[3])).toBe("2");
      expect(wrapper.vm.getSetNumber(sets, 4, mockExercise, sets[4])).toBe("W3");
    });
  });

  describe("Navigation", () => {
    beforeEach(() => {
      wrapper = createWrapper();
    });

    it("navigates back when back button is clicked", async () => {
      const backButton = wrapper.find('[data-testid="back-button"]');

      if (backButton) {
        await backButton.trigger("click");
        expect(mockPush).toHaveBeenCalledWith({ name: 'log' });
      }
    });
  });

  describe("Exercise Display Settings", () => {
    beforeEach(async () => {
      wrapper = createWrapper();
      await nextTick();
      // Wait for settings to load
      await new Promise((resolve) => setTimeout(resolve, 50));
    });

    it("loads weight and distance unit settings on mount", () => {
      expect(getSetting).toHaveBeenCalledWith("weightUnit", "kg");
      expect(getSetting).toHaveBeenCalledWith("distanceUnit", "km");
      expect(wrapper.vm.weightUnit).toBe("kg");
      expect(wrapper.vm.distanceUnit).toBe("km");
    });

    it("uses StrengthSetEditor for strength exercises", async () => {
      // Add a strength exercise
      const mockExercise = {
        name: "Wrist Curl",
        muscleGroup: "Wrist",
        singleArm: true,
        type: "strength",
        displayType: "reps"
      };
      wrapper.vm.addExercise(mockExercise);
      await nextTick();

      // Check that StrengthSetEditor component is used
      expect(wrapper.findComponent({ name: "StrengthSetEditor" }).exists()).toBe(true);
      expect(wrapper.findComponent({ name: "CardioSetEditor" }).exists()).toBe(false);
    });

    it("uses CardioSetEditor for cardio exercises", async () => {
      // Add a cardio exercise
      const mockExercise = {
        name: "Running",
        muscleGroup: "Legs",
        singleArm: false,
        type: "cardio",
        displayType: "time"
      };
      wrapper.vm.addExercise(mockExercise);
      await nextTick();

      // Check that CardioSetEditor component is used
      expect(wrapper.findComponent({ name: "CardioSetEditor" }).exists()).toBe(true);
      expect(wrapper.findComponent({ name: "StrengthSetEditor" }).exists()).toBe(false);
    });

    it("creates new sets with all required fields", async () => {
      // Add an exercise
      const mockExercise = {
        name: "Wrist Curl",
        muscleGroup: "Wrist",
        singleArm: true,
        type: "strength",
        displayType: "reps"
      };
      wrapper.vm.addExercise(mockExercise);
      await nextTick();

      // Check that the set has all required fields
      const newSet = wrapper.vm.workout.exercises[0].sets[0];
      expect(newSet).toHaveProperty("type", "regular");
      expect(newSet).toHaveProperty("weight", null);
      expect(newSet).toHaveProperty("distance", null);
      expect(newSet).toHaveProperty("reps", null);
      expect(newSet).toHaveProperty("time", "");
      expect(newSet).toHaveProperty("rpe", null);
      expect(newSet).toHaveProperty("arm", "");
      expect(newSet).toHaveProperty("notes", "");
    });
  });
});
