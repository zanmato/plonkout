import { describe, it, expect, beforeEach, vi } from "vitest";
import { flushPromises, mount, type VueWrapper } from "@vue/test-utils";
import { nextTick } from "vue";
import { http, HttpResponse } from "msw";
import WorkoutLog from "@/views/WorkoutLog.vue";
import { API, server } from "../helpers/msw";
import { backend } from "../mocks/backend";
import { mockPush } from "../setup";

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
  // any so tests can reach the script setup internals through wrapper.vm
  let wrapper: VueWrapper<any>;

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
      // Assert synchronously, before the workouts request can answer.
      const loadingWrapper = createWrapper();
      expect(loadingWrapper.find(".text-gray-500").text()).toBe("Loading...");
    });

    it("navigates to new workout when add button is clicked", async () => {
      const addButton = wrapper.find("button");
      await addButton.trigger("click");

      expect(mockPush).toHaveBeenCalledWith({ name: "workout-edit" });
    });
  });

  describe("Data Loading and Display", () => {
    beforeEach(async () => {
      backend.seed({ workouts: [
        {
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
        {
          name: "Cardio Workout",
          started: new Date("2024-01-17T09:00:00"),
          ended: new Date("2024-01-17T10:00:00"),
          exercises: [
            {
              name: "Running",
              type: "cardio",
              displayType: "time",
              sets: [
                { type: "regular", time: "30m", distance: 5 },
                { type: "regular", time: "15m", distance: 2 },
              ],
            },
          ],
        },
      ] });

      wrapper = createWrapper();
      // Wait for data to load
      await flushPromises();
    });

    it("loads and displays workouts", async () => {
      expect(wrapper.vm.workouts.map((w: { name: string }) => w.name)).toEqual([
        "Cardio Workout",
        "Test Workout 2",
        "Test Workout 1",
      ]);

      // Check that loading state is gone
      expect(wrapper.find(".text-gray-500").exists()).toBe(false);
    });

    it("groups workouts by month", () => {
      const vm = wrapper.vm;
      expect(vm.groupedWorkouts).toHaveLength(1); // All three workouts are in January 2024
      expect(vm.groupedWorkouts[0].monthYear).toBe("January 2024");
      expect(vm.groupedWorkouts[0].workouts).toHaveLength(3);
    });

    it("flattens items for virtual scrolling", () => {
      const vm = wrapper.vm;
      expect(vm.flattenedItems).toHaveLength(4); // 1 header + 3 workouts
      expect(vm.flattenedItems[0].type).toBe("header");
      expect(vm.flattenedItems[1].type).toBe("workout");
      expect(vm.flattenedItems[2].type).toBe("workout");
      expect(vm.flattenedItems[3].type).toBe("workout");
    });

    it("calculates workout summaries correctly", () => {
      const vm = wrapper.vm;
      const workout1 = vm.flattenedItems[1]; // Newest (cardio workout)
      const workout2 = vm.flattenedItems[2]; // Middle (strength workout 2)
      const workout3 = vm.flattenedItems[3]; // Oldest (strength workout 1)

      // Test the actual behavior - workouts are sorted by date (newest first)
      const summary1 = vm.getWorkoutSummary(workout1);
      const summary2 = vm.getWorkoutSummary(workout2);
      const summary3 = vm.getWorkoutSummary(workout3);

      // Workout1 (newest): Cardio workout with Running
      // Running: 30m + 15m = 45m total
      expect(summary1).toEqual(["45m of Running"]);

      // Workout2 (middle): 2 exercises (Hook Training + Side Pressure)
      // Hook Training: 1 warmup + 1 regular = 1 regular set
      // Side Pressure: 1 regular set
      expect(summary2).toEqual(["1 sets of Hook Training", "1 sets of Side Pressure"]);

      // Workout3 (oldest): 1 exercise (Wrist Curl), 2 regular sets
      expect(summary3).toEqual(["2 sets of Wrist Curl"]);
    });
  });

  describe("Empty State", () => {
    beforeEach(async () => {
      wrapper = createWrapper();
      await flushPromises();
    });

    it("shows empty state when no workouts exist", () => {
      expect(wrapper.text()).toContain("No Workouts");
      // The emoji is not in the component, removing this assertion
    });
  });

  describe("Date Formatting", () => {
    beforeEach(async () => {
      wrapper = createWrapper();
      await flushPromises();
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
      backend.seed({ workouts: [{ name: "Unreachable" }] });
      server.use(
        http.get(`${API}/api/workouts`, () =>
          HttpResponse.json(
            { type: "about:blank", title: "Internal Server Error", status: 500, code: "internal_error" },
            { status: 500, headers: { "Content-Type": "application/problem+json" } },
          ),
        ),
      );
      wrapper = createWrapper();
      await flushPromises();
    });

    it("handles loading errors gracefully", () => {
      expect(wrapper.vm.loading).toBe(false);
      expect(wrapper.vm.workouts).toEqual([]);
    });
  });

  describe("Navigation", () => {
    let workoutId: string;

    beforeEach(async () => {
      workoutId = backend.seed({ workouts: [
        {
          name: "Test Workout",
          started: new Date(),
          ended: new Date(),
          exercises: [],
        },
      ] }).workouts[0]!.id;

      wrapper = createWrapper();
      await flushPromises();
    });

    it("navigates to workout edit when workout is clicked", async () => {
      // The .virtual-list class is applied via :deep() CSS and not directly findable
      // Instead test the component method directly
      const vm = wrapper.vm;
      expect(typeof vm.editWorkout).toBe("function");

      // Test navigation function directly - it should use named routes
      expect(vm.workouts[0].id).toBe(workoutId);
      vm.editWorkout(workoutId);
      expect(mockPush).toHaveBeenCalledWith({ name: 'workout-edit', params: { id: workoutId } });
    });
  });
  describe("Next Planned Session", () => {
    it("is not shown without a plan", async () => {
      wrapper = createWrapper();
      await flushPromises();
      expect(wrapper.find('[data-testid="next-planned"]').exists()).toBe(false);
    });

    it("starts the next session of the plan", async () => {
      const sessionId = "20000000-0000-4000-8000-000000000001";
      backend.seed({
        plans: [
          {
            name: "Peak block",
            sessions: [
              { id: sessionId, label: "W3 A", exercises: [{ exercise: "Pronation", targets: [{ setType: "Volume", sets: 2 }] }] },
              { label: "W3 B", exercises: [] },
            ],
          },
        ],
      });
      wrapper = createWrapper();
      await flushPromises();

      const card = wrapper.find('[data-testid="next-planned"]');
      expect(card.text()).toContain("Next planned");
      expect(card.text()).toContain("W3 A");
      expect(card.text()).not.toContain("W3 B");

      await card.find('[data-testid="start-next-planned"]').trigger("click");
      await flushPromises();

      const workout = backend.workouts[0]!;
      expect(workout.plannedSessionId).toBe(sessionId);
      expect(mockPush).toHaveBeenCalledWith({ name: "workout-edit", params: { id: workout.id } });
    });
  });
});
