<template>
  <div class="statistics h-full flex flex-col">
    <!-- Header -->
    <NeoHeader :title="t('statistics.title')" />

    <!-- Content -->
    <div class="flex-1 overflow-auto hide-scrollbar">
      <div v-if="loading" class="flex items-center justify-center h-32">
        <div class="text-gray-500">{{ t("statistics.loading") }}</div>
      </div>

      <div
        v-else-if="workouts.length === 0"
        class="flex flex-col items-center justify-center h-32 text-gray-500"
      >
        <div class="text-lg mb-2">📊</div>
        <div>{{ t("statistics.noData") }}</div>
      </div>

      <div v-else class="p-4 space-y-6">
        <!-- Overview Stats -->
        <div class="grid grid-cols-2 gap-4">
          <div class="neo-stats-card">
            <div class="text-2xl font-black neo-text">{{ totalWorkouts }}</div>
            <div class="text-sm neo-text-muted">
              {{ t("statistics.cards.totalWorkouts") }}
            </div>
          </div>
          <div class="neo-stats-card">
            <div class="text-2xl font-black neo-text">
              {{ averageWorkoutTime }}
            </div>
            <div class="text-sm neo-text-muted">
              {{ t("statistics.cards.avgDuration") }}
            </div>
          </div>
          <div class="neo-stats-card">
            <div class="text-2xl font-black neo-text">{{ totalSets }}</div>
            <div class="text-sm neo-text-muted">
              {{ t("statistics.cards.totalSets") }}
            </div>
          </div>
          <div class="neo-stats-card">
            <div class="text-2xl font-black neo-text">
              {{ uniqueExercises }}
            </div>
            <div class="text-sm neo-text-muted">
              {{ t("statistics.cards.exercisesUsed") }}
            </div>
          </div>
        </div>

        <!-- Workouts per Month Chart -->
        <div class="neo-chart-container">
          <h3 class="neo-heading-3">
            {{ t("statistics.charts.workoutsPerMonth") }}
          </h3>
          <div class="h-64">
            <canvas ref="workoutsChartCanvas"></canvas>
          </div>
        </div>

        <!-- Workout Duration Chart -->
        <div class="neo-chart-container">
          <h3 class="neo-heading-3">
            {{ t("statistics.charts.durationTrends") }}
          </h3>
          <div class="h-64">
            <canvas ref="durationChartCanvas"></canvas>
          </div>
        </div>

        <!-- Most Used Exercises -->
        <div class="neo-chart-container">
          <h3 class="neo-heading-3">
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
                  class="w-8 h-8 neo-button-accent rounded-full flex items-center justify-center text-sm font-bold"
                >
                  {{ index + 1 }}
                </div>
                <div>
                  <div class="font-bold neo-text">{{ exercise.name }}</div>
                  <div class="text-sm neo-text-muted">
                    {{ exercise.muscleGroup }}
                  </div>
                </div>
              </div>
              <div class="text-right">
                <div class="font-bold neo-text">{{ exercise.count }}</div>
                <div class="text-sm neo-text-muted">
                  {{ t("statistics.charts.workouts") }}
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Exercise Volume Chart -->
        <div class="neo-chart-container">
          <h3 class="neo-heading-3">
            {{ t("statistics.charts.exerciseVolume") }}
          </h3>
          <div class="h-64">
            <canvas ref="volumeChartCanvas"></canvas>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import { useHead } from "@unhead/vue";
import { Chart, registerables } from "chart.js";
import { getWorkouts } from "@/utils/database.js";
import NeoHeader from "@/components/NeoHeader.vue";

Chart.register(...registerables);

const { t, d } = useI18n();
const workouts = ref([]);
const loading = ref(true);

// Set page title
useHead({
  title: () => `${t('statistics.title')} - Plonkout`
});

const workoutsChartCanvas = ref(null);
const durationChartCanvas = ref(null);
const volumeChartCanvas = ref(null);

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
    await nextTick();
    createCharts();
  } catch (error) {
    console.error(t("statistics.loadError"), error);
  } finally {
    loading.value = false;
  }
}

/**
 * Create all charts
 */
function createCharts() {
  createWorkoutsChart();
  createDurationChart();
  createVolumeChart();
}

/**
 * Create workouts per month chart
 */
function createWorkoutsChart() {
  if (!workoutsChartCanvas.value) return;

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

  workoutsChart = new Chart(workoutsChartCanvas.value, {
    type: "bar",
    data: {
      labels: labels.map((label) => {
        const [year, month] = label.split("-");
        return d(new Date(year, month - 1), "monthYear");
      }),
      datasets: [
        {
          label: t("statistics.charts.workouts"),
          data: data,
          backgroundColor: "rgba(59, 130, 246, 0.5)",
          borderColor: "rgb(59, 130, 246)",
          borderWidth: 1,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      scales: {
        y: {
          beginAtZero: true,
          ticks: {
            stepSize: 1,
          },
        },
      },
    },
  });
}

/**
 * Create workout duration trend chart
 */
function createDurationChart() {
  if (!durationChartCanvas.value) return;

  const recentWorkouts = workouts.value
    .filter((w) => w.started && w.ended)
    .slice(-10) // Last 10 workouts with duration
    .map((w) => {
      const start = new Date(w.started);
      const end = new Date(w.ended);
      const duration = Math.round((end - start) / (1000 * 60));
      return {
        date: d(start, "shortDate"),
        duration: duration > 0 ? duration : 0,
      };
    });

  if (recentWorkouts.length === 0) return;

  durationChart = new Chart(durationChartCanvas.value, {
    type: "line",
    data: {
      labels: recentWorkouts.map((w) => w.date),
      datasets: [
        {
          label: t("statistics.charts.duration"),
          data: recentWorkouts.map((w) => w.duration),
          borderColor: "rgb(34, 197, 94)",
          backgroundColor: "rgba(34, 197, 94, 0.1)",
          tension: 0.1,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      scales: {
        y: {
          beginAtZero: true,
          title: {
            display: true,
            text: t("statistics.charts.minutes"),
          },
        },
      },
    },
  });
}

/**
 * Create exercise volume chart
 */
function createVolumeChart() {
  if (!volumeChartCanvas.value) return;

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

  if (topVolumeExercises.length === 0) return;

  volumeChart = new Chart(volumeChartCanvas.value, {
    type: "doughnut",
    data: {
      labels: topVolumeExercises.map(([name]) => name),
      datasets: [
        {
          data: topVolumeExercises.map(([, volume]) => volume),
          backgroundColor: [
            "rgba(59, 130, 246, 0.8)",
            "rgba(34, 197, 94, 0.8)",
            "rgba(168, 85, 247, 0.8)",
            "rgba(249, 115, 22, 0.8)",
            "rgba(239, 68, 68, 0.8)",
          ],
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: {
          position: "bottom",
        },
      },
    },
  });
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

// Cleanup on unmount
import { onUnmounted } from "vue";
onUnmounted(() => {
  destroyCharts();
});
</script>
