<template>
  <div
    v-if="isOpen"
    class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50"
    @click="closeModal"
  >
    <NeoPanel class="w-full max-w-lg max-h-[90vh] overflow-y-auto" @click.stop>
      <!-- Header -->
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-xl font-bold text-black dark:text-white">
          {{ t("exercise.stats.title", { exercise: exercise.name }) }}
        </h3>
        <button
          class="w-8 h-8 flex items-center justify-center rounded-full hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
          @click="closeModal"
        >
          <span class="material-icons text-xl">close</span>
        </button>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="flex items-center justify-center h-32">
        <div class="text-gray-500">{{ t("statistics.loading") }}</div>
      </div>

      <!-- No Data State -->
      <div v-else-if="chartData.length === 0" class="text-center py-8">
        <div class="text-base font-bold text-black dark:text-white mb-2">
          {{ t("exercise.stats.noData") }}
        </div>
        <div class="text-sm text-black dark:text-white opacity-70">
          {{ t("exercise.stats.noDataDescription") }}
        </div>
      </div>

      <!-- Stats Content -->
      <div v-else class="space-y-6">
        <!-- Quick Stats -->
        <div class="grid grid-cols-2 gap-4">
          <div
            class="bg-nb-overlay dark:bg-zinc-800 border-2 border-nb-border rounded-lg p-3 text-center"
          >
            <div class="text-lg font-bold text-black dark:text-white">
              {{ totalSets }}
            </div>
            <div class="text-xs text-black dark:text-white opacity-70">
              {{ t("exercise.stats.totalSets") }}
            </div>
          </div>
          <div
            class="bg-nb-overlay dark:bg-zinc-800 border-2 border-nb-border rounded-lg p-3 text-center"
          >
            <div class="text-lg font-bold text-black dark:text-white">
              {{ personalBest }}
            </div>
            <div class="text-xs text-black dark:text-white opacity-70">
              {{ t("exercise.stats.personalBest") }}
            </div>
          </div>
        </div>

        <!-- Progress Chart -->
        <div>
          <h4 class="text-lg font-semibold text-black dark:text-white mb-3">
            {{ t("exercise.stats.progressChart") }}
          </h4>
          <div class="h-64">
            <canvas ref="chartCanvas"></canvas>
          </div>
        </div>

        <!-- Recent Performance -->
        <div v-if="recentSets.length > 0">
          <h4 class="text-lg font-semibold text-black dark:text-white mb-3">
            {{ t("exercise.stats.recentPerformance") }}
          </h4>
          <div class="space-y-2">
            <div
              v-for="set in recentSets.slice(0, 5)"
              :key="`${set.date}-${set.index}`"
              class="flex justify-between items-center py-2 px-3 bg-nb-overlay dark:bg-zinc-800 border border-nb-border rounded"
            >
              <div class="text-sm text-black dark:text-white">
                {{ formatDate(set.date) }}
                <span
                  v-if="set.intensity"
                  class="ml-1 text-xs font-bold uppercase opacity-70"
                >
                  {{ t(`exercise.intensity.${set.intensity}`) }}
                </span>
              </div>
              <div class="text-sm font-semibold text-black dark:text-white">
                <span v-if="exercise.type === 'cardio'">
                  {{ set.time || "0:00" }}
                  <span v-if="set.distance" class="ml-1 opacity-70">
                    ({{ set.distance }}{{ distanceUnit }})
                  </span>
                </span>
                <span v-else>
                  {{ set.weight || 0 }}{{ weightUnit }} × {{ set.reps || 0 }}
                  <span v-if="set.rpe" class="ml-1 opacity-70">
                    RPE {{ String(set.rpe).replace("RPE ", "") }}
                  </span>
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </NeoPanel>
  </div>
</template>

<script setup lang="ts">
import {
  ref,
  computed,
  watch,
  onUnmounted,
  useTemplateRef,
  nextTick,
} from "vue";
import { useI18n } from "vue-i18n";
import { Chart, registerables, type ChartDataset } from "chart.js";
import { getWorkouts } from "@/utils/database";
import { useUnits } from "@/composables/useUnits";
import { isWorkingSet } from "@/utils/exerciseHistory";
import NeoPanel from "@/components/NeoPanel.vue";
import type {
  DateLike,
  Exercise,
  Intensity,
  Workout,
  WorkoutSet,
} from "@/types/domain";
import "chartjs-adapter-date-fns";

Chart.register(...registerables);

type TimePoint = { x: Date; y: number };
type SeriesKey = Intensity | "untagged";
type RecentSet = WorkoutSet & {
  intensity: Intensity | null;
  date: DateLike;
  index: number;
};

const props = withDefaults(
  defineProps<{
    exercise: Exercise;
    isOpen?: boolean;
  }>(),
  {
    isOpen: false,
  },
);

const emit = defineEmits<{ (e: "close"): void }>();

const { t, d } = useI18n();
const chartCanvas = useTemplateRef<HTMLCanvasElement>("chartCanvas");
const loading = ref(false);
const chartData = ref<Workout[]>([]);
const chart = ref<Chart<"line", TimePoint[]> | null>(null);
const { weightUnit, distanceUnit } = useUnits();

// Computed statistics
const totalSets = computed(() => {
  return chartData.value.reduce((total, workout) => {
    return (
      total +
      workout.exercises
        .filter((ex) => ex.name === props.exercise.name)
        .reduce(
          (exTotal, ex) =>
            exTotal + ex.sets.filter((s) => s.type !== "warmup").length,
          0
        )
    );
  }, 0);
});

const personalBest = computed(() => {
  if (props.exercise.type === "cardio") {
    // For cardio, find best time or distance
    let bestDistance = 0;

    chartData.value.forEach((workout) => {
      workout.exercises
        .filter((ex) => ex.name === props.exercise.name)
        .forEach((ex) => {
          ex.sets.forEach((set) => {
            if (set.distance && set.distance > bestDistance) {
              bestDistance = set.distance;
            }
          });
        });
    });

    return bestDistance > 0
      ? `${bestDistance}${distanceUnit.value}`
      : t("exercise.stats.noPB");
  } else {
    // For strength, find heaviest weight
    let maxWeight = 0;
    chartData.value.forEach((workout) => {
      workout.exercises
        .filter((ex) => ex.name === props.exercise.name)
        .forEach((ex) => {
          ex.sets.forEach((set) => {
            if (set.weight && set.weight > maxWeight) {
              maxWeight = set.weight;
            }
          });
        });
    });

    return maxWeight > 0
      ? `${maxWeight}${weightUnit.value}`
      : t("exercise.stats.noPB");
  }
});

const recentSets = computed(() => {
  const sets: RecentSet[] = [];
  [...chartData.value]
    .sort((a, b) => new Date(b.started).getTime() - new Date(a.started).getTime())
    .slice(0, 10) // Last 10 workouts
    .forEach((workout) => {
      workout.exercises
        .filter((ex) => ex.name === props.exercise.name)
        .forEach((ex) => {
          ex.sets
            .filter((s) => s.type !== "warmup")
            .forEach((set, index) => {
              sets.push({
                ...set,
                intensity: ex.intensity || null,
                date: workout.started,
                index,
              });
            });
        });
    });

  return sets.sort(
    (a, b) => new Date(b.date).getTime() - new Date(a.date).getTime(),
  );
});

const formatDate = (dateStr: DateLike) => {
  const date = new Date(dateStr);
  return d(date, "short");
};

const closeModal = () => {
  emit("close");
};

const loadData = async () => {
  loading.value = true;
  try {
    const workouts = await getWorkouts();

    // Filter workouts that contain this exercise
    chartData.value = workouts.filter((workout) =>
      workout.exercises.some((ex) => ex.name === props.exercise.name)
    );
  } catch (error) {
    console.error("Error loading exercise stats:", error);
    chartData.value = [];
  } finally {
    loading.value = false;
  }
};

const createChart = async () => {
  if (!chartCanvas.value || chartData.value.length === 0) return;

  // Destroy existing chart
  if (chart.value) {
    chart.value.destroy();
    chart.value = null;
  }

  await nextTick();

  const ctx = chartCanvas.value.getContext("2d");

  // One series per intensity tag so heavy and light days can be compared
  const SERIES: Record<SeriesKey, { label: string; color: string }> = {
    heavy: { label: t("exercise.stats.heavy"), color: "#ef4444" },
    light: { label: t("exercise.stats.light"), color: "#0ea5e9" },
    untagged: { label: t("exercise.stats.untagged"), color: "#a855f7" },
  };
  const series: Record<SeriesKey, TimePoint[]> = {
    heavy: [],
    light: [],
    untagged: [],
  };

  [...chartData.value]
    .sort((a, b) => new Date(a.started).getTime() - new Date(b.started).getTime())
    .forEach((workout) => {
      workout.exercises
        .filter((ex) => ex.name === props.exercise.name)
        .forEach((ex) => {
          const bucket =
            (ex.intensity ? series[ex.intensity] : undefined) ||
            series.untagged;
          ex.sets.filter(isWorkingSet).forEach((set) => {
            const y =
              props.exercise.type === "cardio"
                ? set.distance
                : set.weight && set.reps
                  ? set.weight
                  : null;
            if (y) bucket.push({ x: new Date(workout.started), y });
          });
        });
    });

  const datasets: ChartDataset<"line", TimePoint[]>[] = (
    Object.entries(series) as [SeriesKey, TimePoint[]][]
  )
    .filter(([, data]) => data.length > 0)
    .map(([key, data]) => ({
      label: SERIES[key].label,
      data,
      borderColor: SERIES[key].color,
      backgroundColor: SERIES[key].color + "1a",
      borderWidth: 2,
      tension: 0.2,
    }));
  if (datasets.length === 0) return;
  // Only fill the area when there is a single series, overlaps get muddy
  datasets[0]!.fill = datasets.length === 1;

  const isDark = document.documentElement.classList.contains("dark");
  const tickColor = isDark ? "#d4d4d8" : "#52525b";
  const gridColor = isDark
    ? "rgba(212, 212, 216, 0.15)"
    : "rgba(82, 82, 91, 0.15)";

  chart.value = new Chart<"line", TimePoint[]>(ctx!, {
    type: "line",
    data: {
      datasets,
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: {
          display: datasets.length > 1,
          labels: { color: tickColor },
        },
      },
      scales: {
        x: {
          type: "time",
          time: {
            unit: "day",
            displayFormats: {
              day: "MMM dd",
            },
          },
          ticks: {
            color: tickColor,
          },
          grid: {
            color: gridColor,
          },
        },
        y: {
          beginAtZero: true,
          ticks: {
            color: tickColor,
            callback: function (value) {
              if (props.exercise.type === "cardio") {
                return value + distanceUnit.value;
              } else {
                return value + weightUnit.value;
              }
            },
          },
          grid: {
            color: gridColor,
          },
        },
      },
    },
  });
};

// Watch for modal open/close
watch(
  () => props.isOpen,
  async (isOpen) => {
    if (isOpen) {
      await nextTick(); // Ensure DOM is updated
      await loadData();
      await nextTick();
      createChart();
    } else {
      if (chart.value) {
        chart.value.destroy();
        chart.value = null;
      }
    }
  },
  { immediate: true }
);

// Watch for data changes
watch(chartData, () => {
  if (props.isOpen) {
    createChart();
  }
});

onUnmounted(() => {
  if (chart.value) {
    chart.value.destroy();
  }
});
</script>
