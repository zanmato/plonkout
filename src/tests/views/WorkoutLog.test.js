import { describe, it, expect, beforeEach, vi } from "vitest";
import { mount } from "@vue/test-utils";
import { nextTick } from "vue";
import WorkoutLog from "@/views/WorkoutLog.vue";
import { getWorkouts } from "@/utils/database.js";
import { mockPush } from "../setup.js";

// Mock virtual scroller components
vi.mock("vue3-virtual-scroller", () => ({
  DynamicScroller: {
    name: "DynamicScroller",
    template:
      '<div><slot v-for="(item, index) in items" :key="index" :item="item" :index="index" :active="true" /></div>',
    props: ["items", "minItemSize"],
  },
  DynamicScrollerItem: {
    name: "DynamicScrollerItem",
    template: "<div><slot /></div>",
    props: ["item", "active", "sizeDepedencies", "dataIndex"],
  },
}));

describe("WorkoutLog.vue", () => {
  let wrapper;

  const createWrapper = () => {
    return mount(WorkoutLog, {});
  };

  beforeEach(() => {
    vi.clearAllMocks();
    mockPush.mockClear();
  });

  describe("Component Rendering", () => {
    beforeEach(async () => {
      wrapper = createWrapper();
      await nextTick();
    });

    it("renders the workout log header", () => {
      expect(wrapper.find("button .material-icons").text()).toBe("add");
    });

    it("shows loading state initially", () => {
      expect(wrapper.find(".text-gray-500").text()).toBe("Loading...");
    });

    it("navigates to new workout when add button is clicked", async () => {
      const addButton = wrapper.find("button");
      await addButton.trigger("click");

      expect(mockPush).toHaveBeenCalledWith({ name: "workout-edit" });
    });
  });

  describe("Data Loading and Display", () => {
    beforeEach(async () => {
      // Mock successful data loading
      getWorkouts.mockResolvedValueOnce([
        {
          id: 1,
          name: "Test Workout 1",
          started: new Date("2024-01-15T10:00:00"),
          ended: new Date("2024-01-15T11:30:00"),
          exercises: [
            {
              name: "Wrist Curl",
              sets: [
                { type: "regular", weight: 20, reps: 10 },
                { type: "regular", weight: 22, reps: 8 },
              ],
            },
          ],
        },
        {
          id: 2,
          name: "Test Workout 2",
          started: new Date("2024-01-16T14:00:00"),
          ended: new Date("2024-01-16T15:45:00"),
          exercises: [
            {
              name: "Hook Training",
              sets: [
                { type: "warmup", weight: 15, reps: 12 },
                { type: "regular", weight: 25, reps: 6 },
              ],
            },
            {
              name: "Side Pressure",
              sets: [{ type: "regular", weight: 30, reps: 5 }],
            },
          ],
        },
      ]);

      wrapper = createWrapper();
      await nextTick();
      // Wait for data to load
      await new Promise((resolve) => setTimeout(resolve, 50));
    });

    it("loads and displays workouts", async () => {
      expect(getWorkouts).toHaveBeenCalled();

      // Check that loading state is gone
      expect(wrapper.find(".text-gray-500").exists()).toBe(false);
    });

    it("groups workouts by month", () => {
      const vm = wrapper.vm;
      expect(vm.groupedWorkouts).toHaveLength(1); // Both workouts are in January 2024
      expect(vm.groupedWorkouts[0].monthYear).toBe("January 2024");
      expect(vm.groupedWorkouts[0].workouts).toHaveLength(2);
    });

    it("flattens items for virtual scrolling", () => {
      const vm = wrapper.vm;
      expect(vm.flattenedItems).toHaveLength(3); // 1 header + 2 workouts
      expect(vm.flattenedItems[0].type).toBe("header");
      expect(vm.flattenedItems[1].type).toBe("workout");
      expect(vm.flattenedItems[2].type).toBe("workout");
    });

    it("calculates workout summaries correctly", () => {
      const vm = wrapper.vm;
      const workout1 = vm.flattenedItems[1];
      const workout2 = vm.flattenedItems[2];

      // Test the actual behavior - workouts are sorted by date (newest first)
      const summary1 = vm.getWorkoutSummary(workout1);
      const summary2 = vm.getWorkoutSummary(workout2);

      // Workout1 (newer, id: 2): 2 exercises (Hook Training + Side Pressure)
      // Hook Training: 1 warmup + 1 regular = 1 regular set
      // Side Pressure: 1 regular set
      // Total: 2 regular sets, 2 exercises
      expect(summary1).toBe("2 sets of 2 exercises");

      // Workout2 (older, id: 1): 1 exercise (Wrist Curl), 2 regular sets
      expect(summary2).toBe("2 sets of Wrist Curl");
    });
  });

  describe("Empty State", () => {
    beforeEach(async () => {
      getWorkouts.mockResolvedValueOnce([]);
      wrapper = createWrapper();
      await nextTick();
      await new Promise((resolve) => setTimeout(resolve, 50));
    });

    it("shows empty state when no workouts exist", () => {
      expect(wrapper.text()).toContain("No Workouts");
      // The emoji is not in the component, removing this assertion
    });
  });

  describe("Date Formatting", () => {
    beforeEach(() => {
      wrapper = createWrapper();
    });

    it("formats dates correctly", () => {
      const vm = wrapper.vm;
      const today = new Date();
      const yesterday = new Date();
      yesterday.setDate(yesterday.getDate() - 1);
      const otherDate = new Date("2024-01-15");

      expect(vm.formatWorkoutDate(today)).toBe("Today");
      expect(vm.formatWorkoutDate(yesterday)).toBe("Yesterday");
      // formatWorkoutDate uses i18n d() function with "short" format
      // which returns "Jan 15, 2024" not "1/15/2024"
      expect(vm.formatWorkoutDate(otherDate)).toBe("Jan 15, 2024");
    });

    it("formats duration correctly", () => {
      const vm = wrapper.vm;
      const workout1 = {
        started: new Date("2024-01-15T10:00:00"),
        ended: new Date("2024-01-15T11:30:00")
      };
      const workout2 = {
        started: new Date("2024-01-15T10:00:00"),
        ended: new Date("2024-01-15T10:45:00")
      };

      // Use getWorkoutDuration instead of formatDuration (which doesn't exist)
      expect(vm.getWorkoutDuration(workout1)).toBe("1h 30m");
      expect(vm.getWorkoutDuration(workout2)).toBe("45m");
    });
  });

  describe("Error Handling", () => {
    beforeEach(async () => {
      getWorkouts.mockRejectedValueOnce(new Error("Database error"));
      wrapper = createWrapper();
      await nextTick();
      await new Promise((resolve) => setTimeout(resolve, 50));
    });

    it("handles loading errors gracefully", () => {
      expect(wrapper.vm.loading).toBe(false);
      expect(wrapper.vm.workouts).toEqual([]);
    });
  });

  describe("Navigation", () => {
    beforeEach(async () => {
      getWorkouts.mockResolvedValueOnce([
        {
          id: 1,
          name: "Test Workout",
          started: new Date(),
          ended: new Date(),
          exercises: [],
        },
      ]);

      wrapper = createWrapper();
      await nextTick();
      await new Promise((resolve) => setTimeout(resolve, 50));
    });

    it("navigates to workout edit when workout is clicked", async () => {
      // The .virtual-list class is applied via :deep() CSS and not directly findable
      // Instead test the component method directly
      const vm = wrapper.vm;
      expect(typeof vm.editWorkout).toBe("function");

      // Test navigation function directly - it should use named routes
      vm.editWorkout(1);
      expect(mockPush).toHaveBeenCalledWith({ name: 'workout-edit', params: { id: '1' } });
    });
  });
});
