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
          @click="closeModal"
          class="w-8 h-8 flex items-center justify-center rounded-full hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
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
            class="bg-nb-overlay border-2 border-nb-border rounded-lg p-3 text-center"
          >
            <div class="text-lg font-bold text-black dark:text-white">
              {{ totalSets }}
            </div>
            <div class="text-xs text-black dark:text-white opacity-70">
              {{ t("exercise.stats.totalSets") }}
            </div>
          </div>
          <div
            class="bg-nb-overlay border-2 border-nb-border rounded-lg p-3 text-center"
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
              class="flex justify-between items-center py-2 px-3 bg-nb-overlay border border-nb-border rounded"
            >
              <div class="text-sm text-black dark:text-white">
                {{ formatDate(set.date) }}
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
                    RPE {{ set.rpe.replace("RPE ", "") }}
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

<script setup>
import {
  ref,
  computed,
  watch,
  onUnmounted,
  useTemplateRef,
  nextTick,
} from "vue";
import { useI18n } from "vue-i18n";
import { Chart, registerables } from "chart.js";
import { getWorkouts, getSetting } from "@/utils/database.js";
import NeoPanel from "@/components/NeoPanel.vue";

Chart.register(...registerables);

const props = defineProps({
  exercise: {
    type: Object,
    required: true,
  },
  isOpen: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(["close"]);

const { t } = useI18n();
const chartCanvas = useTemplateRef("chartCanvas");
const loading = ref(false);
const chartData = ref([]);
const chart = ref(null);
const weightUnit = ref("kg");
const distanceUnit = ref("km");

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
  const sets = [];
  [...chartData.value]
    .sort((a, b) => new Date(b.started) - new Date(a.started))
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
                date: workout.started,
                index,
              });
            });
        });
    });

  return sets.sort((a, b) => new Date(b.date) - new Date(a.date));
});

const formatDate = (dateStr) => {
  const date = new Date(dateStr);
  return date.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year:
      date.getFullYear() !== new Date().getFullYear() ? "numeric" : undefined,
  });
};

const closeModal = () => {
  emit("close");
};

const loadData = async () => {
  loading.value = true;
  try {
    const [workouts, weightSetting, distanceSetting] = await Promise.all([
      getWorkouts(),
      getSetting("weightUnit", "kg"),
      getSetting("distanceUnit", "km"),
    ]);

    weightUnit.value = weightSetting;
    distanceUnit.value = distanceSetting;

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

  // Prepare data for chart
  const dataPoints = [];
  chartData.value
    .sort((a, b) => new Date(a.started) - new Date(b.started))
    .forEach((workout) => {
      workout.exercises
        .filter((ex) => ex.name === props.exercise.name)
        .forEach((ex) => {
          ex.sets
            .filter((s) => s.type !== "warmup")
            .forEach((set) => {
              if (props.exercise.type === "cardio") {
                if (set.distance) {
                  dataPoints.push({
                    x: new Date(workout.started),
                    y: set.distance,
                  });
                }
              } else {
                if (set.weight && set.reps) {
                  dataPoints.push({
                    x: new Date(workout.started),
                    y: set.weight,
                  });
                }
              }
            });
        });
    });

  if (dataPoints.length === 0) return;

  chart.value = new Chart(ctx, {
    type: "line",
    data: {
      datasets: [
        {
          label:
            props.exercise.type === "cardio"
              ? t("exercise.stats.distance")
              : t("exercise.stats.weight"),
          data: dataPoints,
          borderColor: "#a855f7",
          backgroundColor: "rgba(168, 85, 247, 0.1)",
          borderWidth: 2,
          fill: true,
          tension: 0.2,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: {
          display: false,
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
            color: "#6b7280",
          },
          grid: {
            color: "rgba(107, 114, 128, 0.2)",
          },
        },
        y: {
          beginAtZero: true,
          ticks: {
            color: "#6b7280",
            callback: function (value) {
              if (props.exercise.type === "cardio") {
                return value + distanceUnit.value;
              } else {
                return value + weightUnit.value;
              }
            },
          },
          grid: {
            color: "rgba(107, 114, 128, 0.2)",
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
      await loadData();
      await nextTick();
      createChart();
    } else {
      if (chart.value) {
        chart.value.destroy();
        chart.value = null;
      }
    }
  }
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
