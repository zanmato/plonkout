<template>
  <div class="statistics h-full flex flex-col">
    <!-- Header -->
    <NeoHeader :title="t('statistics.title')" />

    <!-- Content -->
    <div class="flex-1 overflow-auto hide-scrollbar">
      <div v-if="loading" class="flex items-center justify-center h-32">
        <div class="text-gray-500">{{ t("statistics.loading") }}</div>
      </div>

      <NeoPanel v-else-if="workouts.length === 0" class="text-center py-8 m-4">
        <div class="text-base font-bold text-black dark:text-white mb-2">
          {{ t("statistics.noData") }}
        </div>
      </NeoPanel>

      <div v-else class="p-4 space-y-6">
        <!-- Overview Stats -->
        <div class="grid grid-cols-2 gap-4">
          <NeoPanel class="text-center">
            <div class="text-2xl font-black text-black dark:text-white">
              {{ totalWorkouts }}
            </div>
            <div class="text-sm text-black dark:text-white opacity-70">
              {{ t("statistics.cards.totalWorkouts") }}
            </div>
          </NeoPanel>
          <NeoPanel class="text-center">
            <div class="text-2xl font-black text-black dark:text-white">
              {{ averageWorkoutTime }}
            </div>
            <div class="text-sm text-black dark:text-white opacity-70">
              {{ t("statistics.cards.avgDuration") }}
            </div>
          </NeoPanel>
          <NeoPanel class="text-center">
            <div class="text-2xl font-black text-black dark:text-white">
              {{ totalSets }}
            </div>
            <div class="text-sm text-black dark:text-white opacity-70">
              {{ t("statistics.cards.totalSets") }}
            </div>
          </NeoPanel>
          <NeoPanel class="text-center">
            <div class="text-2xl font-black text-black dark:text-white">
              {{ uniqueExercises }}
            </div>
            <div class="text-sm text-black dark:text-white opacity-70">
              {{ t("statistics.cards.exercisesUsed") }}
            </div>
          </NeoPanel>
        </div>

        <!-- Workouts per Month Chart -->
        <NeoPanel>
          <h3 class="text-xl font-semibold text-black dark:text-white mb-2">
            {{ t("statistics.charts.workoutsPerMonth") }}
          </h3>
          <div class="h-64">
            <canvas ref="workoutsChartCanvas"></canvas>
          </div>
        </NeoPanel>

        <!-- Workout Duration Chart -->
        <NeoPanel>
          <h3 class="text-xl font-semibold text-black dark:text-white mb-2">
            {{ t("statistics.charts.durationTrends") }}
          </h3>
          <div class="h-64">
            <canvas ref="durationChartCanvas"></canvas>
          </div>
        </NeoPanel>

        <!-- Most Used Exercises -->
        <NeoPanel>
          <h3 class="text-xl font-semibold text-black dark:text-white mb-2">
            {{ t("statistics.charts.mostUsedExercises") }}
          </h3>
          <div class="space-y-3">
            <div
              v-for="(exercise, index) in topExercises"
              :key="exercise.name"
              class="flex items-center justify-between"
            >
              <div class="flex items-center space-x-3">
                <div
                  class="w-8 h-8 bg-nb-main rounded-full flex items-center justify-center text-sm font-bold text-white border-2 border-nb-border shadow-brutal-sm"
                >
                  {{ index + 1 }}
                </div>
                <div>
                  <div class="font-bold text-black dark:text-white">
                    {{ exercise.name }}
                  </div>
                  <div class="text-sm text-black dark:text-white opacity-70">
                    {{ exercise.muscleGroup }}
                  </div>
                </div>
              </div>
              <div class="text-right">
                <div class="font-bold text-black dark:text-white">
                  {{ exercise.count }}
                </div>
                <div class="text-sm text-black dark:text-white opacity-70">
                  {{ t("statistics.charts.workouts") }}
                </div>
              </div>
            </div>
          </div>
        </NeoPanel>

        <!-- Exercise Volume Chart -->
        <NeoPanel>
          <h3 class="text-xl font-semibold text-black dark:text-white mb-2">
            {{ t("statistics.charts.exerciseVolume") }}
          </h3>
          <div class="h-64">
            <canvas ref="volumeChartCanvas"></canvas>
          </div>
        </NeoPanel>
      </div>
    </div>
  </div>
</template>

<script setup>
import {
  ref,
  onMounted,
  onUnmounted,
  computed,
  nextTick,
  useTemplateRef,
  watch,
} from "vue";
import { useI18n } from "vue-i18n";
import { useHead } from "@unhead/vue";
import { Chart, registerables } from "chart.js";
import { getWorkouts } from "@/utils/database.js";
import NeoHeader from "@/components/NeoHeader.vue";
import NeoPanel from "@/components/NeoPanel.vue";
import "chartjs-adapter-date-fns";

Chart.register(...registerables);

const { t, d } = useI18n();
const workouts = ref([]);
const loading = ref(true);

// Set page title
useHead({
  title: () => t("statistics.title"),
});

const workoutsChartCanvas = useTemplateRef("workoutsChartCanvas");
const durationChartCanvas = useTemplateRef("durationChartCanvas");
const volumeChartCanvas = useTemplateRef("volumeChartCanvas");

const isDark = document.documentElement.classList.contains("dark");
const textColor = isDark ? "#ffffff" : "#000000";

let workoutsChart = null;
let durationChart = null;
let volumeChart = null;

/**
 * Total number of workouts
 */
const totalWorkouts = computed(() => workouts.value.length);

/**
 * Average workout duration in minutes
 */
const averageWorkoutTime = computed(() => {
  const durations = workouts.value
    .filter((w) => w.started && w.ended)
    .map((w) => {
      const start = new Date(w.started);
      const end = new Date(w.ended);
      return Math.round((end - start) / (1000 * 60));
    })
    .filter((d) => d > 0);

  if (durations.length === 0) return "-";

  const avg = Math.round(
    durations.reduce((sum, d) => sum + d, 0) / durations.length
  );
  const hours = Math.floor(avg / 60);
  const minutes = avg % 60;

  if (hours > 0 && minutes > 0) {
    return `${hours}${t("common.duration.hours")} ${minutes}${t(
      "common.duration.minutes"
    )}`;
  } else if (hours > 0) {
    return `${hours}${t("common.duration.hours")}`;
  } else {
    return `${minutes}${t("common.duration.minutes")}`;
  }
});

/**
 * Total number of sets across all workouts
 */
const totalSets = computed(() => {
  return workouts.value.reduce((total, workout) => {
    if (!workout.exercises) return total;
    return (
      total +
      workout.exercises.reduce((exerciseTotal, exercise) => {
        if (!exercise.sets) return exerciseTotal;
        return (
          exerciseTotal +
          exercise.sets.filter((set) => set.type !== "warmup").length
        );
      }, 0)
    );
  }, 0);
});

/**
 * Number of unique exercises used
 */
const uniqueExercises = computed(() => {
  const exerciseNames = new Set();
  workouts.value.forEach((workout) => {
    if (workout.exercises) {
      workout.exercises.forEach((exercise) => {
        exerciseNames.add(exercise.name);
      });
    }
  });
  return exerciseNames.size;
});

/**
 * Top 5 most used exercises
 */
const topExercises = computed(() => {
  const exerciseCounts = {};

  workouts.value.forEach((workout) => {
    if (workout.exercises) {
      workout.exercises.forEach((exercise) => {
        if (!exerciseCounts[exercise.name]) {
          exerciseCounts[exercise.name] = {
            name: exercise.name,
            muscleGroup:
              exercise.muscleGroup || t("exercise.selector.defaultMuscleGroup"),
            count: 0,
          };
        }
        exerciseCounts[exercise.name].count++;
      });
    }
  });

  return Object.values(exerciseCounts)
    .sort((a, b) => b.count - a.count)
    .slice(0, 5);
});

/**
 * Load workouts from database
 */
async function loadWorkouts() {
  try {
    loading.value = true;
    workouts.value = await getWorkouts();

    // Charts will be created automatically by watchers when canvases become available
  } catch (error) {
    console.error(t("statistics.loadError"), error);
  } finally {
    loading.value = false;
  }
}

// Track which charts have been created
const chartsCreated = ref({
  workouts: false,
  duration: false,
  volume: false,
});

// Watch each canvas template ref and create chart when available
watch(
  workoutsChartCanvas,
  (canvas) => {
    if (canvas && !chartsCreated.value.workouts && workouts.value.length >= 0) {
      console.log("Workouts chart canvas available, creating chart...");
      const success = createWorkoutsChart();
      if (success) {
        chartsCreated.value.workouts = true;
      }
    }
  },
  { immediate: true }
);

watch(
  durationChartCanvas,
  (canvas) => {
    if (canvas && !chartsCreated.value.duration && workouts.value.length >= 0) {
      console.log("Duration chart canvas available, creating chart...");
      const success = createDurationChart();
      if (success) {
        chartsCreated.value.duration = true;
      }
    }
  },
  { immediate: true }
);

watch(
  volumeChartCanvas,
  (canvas) => {
    if (canvas && !chartsCreated.value.volume && workouts.value.length >= 0) {
      console.log("Volume chart canvas available, creating chart...");
      const success = createVolumeChart();
      if (success) {
        chartsCreated.value.volume = true;
      }
    }
  },
  { immediate: true }
);

// Also watch for workout data changes to recreate charts if needed
watch(
  workouts,
  () => {
    // Reset chart creation flags and recreate charts
    chartsCreated.value.workouts = false;
    chartsCreated.value.duration = false;
    chartsCreated.value.volume = false;

    nextTick(() => {
      // Trigger chart recreation by re-checking canvases
      if (workoutsChartCanvas.value) {
        const success = createWorkoutsChart();
        if (success) chartsCreated.value.workouts = true;
      }
      if (durationChartCanvas.value) {
        const success = createDurationChart();
        if (success) chartsCreated.value.duration = true;
      }
      if (volumeChartCanvas.value) {
        const success = createVolumeChart();
        if (success) chartsCreated.value.volume = true;
      }
    });
  },
  { deep: true }
);

/**
 * Create workouts per month chart
 */
function createWorkoutsChart() {
  if (!workoutsChartCanvas.value) {
    console.warn("Workouts chart canvas not found");
    return false;
  }

  // Destroy existing chart if it exists
  if (workoutsChart) {
    workoutsChart.destroy();
  }

  const monthCounts = {};

  workouts.value.forEach((workout) => {
    const date = new Date(workout.started);
    const monthKey = `${date.getFullYear()}-${String(
      date.getMonth() + 1
    ).padStart(2, "0")}`;
    monthCounts[monthKey] = (monthCounts[monthKey] || 0) + 1;
  });

  const labels = Object.keys(monthCounts).sort().slice(-6); // Last 6 months
  const data = labels.map((label) => monthCounts[label] || 0);

  // If no data, create a chart with sample data to show the structure
  if (labels.length === 0) {
    const currentDate = new Date();
    const currentMonth = `${currentDate.getFullYear()}-${String(
      currentDate.getMonth() + 1
    ).padStart(2, "0")}`;
    labels.push(currentMonth);
    data.push(0);
  }

  try {
    workoutsChart = new Chart(workoutsChartCanvas.value, {
      type: "bar",
      data: {
        labels: labels.map((label) => {
          const [year, month] = label.split("-");
          const date = new Date(year, month - 1);
          return d(date, "month");
        }),
        datasets: [
          {
            label: t("statistics.charts.workouts"),
            data: data,
            backgroundColor: "rgba(136, 170, 238, 0.6)",
            borderColor: "rgb(136, 170, 238)",
            borderWidth: 2,
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
            ticks: {
              color: textColor,
            },
          },
          y: {
            beginAtZero: true,
            ticks: {
              stepSize: 1,
              color: textColor,
            },
          },
        },
      },
    });
    return true;
  } catch (error) {
    console.error("Error creating workouts chart:", error);
    return false;
  }
}

/**
 * Create workout duration trend chart
 */
function createDurationChart() {
  if (!durationChartCanvas.value) {
    console.warn("Duration chart canvas not found");
    return false;
  }

  // Destroy existing chart if it exists
  if (durationChart) {
    durationChart.destroy();
  }

  const recentWorkouts = workouts.value
    .filter((w) => w.started && w.ended)
    .slice(-10) // Last 10 workouts with duration
    .map((w) => {
      const start = new Date(w.started);
      const end = new Date(w.ended);
      const duration = Math.round((end - start) / (1000 * 60));
      return {
        date: d(start, "short"),
        duration: duration > 0 ? duration : 0,
      };
    });

  // If no data, show placeholder
  let labels = [];
  let data = [];

  if (recentWorkouts.length === 0) {
    labels = [t("statistics.noData")];
    data = [0];
  } else {
    labels = recentWorkouts.map((w) => w.date);
    data = recentWorkouts.map((w) => w.duration);
  }

  try {
    durationChart = new Chart(durationChartCanvas.value, {
      type: "line",
      data: {
        labels: labels,
        datasets: [
          {
            label: t("statistics.charts.duration"),
            data: data,
            borderColor: "rgb(77, 144, 254)",
            backgroundColor: "rgba(77, 144, 254, 0.1)",
            borderWidth: 2,
            fill: true,
            tension: 0.3,
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
            ticks: {
              color: textColor,
            },
          },
          y: {
            beginAtZero: true,
            ticks: {
              color: textColor,
            },
            title: {
              display: true,
              text: t("statistics.charts.minutes"),
              color: textColor,
            },
          },
        },
      },
    });
    return true;
  } catch (error) {
    console.error("Error creating duration chart:", error);
    return false;
  }
}

/**
 * Create exercise volume chart
 */
function createVolumeChart() {
  if (!volumeChartCanvas.value) {
    console.warn("Volume chart canvas not found");
    return false;
  }

  // Destroy existing chart if it exists
  if (volumeChart) {
    volumeChart.destroy();
  }

  const exerciseVolume = {};

  workouts.value.forEach((workout) => {
    if (workout.exercises) {
      workout.exercises.forEach((exercise) => {
        if (!exerciseVolume[exercise.name]) {
          exerciseVolume[exercise.name] = 0;
        }

        if (exercise.sets) {
          exercise.sets.forEach((set) => {
            if (set.type !== "warmup" && set.weight && set.reps) {
              exerciseVolume[exercise.name] += set.weight * set.reps;
            }
          });
        }
      });
    }
  });

  const topVolumeExercises = Object.entries(exerciseVolume)
    .sort(([, a], [, b]) => b - a)
    .slice(0, 5);

  // If no data, show placeholder
  let labels = [];
  let data = [];
  let backgroundColor = [];

  if (topVolumeExercises.length === 0) {
    labels = [t("statistics.noData")];
    data = [1];
    backgroundColor = ["rgba(136, 170, 238, 0.3)"];
  } else {
    labels = topVolumeExercises.map(([name]) => name);
    data = topVolumeExercises.map(([, volume]) => volume);
    backgroundColor = [
      "rgba(136, 170, 238, 0.8)",
      "rgba(77, 144, 254, 0.8)",
      "rgba(168, 85, 247, 0.8)",
      "rgba(249, 115, 22, 0.8)",
      "rgba(239, 68, 68, 0.8)",
    ];
  }

  try {
    volumeChart = new Chart(volumeChartCanvas.value, {
      type: "doughnut",
      data: {
        labels: labels,
        datasets: [
          {
            data: data,
            backgroundColor: backgroundColor,
            borderColor: "#000000",
            borderWidth: 2,
          },
        ],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: {
            position: "bottom",
            labels: {
              padding: 15,
              usePointStyle: true,
              color: textColor,
              font: {
                family: "DM Sans",
                weight: "600",
              },
            },
          },
        },
      },
    });
    return true;
  } catch (error) {
    console.error("Error creating volume chart:", error);
    return false;
  }
}

/**
 * Cleanup charts on unmount
 */
function destroyCharts() {
  if (workoutsChart) {
    workoutsChart.destroy();
    workoutsChart = null;
  }
  if (durationChart) {
    durationChart.destroy();
    durationChart = null;
  }
  if (volumeChart) {
    volumeChart.destroy();
    volumeChart = null;
  }
}

onMounted(() => {
  loadWorkouts();
});

onUnmounted(() => {
  destroyCharts();
});
</script>
