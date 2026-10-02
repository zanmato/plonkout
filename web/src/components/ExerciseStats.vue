<template>
  <div
    v-if="isOpen"
    class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50 overscroll-none"
    @click="closeModal"
  >
    <NeoPanel
      class="w-full max-w-lg max-h-[90vh] overflow-y-auto overscroll-contain"
      @click.stop
    >
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
          <h4 class="text-lg font-semibold text-black dark:text-white">
            {{ t("exercise.stats.progressChart") }}
          </h4>
          <div class="text-xs text-black dark:text-white opacity-70 mb-3">
            {{
              exercise.type === "cardio"
                ? t("exercise.stats.longestHint")
                : t("exercise.stats.topSetHint")
            }}
          </div>
          <div v-if="ranges.length > 1" class="flex gap-2 mb-3">
            <button
              v-for="r in ranges"
              :key="r"
              type="button"
              class="flex-1 py-1 text-xs font-bold border-2 border-nb-border rounded-md"
              :class="
                r === activeRange
                  ? 'bg-purple-500 text-white'
                  : 'bg-nb-overlay text-black dark:bg-zinc-800 dark:text-white'
              "
              @click="setRange(r)"
            >
              {{ t(`exercise.stats.range.${r}`) }}
            </button>
          </div>
          <div class="h-64">
            <canvas ref="chartCanvas"></canvas>
          </div>
          <div
            v-if="ranges.length > 1"
            class="text-xs text-center text-black dark:text-white opacity-60 mt-2"
          >
            {{ t("exercise.stats.panHint") }}
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
import zoomPlugin from "chartjs-plugin-zoom";
import { getWorkouts } from "@/api/data";
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
type Progress = {
  series: Record<SeriesKey, TimePoint[]>;
  first: number;
  last: number;
};
type RangeKey = "1m" | "3m" | "6m" | "1y" | "all";

const DAY = 24 * 60 * 60 * 1000;
const RANGE_DAYS: Record<Exclude<RangeKey, "all">, number> = {
  "1m": 30,
  "3m": 91,
  "6m": 182,
  "1y": 365,
};
type RecentSet = WorkoutSet & {
  intensity: Intensity | null;
  date: DateLike;
  index: number;
};

const props = withDefaults(
  defineProps<{
    /** An exercise list entry or a logged exercise, only these fields are read */
    exercise: Pick<Exercise, "name" | "type">;
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
const activeRange = ref<RangeKey | null>(null);
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

// The best working set of each workout, one series per intensity tag so heavy
// and light days can be compared. Every set on its own zigzags within a day.
const progress = computed<Progress | null>(() => {
  const series: Record<SeriesKey, TimePoint[]> = {
    heavy: [],
    light: [],
    untagged: [],
  };
  let first = Infinity;
  let last = -Infinity;

  [...chartData.value]
    .sort((a, b) => new Date(a.started).getTime() - new Date(b.started).getTime())
    .forEach((workout) => {
      const x = new Date(workout.started);
      const best: Partial<Record<SeriesKey, number>> = {};
      workout.exercises
        .filter((ex) => ex.name === props.exercise.name)
        .forEach((ex) => {
          const key: SeriesKey = ex.intensity ?? "untagged";
          ex.sets.filter(isWorkingSet).forEach((set) => {
            const y =
              props.exercise.type === "cardio"
                ? set.distance
                : set.weight && set.reps
                  ? set.weight
                  : null;
            if (y && y > (best[key] ?? 0)) best[key] = y;
          });
        });
      for (const [key, y] of Object.entries(best) as [SeriesKey, number][]) {
        series[key].push({ x, y });
        first = Math.min(first, x.getTime());
        last = Math.max(last, x.getTime());
      }
    });

  return Number.isFinite(first) ? { series, first, last } : null;
});

// Only offer ranges shorter than the logged history
const ranges = computed<RangeKey[]>(() => {
  const p = progress.value;
  if (!p) return [];
  const span = p.last - p.first;
  return [
    ...(Object.keys(RANGE_DAYS) as Exclude<RangeKey, "all">[]).filter(
      (r) => RANGE_DAYS[r] * DAY < span,
    ),
    "all",
  ];
});

// Pad the ends by a day so the first and last points are not cut in half
function bounds(p: Progress) {
  return { min: p.first - DAY, max: p.last + DAY };
}

function rangeWindow(p: Progress, range: RangeKey) {
  const { min, max } = bounds(p);
  return range === "all"
    ? { min, max }
    : { min: max - RANGE_DAYS[range] * DAY, max };
}

function setRange(range: RangeKey) {
  const p = progress.value;
  if (!p || !chart.value) return;
  activeRange.value = range;
  chart.value.zoomScale("x", rangeWindow(p, range), "default");
}

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

  const p = progress.value;
  if (!p) return;

  const SERIES: Record<SeriesKey, { label: string; color: string }> = {
    heavy: { label: t("exercise.stats.heavy"), color: "#ef4444" },
    light: { label: t("exercise.stats.light"), color: "#0ea5e9" },
    untagged: { label: t("exercise.stats.untagged"), color: "#a855f7" },
  };

  const datasets: ChartDataset<"line", TimePoint[]>[] = (
    Object.entries(p.series) as [SeriesKey, TimePoint[]][]
  )
    .filter(([, data]) => data.length > 0)
    .map(([key, data]) => ({
      label: SERIES[key].label,
      data,
      borderColor: SERIES[key].color,
      backgroundColor: SERIES[key].color + "1a",
      pointBackgroundColor: SERIES[key].color,
      pointRadius: 3,
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

  activeRange.value = ranges.value.includes("3m") ? "3m" : "all";
  const initial = rangeWindow(p, activeRange.value);
  const limits = bounds(p);

  chart.value = new Chart<"line", TimePoint[]>(ctx!, {
    type: "line",
    data: {
      datasets,
    },
    plugins: [zoomPlugin],
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: {
          display: datasets.length > 1,
          labels: { color: tickColor },
        },
        zoom: {
          limits: { x: { ...limits, minRange: 7 * DAY } },
          pan: { enabled: true, mode: "x" },
          zoom: {
            pinch: { enabled: true },
            wheel: { enabled: true, modifierKey: "ctrl" },
            mode: "x",
            onZoomComplete: () => {
              activeRange.value = null;
            },
          },
        },
      },
      scales: {
        x: {
          type: "time",
          min: initial.min,
          max: initial.max,
          time: {
            tooltipFormat: "PP",
            displayFormats: {
              day: "MMM d",
              week: "MMM d",
              month: "MMM yy",
            },
          },
          ticks: {
            color: tickColor,
            maxRotation: 0,
            autoSkipPadding: 12,
          },
          grid: {
            color: gridColor,
          },
        },
        y: {
          grace: "5%",
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

// Keep the page behind the modal still, panning the chart would drag it along
function lockScroll(locked: boolean) {
  const overflow = locked ? "hidden" : "";
  document.documentElement.style.overflow = overflow;
  document.body.style.overflow = overflow;
}

// Watch for modal open/close
watch(
  () => props.isOpen,
  async (isOpen) => {
    lockScroll(isOpen);
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
  lockScroll(false);
  if (chart.value) {
    chart.value.destroy();
  }
});
</script>
