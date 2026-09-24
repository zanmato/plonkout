<template>
  <div class="workout-log h-full flex flex-col">
    <!-- Header -->
    <NeoHeader :title="t('workout.title')">
      <template #right>
        <NeoButton
          variant="primary"
          size="sm"
          class="rounded-full w-10 h-10 !px-0 !py-0"
          @click="addWorkout"
        >
          <template #icon>
            <span class="material-icons">add</span>
          </template>
        </NeoButton>
      </template>
    </NeoHeader>

    <!-- Content -->
    <div class="flex-1 overflow-hidden mt-4">
      <div v-if="loading" class="flex items-center justify-center h-32">
        <div class="text-gray-500">{{ t("workout.loading") }}</div>
      </div>

      <NeoPanel
        v-else-if="flattenedItems.length === 0"
        class="text-center mx-4"
      >
        <div class="text-base font-bold text-black dark:text-white mb-2">
          {{ t("workout.noWorkouts") }}
        </div>
      </NeoPanel>

      <DynamicScroller
        v-else
        :items="flattenedItems"
        :min-item-size="70"
        class="h-full pb-4"
      >
        <template #default="{ item, index, active }">
          <DynamicScrollerItem
            :item="item"
            :active="active"
            :size-dependencies="[item.type]"
            :data-index="index"
            class="px-4"
          >
            <!-- Month header -->
            <div v-if="item.type === 'header'" class="mb-5">
              <!-- Clickable month header -->
              <div
                class="flex justify-between items-center bg-nb-overlay p-4 border-3 border-nb-border rounded-xl shadow-brutal dark:bg-zinc-600 cursor-pointer transition-all duration-200 hover:shadow-none hover:translate-x-1 hover:translate-y-1 active:shadow-none active:translate-x-1 active:translate-y-1"
                @click="toggleMonthExpansion(item.monthYear)"
              >
                <div class="flex items-center space-x-2">
                  <div class="text-base font-bold text-black dark:text-white">
                    {{ item.monthYear }}
                  </div>
                  <span
                    class="material-icons text-black dark:text-white transition-transform duration-200"
                    :class="{ 'rotate-180': isMonthExpanded(item.monthYear) }"
                  >
                    expand_more
                  </span>
                </div>
                <div
                  class="bg-purple-300 text-black px-3 py-1.5 border-2 border-nb-border rounded-md text-xs font-semibold shadow-brutal-sm"
                >
                  {{ t("workout.workoutCount", { count: item.count }) }}
                </div>
              </div>

              <!-- Calendar Grid -->
              <MonthCalendar
                v-if="isMonthExpanded(item.monthYear)"
                :month-year="item.monthYear"
                :workouts="
                  groupedWorkouts.find((g) => g.monthYear === item.monthYear)
                    ?.workouts || []
                "
              />
            </div>

            <!-- Workout item -->
            <div
              v-else
              class="bg-white border-3 border-black dark:bg-zinc-700 dark:text-white rounded-xl p-5 mb-4 cursor-pointer shadow-brutal transition-all duration-200 hover:shadow-none hover:translate-x-1 hover:translate-y-1 active:shadow-none active:translate-x-1 active:translate-y-1"
              @click="editWorkout(item.id!)"
            >
              <div class="flex justify-between items-start mb-4">
                <div
                  class="bg-purple-300 text-black px-2.5 py-1.5 border-2 border-nb-border rounded-md text-xs font-semibold shadow-brutal-sm"
                >
                  {{ formatWorkoutDate(item.started) }}
                </div>
                <div class="flex items-center gap-2">
                  <div
                    v-if="item.ended"
                    class="bg-nb-overlay text-black px-2.5 py-1.5 border-2 border-nb-border rounded-md text-xs font-semibold shadow-brutal-sm"
                  >
                    {{ getWorkoutDuration(item) }}
                  </div>
                  <button
                    :title="t('workout.duplicate')"
                    :aria-label="t('workout.duplicate')"
                    class="flex items-center justify-center bg-purple-300 text-black w-8 h-8 p-2 border-2 border-nb-border rounded-md shadow-brutal-sm transition-all duration-200 hover:shadow-none hover:translate-x-0.5 hover:translate-y-0.5 active:shadow-none active:translate-x-0.5 active:translate-y-0.5"
                    @click.stop="duplicateWorkout(item)"
                  >
                    <span class="material-icons text-base!">content_copy</span>
                  </button>
                </div>
              </div>
              <div class="text-lg font-bold dark:text-white text-black mb-2">
                {{ item.name }}
              </div>
              <div
                class="text-sm dark:text-white text-black opacity-80 leading-relaxed"
              >
                <div
                  v-for="(exerciseSummary, summaryIndex) in getWorkoutSummary(item)"
                  :key="summaryIndex"
                  class="leading-snug"
                >
                  {{ exerciseSummary }}
                </div>
              </div>
            </div>
          </DynamicScrollerItem>
        </template>
      </DynamicScroller>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useHead } from "@unhead/vue";
import { DynamicScroller, DynamicScrollerItem } from "vue3-virtual-scroller";
import {
  getWorkouts,
  saveWorkout,
  initializeDefaultExercises,
} from "@/utils/database";
import { useToast } from "@/composables/useToast";
import NeoButton from "@/components/NeoButton.vue";
import NeoHeader from "@/components/NeoHeader.vue";
import NeoPanel from "@/components/NeoPanel.vue";
import MonthCalendar from "@/components/MonthCalendar.vue";
import type { DateLike, Id, Workout, WorkoutSet } from "@/types/domain";

interface MonthGroup {
  monthYear: string;
  workouts: Workout[];
  sortDate: Date;
}

interface HeaderItem {
  id: string;
  type: "header";
  monthYear: string;
  count: number;
}

type WorkoutItem = Workout & { type: "workout" };

/** Rows of the virtual scroller, a month header or a workout */
type LogItem = HeaderItem | WorkoutItem;

const router = useRouter();
const { t, d } = useI18n();
const { showError } = useToast();

// Set page title
useHead({
  title: () => t("workout.title"),
});
const workouts = ref<Workout[]>([]);
const loading = ref(true);
const expandedMonths = ref(new Set<string>());

/**
 * Group workouts by month and year
 */
const groupedWorkouts = computed<MonthGroup[]>(() => {
  const groups: Record<string, MonthGroup> = {};

  workouts.value.forEach((workout) => {
    const date = new Date(workout.started);
    const monthYear = d(date, "month");

    let group = groups[monthYear];
    if (!group) {
      group = {
        monthYear,
        workouts: [],
        sortDate: date,
      };
      groups[monthYear] = group;
    } else {
      // Update sortDate to ensure it's the newest date in this group
      if (date > group.sortDate) {
        group.sortDate = date;
      }
    }

    group.workouts.push(workout);
  });

  // Sort groups by date (newest first) and workouts within groups by date (newest first)
  return Object.values(groups)
    .sort((a, b) => b.sortDate.getTime() - a.sortDate.getTime())
    .map((group) => ({
      ...group,
      workouts: group.workouts.sort(
        (a, b) =>
          new Date(b.started).getTime() - new Date(a.started).getTime(),
      ),
    }));
});

/**
 * Flatten grouped workouts for virtual scrolling
 */
const flattenedItems = computed<LogItem[]>(() => {
  const items: LogItem[] = [];

  groupedWorkouts.value.forEach((group) => {
    // Add month header
    items.push({
      id: `header-${group.monthYear}`,
      type: "header",
      monthYear: group.monthYear,
      count: group.workouts.length,
    });

    // Add workouts
    group.workouts.forEach((workout) => {
      items.push({
        ...workout,
        type: "workout",
      });
    });
  });

  return items;
});

/**
 * Load workouts from database
 */
async function loadWorkouts() {
  try {
    loading.value = true;
    const data = await getWorkouts();
    // Sort workouts by started date (newest first)
    workouts.value = data.sort(
      (a, b) => new Date(b.started).getTime() - new Date(a.started).getTime(),
    );
  } catch (error) {
    console.error(t("workout.loadError"), error);
  } finally {
    loading.value = false;
  }
}

/**
 * Toggle month calendar expansion
 */
function toggleMonthExpansion(monthYear: string) {
  if (expandedMonths.value.has(monthYear)) {
    expandedMonths.value.delete(monthYear);
  } else {
    expandedMonths.value.add(monthYear);
  }
}

/**
 * Check if month is expanded
 */
function isMonthExpanded(monthYear: string): boolean {
  return expandedMonths.value.has(monthYear);
}

/**
 * Navigate to add new workout
 */
function addWorkout() {
  router.push({ name: "workout-edit" });
}

/**
 * Navigate to edit existing workout
 */
function editWorkout(id: Id) {
  router.push({ name: "workout-edit", params: { id: id.toString() } });
}

/**
 * Create a fresh, empty set
 */
function createNewSet(): WorkoutSet {
  return {
    type: "regular",
    weight: null,
    distance: null,
    reps: null,
    time: "",
    rpe: null,
    arm: "",
    notes: "",
  };
}

/**
 * Quickly duplicate a workout and open the copy for editing
 */
async function duplicateWorkout(workout: Workout) {
  const duplicatedWorkout: Workout = {
    name: workout.name || "",
    started: new Date(),
    ended: null,
    notes: workout.notes,
    exercises: (workout.exercises || []).map((exercise) => ({
      ...exercise,
      sets: [createNewSet()],
    })),
  };

  try {
    const id = await saveWorkout(duplicatedWorkout);
    router.push({ name: "workout-edit", params: { id: id.toString() } });
  } catch (error) {
    console.error("Error duplicating workout:", error);
    showError(t("workout.duplicateError"));
  }
}

/**
 * Format workout date for display
 */
function formatWorkoutDate(date: DateLike): string {
  const workoutDate = new Date(date);
  const today = new Date();
  const yesterday = new Date(today);
  yesterday.setDate(yesterday.getDate() - 1);

  if (workoutDate.toDateString() === today.toDateString()) {
    return t("common.today");
  } else if (workoutDate.toDateString() === yesterday.toDateString()) {
    return t("common.yesterday");
  } else {
    return d(workoutDate, "short");
  }
}

/**
 * Get workout summary (exercises and sets)
 * @returns Array of exercise summary strings
 */
function getWorkoutSummary(workout: Workout): string[] {
  if (!workout.exercises || workout.exercises.length === 0) {
    return [t("workout.summary.noExercises")];
  }

  // Create a summary for each exercise showing individual set counts or time for cardio
  const exerciseSummaries = workout.exercises
    .map((exercise) => {
      const regularSets = exercise.sets
        ? exercise.sets.filter((set) => set.type !== "warmup")
        : [];

      if (regularSets.length === 0) {
        return null; // Skip exercises with no regular sets
      }

      // Check if this is a cardio exercise (has time instead of reps)
      const isCardio =
        exercise.type === "cardio" ||
        (regularSets.length > 0 && regularSets[0]!.time && !regularSets[0]!.reps);

      if (isCardio) {
        // Calculate total time for cardio exercises
        const totalMinutes = regularSets.reduce((total, set) => {
          if (set.time) {
            // Parse time format (could be "30m", "1h 30m", "90", etc.)
            const timeStr = set.time.toString();
            let minutes: number;

            if (timeStr.includes("h") && timeStr.includes("m")) {
              // Format: "1h 30m"
              const hours = parseInt(timeStr.match(/(\d+)h/)?.[1] || "0");
              const mins = parseInt(timeStr.match(/(\d+)m/)?.[1] || "0");
              minutes = hours * 60 + mins;
            } else if (timeStr.includes("h")) {
              // Format: "1h"
              const hours = parseInt(timeStr.match(/(\d+)h/)?.[1] || "0");
              minutes = hours * 60;
            } else if (timeStr.includes("m")) {
              // Format: "30m"
              minutes = parseInt(timeStr.match(/(\d+)m/)?.[1] || "0");
            } else {
              // Assume it's just minutes as a number
              minutes = parseInt(timeStr) || 0;
            }

            return total + minutes;
          }
          return total;
        }, 0);

        if (totalMinutes > 0) {
          const hours = Math.floor(totalMinutes / 60);
          const mins = totalMinutes % 60;

          let timeDisplay: string;
          if (hours > 0 && mins > 0) {
            timeDisplay = `${hours}h ${mins}m`;
          } else if (hours > 0) {
            timeDisplay = `${hours}h`;
          } else {
            timeDisplay = `${mins}m`;
          }

          return `${timeDisplay} of ${exercise.name}`;
        }
      }

      // For strength exercises, show set count
      return t("workout.summary.singleExercise", {
        totalSets: regularSets.length,
        exerciseName: exercise.name,
      });
    })
    .filter((summary): summary is string => Boolean(summary)); // Remove null entries

  return exerciseSummaries.length > 0
    ? exerciseSummaries
    : [t("workout.summary.noExercises")];
}

/**
 * Get workout duration as localized string
 * @returns Duration string like "3h 30m" or null
 */
function getWorkoutDuration(workout: Workout): string | null {
  if (!workout.started || !workout.ended) {
    return null;
  }

  const start = new Date(workout.started);
  const end = new Date(workout.ended);
  const diffMs = end.getTime() - start.getTime();
  const diffMins = Math.round(diffMs / (1000 * 60));

  if (diffMins <= 0) {
    return null;
  }

  const hours = Math.floor(diffMins / 60);
  const minutes = diffMins % 60;

  if (hours > 0 && minutes > 0) {
    return `${hours}${t("common.duration.hours")} ${minutes}${t(
      "common.duration.minutes",
    )}`;
  } else if (hours > 0) {
    return `${hours}${t("common.duration.hours")}`;
  } else {
    return `${minutes}${t("common.duration.minutes")}`;
  }
}

onMounted(async () => {
  await initializeDefaultExercises();
  await loadWorkouts();
});
</script>

<style scoped>
/* Virtual scroller styling */
:deep(.virtual-list) {
  height: 100%;
}

:deep(.virtual-list-item) {
  padding-bottom: 0.5rem;
}
</style>
