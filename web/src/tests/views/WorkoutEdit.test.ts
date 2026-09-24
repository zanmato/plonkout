import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { flushPromises, mount, type VueWrapper } from "@vue/test-utils";
import { nextTick } from "vue";
import { http, HttpResponse } from "msw";
import WorkoutEdit from "@/views/WorkoutEdit.vue";
import type { Exercise } from "@/types/domain";
import { clearSettingsCache } from "@/api/data";
import { API, server } from "../helpers/msw";
import { backend } from "../mocks/backend";
import { mockPush, mockReplace } from "../setup";

const confirm = vi.mocked(global.confirm);

/** An entry of the exercise list, as the exercise selector hands it over */
const listExercise = (fields: Partial<Exercise> & { name: string }): Exercise => ({
  id: crypto.randomUUID(),
  muscleGroup: "Forearm",
  singleArm: false,
  type: "strength",
  displayType: "reps",
  archived: false,
  created: "2024-01-01T00:00:00.000Z",
  ...fields,
});

// Mock useToast composable
const mockShowSuccess = vi.fn();
const mockShowError = vi.fn();
const mockShowInfo = vi.fn();

vi.mock("@/composables/useToast", () => ({
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
  // any so tests can reach the script setup internals through wrapper.vm
  let wrapper: VueWrapper<any>;

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

  /** Opens the header's context menu and finds one of its buttons */
  const contextMenuButton = async (label: string) => {
    const menuButton = wrapper
      .findAll("button")
      .find((btn) => btn.text().includes("more_vert"));
    expect(menuButton).toBeDefined();
    await menuButton!.trigger("click");
    const button = wrapper.findAll("button").find((btn) => btn.text().includes(label));
    expect(button).toBeDefined();
    return button!;
  };

  beforeEach(() => {
    vi.clearAllMocks();
    mockPush.mockClear();
    mockReplace.mockClear();

    backend.seed({
      settings: {
        weightUnit: "kg",
        distanceUnit: "km",
        exerciseDisplay: "reps",
      },
    });
  });

  afterEach(() => {
    // Unmounting clears a pending auto-save, so it cannot land in a later test
    wrapper?.unmount();
  });

  describe("New Workout", () => {
    beforeEach(async () => {
      wrapper = createWrapper();
      await flushPromises();
    });

    it("renders new workout form", () => {
      expect(wrapper.find("h1").text()).toBe("New Workout");
      expect(wrapper.find(".workout-edit").exists()).toBe(true);
    });

    it("initializes with empty workout data", () => {
      const workoutNameInput = wrapper.find<HTMLInputElement>('input[type="text"]');
      expect(workoutNameInput.element.value).toBe("");
    });

    it("can add new exercise", async () => {
      // Find button with "Add Exercise" functionality
      const addExerciseButton = wrapper.find(
        '[data-testid="add-exercise-button"]'
      );

      expect(addExerciseButton.exists()).toBe(true);
      await addExerciseButton.trigger("click");
      await nextTick();
      expect(wrapper.vm.showExerciseSelector).toBe(true);
    });

    it("saves new workout successfully", async () => {
      // Set workout data
      await wrapper.find('input[type="text"]').setValue("Test Workout");

      // Saving is automatic, run it now instead of waiting for the debounce
      await wrapper.vm.saveWorkout();
      await flushPromises();

      expect(backend.workouts).toHaveLength(1);
      const saved = backend.workouts[0]!;
      expect(saved).toMatchObject({ name: "Test Workout", revision: 1 });
      expect(wrapper.vm.workout.id).toBe(saved.id);
      expect(mockShowError).not.toHaveBeenCalled();
      expect(mockReplace).toHaveBeenCalledWith({ name: 'workout-edit', params: { id: saved.id } });
    });
  });

  describe("Edit Existing Workout", () => {
    let workoutId: string;

    beforeEach(async () => {
      workoutId = backend.seed({
        workouts: [
          {
            name: "Test Workout",
            started: new Date("2024-01-15T10:00:00"),
            ended: null,
            notes: "",
            exercises: [{ name: "Wrist Curl", muscleGroup: "Forearm", sets: [{ weight: 20, reps: 10 }] }],
          },
        ],
      }).workouts[0]!.id;
      wrapper = createWrapper({ id: workoutId });
      // Wait for the component to load the data
      await flushPromises();
    });

    it("loads existing workout data", async () => {
      expect(wrapper.vm.workout.id).toBe(workoutId);
      expect(wrapper.find<HTMLInputElement>('input[type="text"]').element.value).toBe("Test Workout");
      expect(wrapper.vm.workout.exercises[0].name).toBe("Wrist Curl");
    });

    it("renders edit workout form", () => {
      expect(wrapper.find("h1").text()).toBe("Edit Workout");
    });

    it("can duplicate workout", async () => {
      const duplicateBtn = await contextMenuButton("Duplicate Workout");
      await duplicateBtn.trigger("click");
      await flushPromises();

      expect(backend.workouts).toHaveLength(2);
      const copy = backend.workouts.find((w) => w.id !== workoutId)!;
      expect(copy).toMatchObject({ name: "Test Workout", ended: null, revision: 1 });
      expect(copy.exercises.map((e) => e.name)).toEqual(["Wrist Curl"]);
      expect(mockPush).toHaveBeenCalledWith({ name: 'workout-edit', params: { id: copy.id } });
    });

    it("preserves workout ID when saving existing workout", async () => {
      wrapper.vm.workout.notes = "Felt strong";
      await wrapper.vm.saveWorkout();
      await flushPromises();

      // The stored workout was updated in place, not copied
      expect(backend.workouts).toHaveLength(1);
      expect(backend.workout(workoutId)).toMatchObject({
        id: workoutId,
        name: "Test Workout",
        notes: "Felt strong",
        revision: 2,
      });
      expect(wrapper.vm.workout.revision).toBe(2);
      expect(mockReplace).not.toHaveBeenCalled();
    });

    it("saves again after its own save without a conflict", async () => {
      wrapper.vm.workout.notes = "First";
      await wrapper.vm.saveWorkout();
      wrapper.vm.workout.notes = "Second";
      await wrapper.vm.saveWorkout();
      await flushPromises();

      expect(mockShowError).not.toHaveBeenCalled();
      expect(backend.workout(workoutId)).toMatchObject({ notes: "Second", revision: 3 });
    });

    it("reloads the stored workout when another device saved first", async () => {
      // Another device saves a newer revision after this editor loaded
      Object.assign(backend.workout(workoutId), { name: "From phone", revision: 2 });

      wrapper.vm.workout.name = "From laptop";
      await wrapper.vm.saveWorkout();
      await flushPromises();

      expect(mockShowError).toHaveBeenCalledWith(
        "This workout was changed on another device. Reloaded the latest version."
      );
      expect(backend.workout(workoutId)).toMatchObject({ name: "From phone", revision: 2 });
      expect(wrapper.vm.workout).toMatchObject({ name: "From phone", revision: 2 });
    });

    it("can save workout as a template", async () => {
      const templateBtn = await contextMenuButton("Save as template");
      await templateBtn.trigger("click");
      await flushPromises();

      expect(backend.templates).toHaveLength(1);
      expect(backend.templates[0]).toMatchObject({ name: "Test Workout" });
      expect(backend.templates[0]!.exercises.map((e) => e.name)).toEqual(["Wrist Curl"]);
      expect(mockShowSuccess).toHaveBeenCalledWith("Workout template saved");
    });

    it("can delete workout", async () => {
      confirm.mockReturnValueOnce(true);

      const deleteBtn = await contextMenuButton("Delete Workout");
      await deleteBtn.trigger("click");
      await flushPromises();

      expect(backend.workouts).toEqual([]);
      expect(mockPush).toHaveBeenCalledWith({ name: 'log' });
    });
  });

  describe("Exercise Management", () => {
    beforeEach(async () => {
      wrapper = createWrapper();
      await flushPromises();
    });

    it("can add sets to exercises", async () => {
      // First add an exercise
      const mockExercise = listExercise({
        name: "Wrist Curl",
        muscleGroup: "Wrist",
        singleArm: true,
      });
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
      confirm.mockReturnValueOnce(true);

      // Add an exercise first
      const mockExercise = listExercise({
        name: "Wrist Curl",
        muscleGroup: "Wrist",
        singleArm: true,
      });
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
      const mockExercise = listExercise({
        name: "Wrist Curl",
        muscleGroup: "Wrist",
        singleArm: true,
      });
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
    beforeEach(async () => {
      wrapper = createWrapper();
      await flushPromises();
    });

    it("handles save errors gracefully", async () => {
      server.use(
        http.post(`${API}/api/workouts`, () =>
          HttpResponse.json(
            { type: "about:blank", title: "Internal Server Error", status: 500, code: "internal_error" },
            { status: 500, headers: { "Content-Type": "application/problem+json" } }
          )
        )
      );

      await wrapper.vm.saveWorkout();
      await flushPromises();

      expect(mockShowError).toHaveBeenCalledWith("Error saving workout");
      expect(backend.workouts).toEqual([]);
      expect(mockReplace).not.toHaveBeenCalled();
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
    beforeEach(async () => {
      wrapper = createWrapper();
      await flushPromises();
    });

    it("navigates back when back button is clicked", async () => {
      const backButton = wrapper.find('[data-testid="back-button"]');

      expect(backButton.exists()).toBe(true);
      await backButton.trigger("click");
      expect(mockPush).toHaveBeenCalledWith({ name: 'log' });
    });
  });

  describe("Exercise Display Settings", () => {
    beforeEach(async () => {
      wrapper = createWrapper();
      // Wait for settings to load
      await flushPromises();
    });

    it("loads weight and distance unit settings on mount", async () => {
      expect(wrapper.vm.weightUnit).toBe("kg");
      expect(wrapper.vm.distanceUnit).toBe("km");

      // Units other than the defaults show they come from the stored settings
      wrapper.unmount();
      backend.seed({ settings: { weightUnit: "lbs", distanceUnit: "mi" } });
      clearSettingsCache();
      wrapper = createWrapper();
      await flushPromises();

      expect(wrapper.vm.weightUnit).toBe("lbs");
      expect(wrapper.vm.distanceUnit).toBe("mi");
    });

    it("uses StrengthSetEditor for strength exercises", async () => {
      // Add a strength exercise
      const mockExercise = listExercise({
        name: "Wrist Curl",
        muscleGroup: "Wrist",
        singleArm: true,
        type: "strength",
        displayType: "reps"
      });
      wrapper.vm.addExercise(mockExercise);
      await nextTick();

      // Check that StrengthSetEditor component is used
      expect(wrapper.findComponent({ name: "StrengthSetEditor" }).exists()).toBe(true);
      expect(wrapper.findComponent({ name: "CardioSetEditor" }).exists()).toBe(false);
    });

    it("uses CardioSetEditor for cardio exercises", async () => {
      // Add a cardio exercise
      const mockExercise = listExercise({
        name: "Running",
        muscleGroup: "Legs",
        singleArm: false,
        type: "cardio",
        displayType: "time"
      });
      wrapper.vm.addExercise(mockExercise);
      await nextTick();

      // Check that CardioSetEditor component is used
      expect(wrapper.findComponent({ name: "CardioSetEditor" }).exists()).toBe(true);
      expect(wrapper.findComponent({ name: "StrengthSetEditor" }).exists()).toBe(false);
    });

    it("creates new sets with all required fields", async () => {
      // Add an exercise
      const mockExercise = listExercise({
        name: "Wrist Curl",
        muscleGroup: "Wrist",
        singleArm: true,
        type: "strength",
        displayType: "reps"
      });
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
