<template>
  <div
    v-if="isOpen"
    class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50"
    @click="closeModal"
  >
    <NeoPanel class="w-full max-w-lg max-h-[90vh] overflow-y-auto" @click.stop>
      <!-- Header -->
      <div class="flex items-center justify-between mb-4">
        <div class="flex items-center gap-2">
          <h3 class="text-xl font-bold text-black dark:text-white">
            {{ t("exercise.plan.title", { exercise: exercise.name }) }}
          </h3>
          <button
            @click="showSettings = !showSettings"
            class="w-7 h-7 flex items-center justify-center rounded-full hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
            :title="t('exercise.plan.configureExercise')"
          >
            <span class="material-icons text-lg">{{
              showSettings ? "close" : "settings"
            }}</span>
          </button>
        </div>
        <button
          @click="closeModal"
          class="w-8 h-8 flex items-center justify-center rounded-full hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
        >
          <span class="material-icons text-xl">close</span>
        </button>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="flex items-center justify-center h-32">
        <div class="text-gray-500">{{ t("common.loading") }}</div>
      </div>

      <!-- No Historical Data -->
      <div v-else-if="!loading && estimated1RM <= 0" class="text-center py-8">
        <span class="material-icons text-4xl text-gray-400 mb-2"
          >fitness_center</span
        >
        <div class="text-base font-bold text-black dark:text-white mb-2">
          {{ t("exercise.plan.noData") }}
        </div>
        <div class="text-sm text-black dark:text-white opacity-70">
          {{ t("exercise.plan.noDataDescription") }}
        </div>
      </div>

      <!-- Settings Panel (collapsible) -->
      <div
        v-else-if="showSettings"
        class="space-y-4 mb-6 p-4 bg-nb-overlay dark:bg-zinc-800 border-2 border-nb-border rounded-lg"
      >
        <h4 class="text-lg font-semibold text-black dark:text-white">
          {{ t("exercise.plan.exerciseSettings") }}
        </h4>
        <p class="text-sm text-black dark:text-white opacity-70">
          {{ t("exercise.plan.settingsDescription") }}
        </p>

        <!-- Weeks per Block -->
        <div>
          <label
            class="block text-sm font-medium text-black dark:text-white mb-1"
          >
            {{ t("exercise.plan.weeksPerBlock") }}
            <span
              v-if="!hasExerciseOverride('weeksPerBlock')"
              class="opacity-50"
            >
              ({{ t("exercise.plan.global") }})
            </span>
          </label>
          <input
            v-model.number="exerciseWeeksPerBlock"
            type="number"
            min="0"
            max="12"
            :placeholder="String(globalSettings.weeksPerBlock)"
            @change="saveExerciseSettings"
            class="w-full px-3 py-2 bg-white dark:bg-zinc-700 border-2 border-nb-border rounded-md text-black dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
          />
          <p class="text-xs text-black dark:text-white opacity-50 mt-1">
            {{ t("exercise.plan.setZeroForGlobal") }}
          </p>
        </div>

        <!-- Workouts per Week -->
        <div>
          <label
            class="block text-sm font-medium text-black dark:text-white mb-1"
          >
            {{ t("exercise.plan.workoutsPerWeek") }}
            <span
              v-if="!hasExerciseOverride('workoutsPerWeek')"
              class="opacity-50"
            >
              ({{ t("exercise.plan.global") }})
            </span>
          </label>
          <input
            v-model.number="exerciseWorkoutsPerWeek"
            type="number"
            min="0"
            max="7"
            :placeholder="String(globalSettings.workoutsPerWeek)"
            @change="saveExerciseSettings"
            class="w-full px-3 py-2 bg-white dark:bg-zinc-700 border-2 border-nb-border rounded-md text-black dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
          />
        </div>

        <!-- Start Percentage -->
        <div>
          <label
            class="block text-sm font-medium text-black dark:text-white mb-1"
          >
            {{ t("exercise.plan.startPercentage") }}
            <span
              v-if="!hasExerciseOverride('startPercentage')"
              class="opacity-50"
            >
              ({{ t("exercise.plan.global") }})
            </span>
          </label>
          <div class="flex items-center">
            <input
              v-model.number="exerciseStartPercentage"
              type="number"
              min="0"
              max="90"
              :placeholder="String(globalSettings.startPercentage)"
              @change="saveExerciseSettings"
              class="flex-1 px-3 py-2 bg-white dark:bg-zinc-700 border-2 border-nb-border rounded-md text-black dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
            />
            <span class="ml-2 text-black dark:text-white">%</span>
          </div>
        </div>

        <!-- Progression per Week -->
        <div>
          <label
            class="block text-sm font-medium text-black dark:text-white mb-1"
          >
            {{ t("exercise.plan.progressionPerWeek") }}
            <span
              v-if="!hasExerciseOverride('progressionPerWeek')"
              class="opacity-50"
            >
              ({{ t("exercise.plan.global") }})
            </span>
          </label>
          <div class="flex items-center">
            <input
              v-model.number="exerciseProgressionPerWeek"
              type="number"
              min="0"
              max="10"
              :placeholder="String(globalSettings.progressionPerWeek)"
              @change="saveExerciseSettings"
              class="flex-1 px-3 py-2 bg-white dark:bg-zinc-700 border-2 border-nb-border rounded-md text-black dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
            />
            <span class="ml-2 text-black dark:text-white">%</span>
          </div>
        </div>
      </div>

      <!-- Main Content -->
      <div v-else-if="!loading && estimated1RM > 0" class="space-y-6">
        <!-- No Active Block State -->
        <div v-if="!blockData.hasActiveBlock" class="text-center py-4">
          <span class="material-icons text-4xl text-purple-500 mb-2"
            >timeline</span
          >
          <div class="text-base font-bold text-black dark:text-white mb-2">
            {{ t("exercise.plan.noActiveBlock") }}
          </div>
          <div class="text-sm text-black dark:text-white opacity-70 mb-4">
            {{ t("exercise.plan.noActiveBlockDescription") }}
          </div>

          <!-- Estimated 1RM Display -->
          <div
            class="bg-nb-overlay dark:bg-zinc-800 border-2 border-nb-border rounded-lg p-4 mb-4"
          >
            <div class="text-sm text-black dark:text-white opacity-70">
              {{ t("exercise.plan.estimated1RM") }}
            </div>
            <div class="text-2xl font-bold text-purple-500">
              {{ estimated1RM }} {{ weightUnit }}
            </div>
          </div>

          <!-- Starting Week Selector -->
          <div class="mb-4">
            <label
              class="block text-sm font-medium text-black dark:text-white mb-2"
            >
              {{ t("exercise.plan.startAtWeek") }}
            </label>
            <div class="flex items-center justify-center gap-2">
              <button
                @click="startWeek = Math.max(1, startWeek - 1)"
                class="w-8 h-8 flex items-center justify-center rounded-full bg-nb-overlay dark:bg-zinc-700 border-2 border-nb-border hover:bg-gray-200 dark:hover:bg-zinc-600"
              >
                <span class="material-icons text-sm">remove</span>
              </button>
              <div
                class="text-2xl font-bold text-black dark:text-white w-16 text-center"
              >
                {{ startWeek }}
              </div>
              <button
                @click="
                  startWeek = Math.min(effectiveTotalWeeks, startWeek + 1)
                "
                class="w-8 h-8 flex items-center justify-center rounded-full bg-nb-overlay dark:bg-zinc-700 border-2 border-nb-border hover:bg-gray-200 dark:hover:bg-zinc-600"
              >
                <span class="material-icons text-sm">add</span>
              </button>
            </div>
            <div class="text-xs text-black dark:text-white opacity-50 mt-1">
              {{ t("exercise.plan.startAtWeekDescription") }}
            </div>
          </div>

          <NeoButton @click="handleStartNewBlock" variant="primary" full-width>
            <template #icon>
              <span class="material-icons">play_arrow</span>
            </template>
            {{ t("exercise.plan.startNewBlock") }}
          </NeoButton>
        </div>

        <!-- Block Complete State -->
        <div v-else-if="blockData.isComplete" class="text-center py-4">
          <span class="material-icons text-4xl text-green-500 mb-2"
            >check_circle</span
          >
          <div class="text-base font-bold text-black dark:text-white mb-2">
            {{ t("exercise.plan.blockComplete") }}
          </div>
          <div class="text-sm text-black dark:text-white opacity-70 mb-4">
            {{ t("exercise.plan.blockCompleteDescription") }}
          </div>

          <!-- Starting vs Target -->
          <div class="grid grid-cols-2 gap-4 mb-4">
            <div
              class="bg-nb-overlay dark:bg-zinc-800 border-2 border-nb-border rounded-lg p-3 text-center"
            >
              <div class="text-sm text-black dark:text-white opacity-70">
                {{ t("exercise.plan.startingMax") }}
              </div>
              <div class="text-lg font-bold text-black dark:text-white">
                {{ blockData.blockState.startingEstimated1RM }} {{ weightUnit }}
              </div>
            </div>
            <div
              class="bg-nb-overlay dark:bg-zinc-800 border-2 border-nb-border rounded-lg p-3 text-center"
            >
              <div class="text-sm text-black dark:text-white opacity-70">
                {{ t("exercise.plan.targetMax") }}
              </div>
              <div class="text-lg font-bold text-purple-500">
                {{ blockData.blockState.targetMax }} {{ weightUnit }}
              </div>
            </div>
          </div>

          <NeoButton @click="handleStartNewBlock" variant="primary" full-width>
            <template #icon>
              <span class="material-icons">refresh</span>
            </template>
            {{ t("exercise.plan.startNewBlock") }}
          </NeoButton>
        </div>

        <!-- Active Block State -->
        <div v-else>
          <!-- Week Indicator -->
          <div class="text-center mb-4">
            <div class="text-sm text-black dark:text-white opacity-70">
              {{ t("exercise.plan.currentBlock") }}
            </div>
            <div class="text-2xl font-bold text-black dark:text-white">
              {{
                t("exercise.plan.weekOf", {
                  week: blockData.currentWeek,
                  total: blockData.totalWeeks,
                })
              }}
            </div>
          </div>

          <!-- Recommended Weight (Main Display) -->
          <div
            class="bg-purple-100 dark:bg-purple-900/30 border-2 border-purple-500 rounded-lg p-6 text-center mb-4"
          >
            <div class="text-sm text-black dark:text-white opacity-70 mb-1">
              {{ t("exercise.plan.recommendedWeight") }}
            </div>
            <div class="text-4xl font-bold text-purple-500 mb-2">
              {{ blockData.recommendedWeight }} {{ weightUnit }}
            </div>
            <div class="text-sm text-black dark:text-white opacity-70">
              {{ blockData.currentPercentage }}% {{ t("exercise.plan.ofMax") }}
            </div>
          </div>

          <!-- Rep Range -->
          <div
            v-if="blockData.recommendedReps"
            class="bg-nb-overlay dark:bg-zinc-800 border-2 border-nb-border rounded-lg p-4 text-center mb-4"
          >
            <div class="text-sm text-black dark:text-white opacity-70">
              {{ t("exercise.plan.targetReps") }}
            </div>
            <div class="text-xl font-bold text-black dark:text-white">
              {{ blockData.recommendedReps.min }}-{{
                blockData.recommendedReps.max
              }}
              {{ t("exercise.plan.reps") }}
            </div>
          </div>

          <!-- Block Progress Bar -->
          <div class="mb-4">
            <div
              class="flex justify-between text-sm text-black dark:text-white mb-2"
            >
              <span>{{ t("exercise.plan.blockProgress") }}</span>
              <span>
                {{ blockData.blockState.blockWorkoutCount }}/{{
                  blockData.totalWorkouts
                }}
                {{ t("exercise.plan.workouts") }}
              </span>
            </div>
            <div
              class="h-3 bg-gray-200 dark:bg-zinc-700 rounded-full overflow-hidden"
            >
              <div
                class="h-full bg-purple-500 transition-all duration-300"
                :style="{
                  width: `${(blockData.blockState.blockWorkoutCount / blockData.totalWorkouts) * 100}%`,
                }"
              ></div>
            </div>
            <!-- Week markers -->
            <div class="flex justify-between mt-1">
              <span
                v-for="w in blockData.totalWeeks"
                :key="w"
                class="text-xs"
                :class="
                  w <= blockData.currentWeek
                    ? 'text-purple-500 font-bold'
                    : 'text-gray-400'
                "
              >
                W{{ w }}
              </span>
            </div>
          </div>

          <!-- Estimated 1RM Info -->
          <div
            class="bg-nb-overlay dark:bg-zinc-800 border-2 border-nb-border rounded-lg p-3 mb-4"
          >
            <div class="flex justify-between items-center">
              <span class="text-sm text-black dark:text-white opacity-70">
                {{ t("exercise.plan.estimated1RM") }}
              </span>
              <span class="text-sm font-semibold text-black dark:text-white">
                {{ blockData.blockState.startingEstimated1RM }} {{ weightUnit }}
              </span>
            </div>
            <div class="flex justify-between items-center mt-2">
              <span class="text-sm text-black dark:text-white opacity-70">
                {{ t("exercise.plan.targetMax") }}
              </span>
              <span class="text-sm font-semibold text-purple-500">
                {{ blockData.blockState.targetMax }} {{ weightUnit }}
              </span>
            </div>
          </div>

          <!-- Action Buttons -->
          <div class="flex gap-3">
            <NeoButton
              v-if="onApplyWeight"
              @click="applyWeight"
              variant="primary"
              full-width
            >
              <template #icon>
                <span class="material-icons">add</span>
              </template>
              {{ t("exercise.plan.applyToSet") }}
            </NeoButton>
            <NeoButton
              @click="handleStartNewBlock"
              :variant="onApplyWeight ? 'secondary' : 'primary'"
              :full-width="!onApplyWeight"
              :class="{ 'flex-1': onApplyWeight }"
            >
              <template #icon>
                <span class="material-icons">restart_alt</span>
              </template>
              {{ t("exercise.plan.restartBlock") }}
            </NeoButton>
          </div>
        </div>
      </div>
    </NeoPanel>
  </div>
</template>

<script setup>
import { ref, watch, computed } from "vue";
import { useI18n } from "vue-i18n";
import { getSetting } from "@/utils/database.js";
import {
  getBlockDisplayData,
  startNewBlock,
  updateExerciseSettings,
  getGlobalSettings,
  getBestEstimated1RM,
} from "@/utils/blockPeriodization.js";
import { getWorkouts } from "@/utils/database.js";
import NeoPanel from "@/components/NeoPanel.vue";
import NeoButton from "@/components/NeoButton.vue";

const props = defineProps({
  exercise: {
    type: Object,
    required: true,
  },
  isOpen: {
    type: Boolean,
    default: false,
  },
  arm: {
    type: String,
    default: "",
  },
  onApplyWeight: {
    type: Function,
    default: null,
  },
});

const emit = defineEmits(["close", "weightApplied", "blockStarted"]);

const { t } = useI18n();
const loading = ref(false);
const showSettings = ref(false);
const weightUnit = ref("kg");
const workouts = ref([]);
const blockData = ref({});
const estimated1RM = ref(0);

// Global settings
const globalSettings = ref({
  weeksPerBlock: 5,
  workoutsPerWeek: 1,
  startPercentage: 75,
  progressionPerWeek: 5,
});

// Per-exercise settings (0 means use global)
const exerciseWeeksPerBlock = ref(0);
const exerciseWorkoutsPerWeek = ref(0);
const exerciseStartPercentage = ref(0);
const exerciseProgressionPerWeek = ref(0);

// Start week for new blocks
const startWeek = ref(1);

// Computed total weeks based on settings
const effectiveTotalWeeks = computed(() => {
  return exerciseWeeksPerBlock.value > 0
    ? exerciseWeeksPerBlock.value
    : globalSettings.value.weeksPerBlock;
});

// Check if exercise has an override for a setting
const hasExerciseOverride = (setting) => {
  const blockState = blockData.value?.blockState;
  if (!blockState) return false;
  return blockState[setting] !== null && blockState[setting] !== undefined;
};

const closeModal = () => {
  showSettings.value = false;
  emit("close");
};

const loadData = async () => {
  loading.value = true;
  try {
    const [workoutsData, weightSetting, global] = await Promise.all([
      getWorkouts(),
      getSetting("weightUnit", "kg"),
      getGlobalSettings(),
    ]);

    weightUnit.value = weightSetting;
    workouts.value = workoutsData;
    globalSettings.value = global;

    // Calculate estimated 1RM
    estimated1RM.value = getBestEstimated1RM(
      workoutsData,
      props.exercise.name,
      props.arm,
    );

    // Load block display data
    blockData.value = await getBlockDisplayData(
      props.exercise.name,
      workoutsData,
      props.arm,
    );

    // Load per-exercise settings into form
    const blockState = blockData.value.blockState;
    if (blockState) {
      exerciseWeeksPerBlock.value = blockState.weeksPerBlock || 0;
      exerciseWorkoutsPerWeek.value = blockState.workoutsPerWeek || 0;
      exerciseStartPercentage.value = blockState.startPercentage || 0;
      exerciseProgressionPerWeek.value = blockState.progressionPerWeek || 0;
    }
  } catch (error) {
    console.error("Error loading plan data:", error);
  } finally {
    loading.value = false;
  }
};

const handleStartNewBlock = async () => {
  if (estimated1RM.value <= 0) return;

  const settings = {};
  if (exerciseWeeksPerBlock.value > 0)
    settings.weeksPerBlock = exerciseWeeksPerBlock.value;
  if (exerciseWorkoutsPerWeek.value > 0)
    settings.workoutsPerWeek = exerciseWorkoutsPerWeek.value;
  if (exerciseStartPercentage.value > 0)
    settings.startPercentage = exerciseStartPercentage.value;
  if (exerciseProgressionPerWeek.value > 0)
    settings.progressionPerWeek = exerciseProgressionPerWeek.value;

  // Include start week if not week 1
  if (startWeek.value > 1) {
    settings.startWeek = startWeek.value;
  }

  await startNewBlock(props.exercise.name, estimated1RM.value, settings);

  // Reset start week for next time
  startWeek.value = 1;

  // Reload data
  await loadData();

  emit("blockStarted", props.exercise.name);
};

const saveExerciseSettings = async () => {
  const settings = {
    weeksPerBlock: exerciseWeeksPerBlock.value || null,
    workoutsPerWeek: exerciseWorkoutsPerWeek.value || null,
    startPercentage: exerciseStartPercentage.value || null,
    progressionPerWeek: exerciseProgressionPerWeek.value || null,
  };

  await updateExerciseSettings(props.exercise.name, settings);

  // Reload block data to reflect new settings
  blockData.value = await getBlockDisplayData(
    props.exercise.name,
    workouts.value,
    props.arm,
  );
};

const applyWeight = () => {
  if (props.onApplyWeight && blockData.value.recommendedWeight > 0) {
    props.onApplyWeight(blockData.value.recommendedWeight);
    emit("weightApplied", blockData.value.recommendedWeight);
    closeModal();
  }
};

// Watch for modal open
watch(
  () => props.isOpen,
  async (isOpen) => {
    if (isOpen) {
      showSettings.value = false;
      startWeek.value = 1; // Reset start week
      await loadData();
    }
  },
  { immediate: true },
);
</script>
