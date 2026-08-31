<template>
  <div class="workout-edit h-full flex flex-col">
    <!-- Header -->
    <NeoHeader>
      <template #left>
        <NeoButton
          variant="primary"
          size="sm"
          data-testid="back-button"
          @click="goBack"
        >
          <template #icon>
            <span class="material-icons">arrow_back</span>
          </template>
        </NeoButton>
      </template>

      <template #middle>
        <h1>
          {{ isNew ? t("workout.new") : t("workout.edit") }}
        </h1>
      </template>

      <template #right>
        <NeoButton
          v-if="!isNew"
          variant="overlay"
          size="sm"
          @click="showContextMenu = true"
        >
          <template #icon>
            <span class="material-icons">more_vert</span>
          </template>
        </NeoButton>
        <!-- Auto-save indicator -->
        <div
          v-if="saving"
          class="flex items-center space-x-2 text-sm font-semibold text-black dark:text-white bg-nb-bg border-2 border-nb-border rounded-md px-3 py-1 shadow-brutal-sm"
        >
          <div
            class="w-3 h-3 border-2 border-nb-text border-t-transparent rounded-full animate-spin"
          ></div>
          <span>{{ t("workout.saving") }}</span>
        </div>
      </template>
    </NeoHeader>

    <!-- Context Menu -->
    <div
      v-if="showContextMenu"
      class="fixed inset-0 z-50 flex items-end justify-center"
      style="background-color: rgba(1, 0, 0, 0.4)"
      @click="showContextMenu = false"
    >
      <NeoPanel class="w-full max-w-md safe-area-bottom" @click.stop>
        <div class="space-y-3">
          <NeoButton
            v-if="!isTemplate"
            variant="secondary"
            full-width
            @click="saveAsTemplate"
          >
            <template #icon>
              <span class="material-icons">save_as</span>
            </template>
            {{ t("workout.saveAsTemplate") }}
          </NeoButton>
          <NeoButton
            v-if="!isTemplate"
            variant="primary"
            full-width
            @click="duplicateWorkout"
          >
            <template #icon>
              <span class="material-icons">content_copy</span>
            </template>
            {{ t("workout.duplicate") }}
          </NeoButton>
          <DestructiveButton
            :confirm-text="
              isTemplate ? t('templates.delete') : t('workout.delete')
            "
            full-width
            @confirm="deleteWorkoutConfirmed"
          >
            <template #icon>
              <span class="material-icons">delete</span>
            </template>
            {{ isTemplate ? t("templates.delete") : t("workout.delete") }}
          </DestructiveButton>
        </div>
        <NeoButton
          variant="overlay"
          full-width
          class="mt-4"
          @click="showContextMenu = false"
        >
          {{ t("common.cancel") }}
        </NeoButton>
      </NeoPanel>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-auto hide-scrollbar">
      <div class="p-4 space-y-6">
        <!-- Workout Details -->
        <NeoPanel class="space-y-6">
          <!-- Workout Name -->
          <div class="floating-label-container">
            <input
              id="workout-name"
              v-model="workout.name"
              type="text"
              class="floating-input"
              :placeholder="t('workout.namePlaceholder')"
            />
            <label for="workout-name" class="floating-label">
              {{ t("workout.name") }}
            </label>
          </div>

          <!-- Started/Ended Date Fields (only for workouts, not templates) -->
          <div v-if="!isTemplate" class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div class="floating-label-container">
              <input
                id="workout-started"
                v-model="startedDatetime"
                type="datetime-local"
                class="floating-input datetime-fixed"
              />
              <label
                for="workout-started"
                class="floating-label datetime-label-fixed"
              >
                {{ t("workout.started") }}
              </label>
            </div>

            <div class="floating-label-container">
              <input
                id="workout-ended"
                v-model="endedDatetime"
                type="datetime-local"
                class="floating-input datetime-fixed"
              />
              <label
                for="workout-ended"
                class="floating-label datetime-label-fixed"
              >
                {{ t("workout.ended") }}
              </label>
            </div>
          </div>

          <!-- Workout Notes -->
          <div class="floating-label-container">
            <textarea
              id="workout-notes"
              v-model="workout.notes"
              rows="3"
              class="floating-textarea"
              :placeholder="t('workout.notesPlaceholder')"
            ></textarea>
            <label for="workout-notes" class="floating-label">
              {{ t("workout.notes") }}
            </label>
          </div>
        </NeoPanel>

        <!-- Exercises -->
        <div class="space-y-6">
          <NeoPanel
            v-for="(exercise, exerciseIndex) in workout.exercises"
            :key="exerciseIndex"
            padding="none"
            class="relative"
          >
            <!-- Sticky Exercise Header -->
            <div
              class="sticky top-0 z-10 bg-nb-bg dark:bg-zinc-700 border-b-2 border-nb-border pb-4"
            >
              <div class="flex items-center justify-between gap-2">
                <div class="min-w-0">
                  <div
                    class="text-xl font-bold text-black dark:text-white truncate"
                  >
                    {{ exercise.name }}
                  </div>
                  <!-- Intensity toggle -->
                  <button
                    type="button"
                    :class="[
                      'mt-1 inline-flex items-center gap-1 border-2 border-nb-border rounded-full px-2 py-0.5 text-xs font-bold shadow-brutal-sm',
                      intensityClass(exercise.intensity),
                    ]"
                    :title="t('exercise.intensity.label')"
                    :data-testid="`intensity-toggle-${exerciseIndex}`"
                    @click="cycleIntensity(exerciseIndex)"
                  >
                    <span class="material-icons text-sm">bolt</span>
                    {{
                      t(`exercise.intensity.${exercise.intensity || "any"}`)
                    }}
                  </button>
                </div>
                <div class="flex items-center space-x-2 shrink-0">
                  <NeoButton
                    v-if="exercise.type !== 'cardio'"
                    variant="secondary"
                    size="sm"
                    class="w-8 h-8 !px-0 !py-0 rounded-full"
                    :title="t('exercise.plan.buttonTitle')"
                    @click="openExercisePlan(exerciseIndex)"
                  >
                    <template #icon>
                      <span class="material-icons">timeline</span>
                    </template>
                  </NeoButton>
                  <NeoButton
                    variant="secondary"
                    size="sm"
                    class="w-8 h-8 !px-0 !py-0 rounded-full"
                    @click="openExerciseStats(exerciseIndex)"
                  >
                    <template #icon>
                      <span class="material-icons">bar_chart</span>
                    </template>
                  </NeoButton>
                  <DestructiveButton
                    :confirm-text="t('workout.delete')"
                    size="sm"
                    class="w-8 h-8 !px-0 !py-0 rounded-full"
                    icon-only
                    @confirm="removeExercise(exerciseIndex)"
                  >
                    <template #icon>
                      <span class="material-icons">delete</span>
                    </template>
                  </DestructiveButton>
                </div>
              </div>
            </div>

            <!-- Last time summary -->
            <div
              v-if="!isTemplate && getLastTimeSummary(exercise)"
              class="mt-3 px-1 text-sm text-black dark:text-white"
            >
              <span class="font-semibold opacity-70">
                {{
                  t("exercise.lastTime", {
                    date: d(getLastTimeSummary(exercise).date, "short"),
                  })
                }}
              </span>
              <span class="ml-1 font-bold text-purple-600 dark:text-purple-400">
                {{ getLastTimeSummary(exercise).sets }}
              </span>
            </div>

            <!-- Scrollable Sets Area -->
            <div class="pt-4 px-[2px]">
              <div class="flex flex-col gap-3">
                <!-- Use Cardio Set Editor for cardio exercises -->
                <template v-if="exercise.type === 'cardio'">
                  <CardioSetEditor
                    v-for="(set, setIndex) in exercise.sets"
                    :key="`cardio-${setIndex}`"
                    :ref="(el) => registerSetEditor(exerciseIndex, setIndex, el)"
                    :set="set"
                    :exercise="exercise"
                    :exercise-index="exerciseIndex"
                    :set-index="setIndex"
                    :set-number="
                      getSetNumber(exercise.sets, setIndex, exercise, set)
                    "
                    :distance-unit="distanceUnit"
                    @toggle-set-type="toggleSetType(exerciseIndex, setIndex)"
                    @update:distance="
                      updateSetField(
                        exerciseIndex,
                        setIndex,
                        'distance',
                        $event,
                      )
                    "
                    @update:time="
                      updateSetField(exerciseIndex, setIndex, 'time', $event)
                    "
                    @update:rpe="
                      updateSetField(exerciseIndex, setIndex, 'rpe', $event)
                    "
                    @update:notes="
                      updateSetField(exerciseIndex, setIndex, 'notes', $event)
                    "
                    @add-set="addSet(exerciseIndex)"
                    @delete-set="deleteSet(exerciseIndex, setIndex)"
                  />
                </template>

                <!-- Use Strength Set Editor for strength exercises -->
                <template v-else>
                  <StrengthSetEditor
                    v-for="(set, setIndex) in exercise.sets"
                    :key="`strength-${setIndex}`"
                    :ref="(el) => registerSetEditor(exerciseIndex, setIndex, el)"
                    :set="set"
                    :exercise="exercise"
                    :exercise-index="exerciseIndex"
                    :set-index="setIndex"
                    :set-number="
                      getSetNumber(exercise.sets, setIndex, exercise, set)
                    "
                    :weight-unit="weightUnit"
                    :is-weight-record="
                      set.weight &&
                      isWeightRecord(exercise.name, set.weight, set.arm)
                    "
                    :is-rep-record="
                      set.reps &&
                      set.weight &&
                      isRepRecord(exercise.name, set.weight, set.reps, set.arm)
                    "
                    :previous-reps="
                      set.weight
                        ? getPreviousReps(exercise, set.weight, set.arm)
                        : null
                    "
                    :highest-reps="
                      set.weight
                        ? getHighestReps(exercise.name, set.weight, set.arm)
                        : null
                    "
                    :max-percentage="
                      getMaxPercentage(exercise.name, set.weight, set.arm)
                    "
                    @toggle-set-type="toggleSetType(exerciseIndex, setIndex)"
                    @update:weight="
                      updateSetField(exerciseIndex, setIndex, 'weight', $event)
                    "
                    @update:reps="
                      updateSetField(exerciseIndex, setIndex, 'reps', $event)
                    "
                    @update:time="
                      updateSetField(exerciseIndex, setIndex, 'time', $event)
                    "
                    @update:rpe="
                      updateSetField(exerciseIndex, setIndex, 'rpe', $event)
                    "
                    @update:arm="
                      updateSetField(exerciseIndex, setIndex, 'arm', $event)
                    "
                    @update:notes="
                      updateSetField(exerciseIndex, setIndex, 'notes', $event)
                    "
                    @add-set="addSet(exerciseIndex)"
                    @delete-set="deleteSet(exerciseIndex, setIndex)"
                  />
                </template>
              </div>

              <!-- Total Volume Display -->
              <div
                v-if="
                  exercise.type === 'strength' && getTotalVolume(exercise) > 0
                "
                class="mt-3 text-right"
              >
                <div class="text-md">
                  {{ t("exercise.totalVolume") }}
                  <span class="font-bold text-purple-600 dark:text-purple-400">{{ getTotalVolume(exercise) }} {{ weightUnit }}</span>
                </div>
              </div>

              <!-- Add Set Button -->
              <NeoButton
                variant="primary"
                class="mt-4 w-full"
                @click="addSet(exerciseIndex)"
              >
                <template #icon>
                  <span class="material-icons">add</span>
                </template>
                {{ t("workout.addSet") }}
              </NeoButton>
            </div>
          </NeoPanel>

          <!-- Add Exercise Button -->
          <NeoButton
            variant="primary"
            size="lg"
            full-width
            data-testid="add-exercise-button"
            @click="showExerciseSelector = true"
          >
            <template #icon>
              <span class="material-icons">add</span>
            </template>
            {{ t("workout.addExercise") }}
          </NeoButton>
        </div>
      </div>
    </div>

    <!-- Exercise Selector Modal -->
    <ExerciseSelector
      v-if="showExerciseSelector"
      :workout-name="workout.name"
      :workout-id="workout.id"
      @close="showExerciseSelector = false"
      @select="addExercise"
    />

    <!-- Exercise Stats Modal -->
    <ExerciseStats
      v-if="selectedExerciseForStats"
      :exercise="selectedExerciseForStats"
      :is-open="showExerciseStats"
      @close="
        showExerciseStats = false;
        selectedExerciseForStats = null;
      "
    />

    <!-- Exercise Plan Modal -->
    <ExercisePlanModal
      v-if="selectedExerciseForPlan"
      :exercise="selectedExerciseForPlan"
      :arm="selectedExerciseArmForPlan"
      :is-open="showExercisePlan"
      :on-apply-weight="applyPlanWeight"
      @close="
        showExercisePlan = false;
        selectedExerciseForPlan = null;
      "
    />
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed, watch, nextTick } from "vue";
import { useRouter, useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import { useHead } from "@unhead/vue";
import {
  getWorkout,
  getWorkoutTemplate,
  saveWorkout as saveWorkoutToDB,
  deleteWorkout,
  deleteWorkoutTemplate,
  saveWorkoutTemplate,
} from "@/utils/database.js";
import ExerciseSelector from "@/components/ExerciseSelector.vue";
import ExerciseStats from "@/components/ExerciseStats.vue";
import ExercisePlanModal from "@/components/ExercisePlanModal.vue";
import NeoButton from "@/components/NeoButton.vue";
import NeoPanel from "@/components/NeoPanel.vue";
import NeoHeader from "@/components/NeoHeader.vue";
import DestructiveButton from "@/components/DestructiveButton.vue";
import StrengthSetEditor from "@/components/StrengthSetEditor.vue";
import CardioSetEditor from "@/components/CardioSetEditor.vue";
import { useToast } from "@/composables/useToast.js";
import { useUnits } from "@/composables/useUnits.js";
import { useExerciseHistory } from "@/composables/useExerciseHistory.js";
import { INTENSITIES, formatEntrySets } from "@/utils/exerciseHistory.js";
import { incrementBlockWorkout } from "@/utils/blockPeriodization.js";

const router = useRouter();
const route = useRoute();
const { t, d } = useI18n();
const { showSuccess, showError } = useToast();

const props = defineProps({
  id: {
    type: String,
    default: null,
  },
});

const workout = ref({
  id: null,
  name: "",
  started: new Date(),
  ended: null,
  notes: "",
  exercises: [],
});

const saving = ref(false);
const showContextMenu = ref(false);
const showExerciseSelector = ref(false);
const showExerciseStats = ref(false);
const selectedExerciseForStats = ref(null);
const showExercisePlan = ref(false);
const selectedExerciseForPlan = ref(null);
const selectedExerciseArmForPlan = ref("");
const isInitialLoad = ref(true);
const setEditorRefs = new Map();
let saveTimeout = null;
let pendingSave = false;

const { weightUnit, distanceUnit } = useUnits();
const {
  loadHistory,
  getMaxPercentage,
  getPreviousReps,
  getHighestReps,
  isRepRecord,
  isWeightRecord,
  getLastTime,
} = useExerciseHistory(workout);

const isNew = computed(() => !props.id);
const isTemplate = computed(() => route.name === "template-edit");

// Set dynamic page title
useHead({
  title: () => {
    const entityName =
      workout.value.name ||
      t(isTemplate.value ? "templates.unnamed" : "workout.unnamed");
    const newKey = isTemplate.value ? "templates.new" : "workout.new";
    const editKey = isTemplate.value ? "templates.edit" : "workout.edit";
    const prefix = isNew.value ? t(newKey) : t(editKey);
    return `${prefix}${isNew.value ? "" : `: ${entityName}`}`;
  },
});

/**
 * Computed property for started datetime input
 */
const startedDatetime = computed({
  get: () => formatDatetimeLocal(workout.value.started),
  set: (value) => {
    workout.value.started = value ? new Date(value) : new Date();
  },
});

/**
 * Computed property for ended datetime input
 */
const endedDatetime = computed({
  get: () =>
    workout.value.ended ? formatDatetimeLocal(workout.value.ended) : "",
  set: (value) => {
    workout.value.ended = value ? new Date(value) : null;
  },
});

/**
 * Format date for datetime-local input
 * @param {Date} date - The date to format
 * @returns {string} Formatted datetime string
 */
function formatDatetimeLocal(date) {
  if (!date) return "";
  const d = new Date(date);
  d.setMinutes(d.getMinutes() - d.getTimezoneOffset());
  return d.toISOString().slice(0, 16);
}

/**
 * Load existing workout or template data
 */
async function loadWorkout() {
  if (props.id) {
    try {
      const data = isTemplate.value
        ? await getWorkoutTemplate(parseInt(props.id))
        : await getWorkout(parseInt(props.id));
      if (data) {
        workout.value = {
          ...data,
          started: data.started ? new Date(data.started) : new Date(),
          ended: data.ended ? new Date(data.ended) : null,
        };
      }
    } catch (error) {
      console.error(
        isTemplate.value ? t("templates.loadError") : t("workout.loadError"),
        error,
      );
    }
  }
}

/**
 * Save workout or template to database
 */
async function saveWorkout() {
  // A save is already running, remember to run again once it finishes so
  // edits made during the save are not lost
  if (saving.value) {
    pendingSave = true;
    return;
  }

  const wasNew = isNew.value;

  try {
    saving.value = true;
    const rawData = {
      ...workout.value,
      updated: new Date(),
    };

    // Remove the id field so let the database generate it
    if (wasNew) {
      delete rawData.id;
    }

    const data = JSON.parse(JSON.stringify(rawData));

    let id;
    if (isTemplate.value) {
      id = await saveWorkoutTemplate(data);
      if (wasNew) {
        workout.value.id = id;
        router.replace({
          name: "template-edit",
          params: { id: id.toString() },
        });
      }
    } else {
      id = await saveWorkoutToDB(data);
      if (wasNew) {
        workout.value.id = id;
        router.replace({ name: "workout-edit", params: { id: id.toString() } });

        // Increment block workout count for each exercise in this new workout
        for (const exercise of workout.value.exercises) {
          if (exercise.name) {
            await incrementBlockWorkout(exercise.name);
          }
        }
      }
    }
  } catch (error) {
    console.error("Error saving:", error);
    showError(
      isTemplate.value ? t("templates.saveError") : t("workout.saveError"),
    );
  } finally {
    saving.value = false;
    if (pendingSave) {
      pendingSave = false;
      saveWorkout();
    }
  }
}

/**
 * Debounced auto-save function
 */
function debouncedSave() {
  // Skip auto-save during initial load
  if (isInitialLoad.value) return;

  // Skip auto-save if workout has no name and no exercises (empty workout)
  if (!workout.value.name.trim() && workout.value.exercises.length === 0) {
    return;
  }

  // Clear existing timeout
  if (saveTimeout) {
    clearTimeout(saveTimeout);
  }

  // Set new timeout for 1 second debounce
  saveTimeout = setTimeout(() => {
    saveWorkout();
  }, 1000);
}

/**
 * Add a new exercise to the workout
 * @param {Object} exercise - The exercise to add
 */
function addExercise(exercise) {
  workout.value.exercises.push({
    ...exercise,
    intensity: exercise.intensity || null,
    sets: [createNewSet()],
  });
  showExerciseSelector.value = false;
}

/**
 * Remove an exercise from the workout
 * @param {number} exerciseIndex - Index of exercise to remove
 */
function removeExercise(exerciseIndex) {
  workout.value.exercises.splice(exerciseIndex, 1);
}

/**
 * Delete a set from an exercise
 * @param {number} exerciseIndex - Index of the exercise
 * @param {number} setIndex - Index of the set to delete
 */
function deleteSet(exerciseIndex, setIndex) {
  const exercise = workout.value.exercises[exerciseIndex];

  // Don't delete if it's the only set in the exercise
  if (exercise.sets.length <= 1) {
    return;
  }

  exercise.sets.splice(setIndex, 1);
}

/**
 * Add a new set to an exercise
 * @param {number} exerciseIndex - Index of the exercise
 */
function addSet(exerciseIndex) {
  workout.value.exercises[exerciseIndex].sets.push(createNewSet());

  // Focus the first input of the new set once it is rendered
  nextTick(() => {
    const newSetIndex = workout.value.exercises[exerciseIndex].sets.length - 1;
    const editor = setEditorRefs.get(`${exerciseIndex}-${newSetIndex}`);
    if (typeof editor?.focus === "function") editor.focus();
  });
}

/**
 * Track set editor component instances so new sets can be focused
 * @param {number} exerciseIndex - Index of the exercise
 * @param {number} setIndex - Index of the set
 * @param {Object|null} el - Component instance, or null when unmounted
 */
function registerSetEditor(exerciseIndex, setIndex, el) {
  const key = `${exerciseIndex}-${setIndex}`;
  if (el) {
    setEditorRefs.set(key, el);
  } else {
    setEditorRefs.delete(key);
  }
}

/**
 * Cycle an exercise's intensity tag: none -> heavy -> light -> none
 * @param {number} exerciseIndex - Index of the exercise
 */
function cycleIntensity(exerciseIndex) {
  const exercise = workout.value.exercises[exerciseIndex];
  const current = INTENSITIES.indexOf(exercise.intensity);
  exercise.intensity =
    current === INTENSITIES.length - 1 ? null : INTENSITIES[current + 1];
}

/**
 * Tailwind classes for the intensity pill
 * @param {string|null} intensity
 * @returns {string}
 */
function intensityClass(intensity) {
  if (intensity === "heavy") return "bg-red-400 text-black";
  if (intensity === "light") return "bg-sky-300 text-black";
  return "bg-nb-overlay text-black opacity-60 dark:bg-zinc-800 dark:text-white";
}

/**
 * Summary of the last time this exercise was done (within the compare scope)
 * @param {Object} exercise - The exercise
 * @returns {{ date: Date, sets: string, intensity: string|null }|null}
 */
function getLastTimeSummary(exercise) {
  const entry = getLastTime(exercise);
  if (!entry) return null;
  const sets = formatEntrySets(entry, { type: exercise.type });
  return sets ? { date: entry.date, sets, intensity: entry.intensity } : null;
}

/**
 * Create a new empty set
 * @returns {Object} New set object
 */
function createNewSet() {
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
 * Toggle set type between regular and warmup
 * @param {number} exerciseIndex - Index of the exercise
 * @param {number} setIndex - Index of the set
 */
function toggleSetType(exerciseIndex, setIndex) {
  const set = workout.value.exercises[exerciseIndex].sets[setIndex];
  set.type = set.type === "warmup" ? "regular" : "warmup";
}

/**
 * Update a specific field in a set
 * @param {number} exerciseIndex - Index of the exercise
 * @param {number} setIndex - Index of the set
 * @param {string} field - Field name to update
 * @param {*} value - New value for the field
 */
function updateSetField(exerciseIndex, setIndex, field, value) {
  workout.value.exercises[exerciseIndex].sets[setIndex][field] = value;
}

/**
 * Get display number for a set
 * @param {Array} sets - All sets for the exercise
 * @param {number} setIndex - Index of the current set
 * @param {Object} exercise - The exercise object
 * @param {Object} currentSet - The current set object
 * @returns {string} Set number display
 */
function getSetNumber(sets, setIndex, exercise, currentSet) {
  const set = sets[setIndex];

  if (set.type === "warmup") {
    // For warmup sets, count per arm for single-arm exercises
    if (exercise.singleArm && currentSet.arm) {
      const warmupSetsForArm = sets
        .slice(0, setIndex + 1)
        .filter((s) => s.type === "warmup" && s.arm === currentSet.arm);
      const armSuffix =
        currentSet.arm === "left" ? "L" : currentSet.arm === "right" ? "R" : "";
      return `W${warmupSetsForArm.length}${armSuffix}`;
    } else {
      // For non-single arm exercises, count all warmup sets
      const warmupSets = sets
        .slice(0, setIndex + 1)
        .filter((s) => s.type === "warmup");
      return `W${warmupSets.length}`;
    }
  } else {
    // For regular sets
    if (exercise.singleArm && currentSet.arm) {
      // Count sets for the specific arm
      const armSets = sets
        .slice(0, setIndex + 1)
        .filter((s) => s.type === "regular" && s.arm === currentSet.arm);
      const armSuffix =
        currentSet.arm === "left" ? "L" : currentSet.arm === "right" ? "R" : "";
      return `${armSets.length}${armSuffix}`;
    } else {
      // For non-single arm exercises or sets without arm specified
      const regularSets = sets
        .slice(0, setIndex + 1)
        .filter((s) => s.type === "regular");
      return `${regularSets.length}`;
    }
  }
}

/**
 * Calculate total volume for an exercise (sum of reps * weight for all regular sets)
 * @param {Object} exercise - The exercise object
 * @returns {number} - Total volume
 */
function getTotalVolume(exercise) {
  if (!exercise.sets || exercise.type !== "strength") {
    return 0;
  }

  return exercise.sets
    .filter((set) => set.type === "regular" || !set.type) // Include sets without type (default to regular)
    .reduce((total, set) => {
      const weight = parseFloat(set.weight) || 0;
      const reps = parseInt(set.reps) || 0;
      return total + weight * reps;
    }, 0);
}

/**
 * Show exercise statistics
 * @param {number} exerciseIndex - Index of the exercise
 */
function openExerciseStats(exerciseIndex) {
  selectedExerciseForStats.value = workout.value.exercises[exerciseIndex];
  showExerciseStats.value = true;
}

/**
 * Show exercise plan modal
 * @param {number} exerciseIndex - Index of the exercise
 */
function openExercisePlan(exerciseIndex) {
  const exercise = workout.value.exercises[exerciseIndex];
  selectedExerciseForPlan.value = exercise;
  // Determine arm for single-arm exercises
  const hasArmSets = exercise.sets?.some((s) => s.arm);
  selectedExerciseArmForPlan.value = hasArmSets
    ? exercise.sets?.find((s) => s.arm)?.arm || ""
    : "";
  showExercisePlan.value = true;
}

/**
 * Apply recommended weight to first empty set
 * @param {number} weight - Weight to apply
 */
function applyPlanWeight(weight) {
  if (!selectedExerciseForPlan.value) return;

  const exerciseIndex = workout.value.exercises.findIndex(
    (ex) => ex === selectedExerciseForPlan.value,
  );
  if (exerciseIndex === -1) return;

  const exercise = workout.value.exercises[exerciseIndex];

  // Find first set without weight
  const emptySetIndex = exercise.sets?.findIndex(
    (s) => !s.weight || s.weight === 0,
  );

  if (emptySetIndex !== undefined && emptySetIndex >= 0) {
    updateSetField(exerciseIndex, emptySetIndex, "weight", weight);
  } else if (exercise.sets && exercise.sets.length > 0) {
    // If no empty set, update the last set
    updateSetField(exerciseIndex, exercise.sets.length - 1, "weight", weight);
  }
}

/**
 * Duplicate the current workout
 */
async function duplicateWorkout() {
  showContextMenu.value = false;

  const duplicatedWorkout = {
    name: workout.value.name || "",
    started: new Date(),
    ended: null,
    notes: workout.value.notes,
    exercises: workout.value.exercises.map((exercise) => ({
      ...exercise,
      sets: [createNewSet()],
    })),
  };

  try {
    const id = await saveWorkoutToDB(duplicatedWorkout);
    router.push({ name: "workout-edit", params: { id: id.toString() } });
  } catch (error) {
    console.error("Error duplicating workout:", error);
    showError(t("workout.duplicateError"));
  }
}

/**
 * Save current workout as template
 */
async function saveAsTemplate() {
  showContextMenu.value = false;

  try {
    const templateData = {
      name: workout.value.name || t("workout.unnamed"),
      notes: workout.value.notes,
      exercises: workout.value.exercises.map((exercise) => ({
        ...exercise,
        sets: exercise.sets.map((set) => ({
          type: set.type,
          weight: null, // Clear weight values for template
          reps: null, // Clear reps values for template
          rpe: set.rpe,
          arm: set.arm,
          notes: "", // Clear notes for template
        })),
      })),
    };

    await saveWorkoutTemplate(templateData);
    showSuccess(t("workout.templateSaved"));
  } catch (error) {
    console.error("Error saving template:", error);
    showError(t("workout.saveError"));
  }
}

/**
 * Delete workout or template (called from DestructiveButton after confirmation)
 */
async function deleteWorkoutConfirmed() {
  showContextMenu.value = false;

  try {
    if (isTemplate.value) {
      await deleteWorkoutTemplate(parseInt(props.id));
      router.push({ name: "templates" });
    } else {
      await deleteWorkout(parseInt(props.id));
      router.push({ name: "log" });
    }
  } catch (error) {
    console.error("Error deleting:", error);
    showError(
      isTemplate.value ? t("templates.deleteError") : t("workout.deleteError"),
    );
  }
}

/**
 * Go back to workout list or templates list
 */
function goBack() {
  router.push({ name: isTemplate.value ? "templates" : "log" });
}

onMounted(async () => {
  await Promise.all([loadHistory(), loadWorkout()]);

  // Allow auto-save after initial load is complete
  await nextTick();
  isInitialLoad.value = false;
});

// Watch for changes in workout data and auto-save
watch(
  workout,
  () => {
    debouncedSave();
  },
  { deep: true },
);

// Cleanup timeout on component unmount
onUnmounted(() => {
  if (saveTimeout) {
    clearTimeout(saveTimeout);
  }
});
</script>
