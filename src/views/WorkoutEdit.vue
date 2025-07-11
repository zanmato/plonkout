<template>
  <div class="workout-edit h-full flex flex-col">
    <!-- Header -->
    <NeoHeader>
      <template #left>
        <NeoButton
          @click="goBack"
          variant="primary"
          size="sm"
          data-testid="back-button"
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
          @click="showContextMenu = true"
          variant="overlay"
          size="sm"
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
      @click="showContextMenu = false"
      class="fixed inset-0 z-50 flex items-end justify-center"
      style="background-color: rgba(0, 0, 0, 0.4)"
    >
      <NeoPanel @click.stop class="w-full max-w-md safe-area-bottom">
        <div class="space-y-3">
          <NeoButton @click="saveAsTemplate" variant="secondary" full-width>
            <template #icon>
              <span class="material-icons">save_as</span>
            </template>
            {{ t("workout.saveAsTemplate") }}
          </NeoButton>
          <NeoButton @click="duplicateWorkout" variant="primary" full-width>
            <template #icon>
              <span class="material-icons">content_copy</span>
            </template>
            {{ t("workout.duplicate") }}
          </NeoButton>
          <DestructiveButton
            @confirm="deleteWorkoutConfirmed"
            :confirm-text="t('workout.delete')"
            full-width
          >
            <template #icon>
              <span class="material-icons">delete</span>
            </template>
            {{ t("workout.delete") }}
          </DestructiveButton>
        </div>
        <NeoButton
          @click="showContextMenu = false"
          variant="overlay"
          full-width
          class="mt-4"
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
              v-model="workout.name"
              type="text"
              id="workout-name"
              class="floating-input"
              :placeholder="t('workout.namePlaceholder')"
            />
            <label for="workout-name" class="floating-label">
              {{ t("workout.name") }}
            </label>
          </div>

          <!-- Started/Ended Date Fields -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div class="floating-label-container">
              <input
                v-model="startedDatetime"
                type="datetime-local"
                id="workout-started"
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
                v-model="endedDatetime"
                type="datetime-local"
                id="workout-ended"
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
              v-model="workout.notes"
              rows="3"
              id="workout-notes"
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
            padding="md"
          >
            <!-- Exercise Header -->
            <div class="flex items-center justify-between mb-5">
              <div class="text-xl font-bold text-black dark:text-white">
                {{ exercise.name }}
              </div>
              <div class="flex items-center space-x-2">
                <NeoButton
                  @click="openExerciseStats(exerciseIndex)"
                  variant="secondary"
                  size="sm"
                  class="w-8 h-8 !px-0 !py-0 rounded-full"
                >
                  <template #icon>
                    <span class="material-icons">bar_chart</span>
                  </template>
                </NeoButton>
                <DestructiveButton
                  @confirm="removeExercise(exerciseIndex)"
                  :confirm-text="t('workout.delete')"
                  size="sm"
                  class="w-8 h-8 !px-0 !py-0 rounded-full"
                  icon-only
                >
                  <template #icon>
                    <span class="material-icons">delete</span>
                  </template>
                </DestructiveButton>
              </div>
            </div>

            <div class="flex flex-col gap-3">
              <!-- Use Cardio Set Editor for cardio exercises -->
              <template v-if="exercise.type === 'cardio'">
                <CardioSetEditor
                  v-for="(set, setIndex) in exercise.sets"
                  :key="`cardio-${setIndex}`"
                  :set="set"
                  :exercise="exercise"
                  :set-number="
                    getSetNumber(exercise.sets, setIndex, exercise, set)
                  "
                  :distance-unit="distanceUnit"
                  :rpe-options="rpeOptions"
                  @toggle-set-type="toggleSetType(exerciseIndex, setIndex)"
                  @update:distance="
                    updateSetField(exerciseIndex, setIndex, 'distance', $event)
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
                />
              </template>

              <!-- Use Strength Set Editor for strength exercises -->
              <template v-else>
                <StrengthSetEditor
                  v-for="(set, setIndex) in exercise.sets"
                  :key="`strength-${setIndex}`"
                  :set="set"
                  :exercise="exercise"
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
                      ? getPreviousReps(exercise.name, set.weight, set.arm)
                      : null
                  "
                  :max-percentage="
                    getMaxPercentage(exercise.name, set.weight, set.arm)
                  "
                  :rpe-options="rpeOptions"
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
                />
              </template>
            </div>

            <!-- Add Set Button -->
            <NeoButton
              @click="addSet(exerciseIndex)"
              variant="primary"
              class="mt-4 w-full"
            >
              <template #icon>
                <span class="material-icons">add</span>
              </template>
              {{ t("workout.addSet") }}
            </NeoButton>
          </NeoPanel>

          <!-- Add Exercise Button -->
          <NeoButton
            @click="showExerciseSelector = true"
            variant="primary"
            size="lg"
            full-width
            data-testid="add-exercise-button"
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
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed, watch, nextTick } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useHead } from "@unhead/vue";
import {
  getWorkout,
  getWorkouts,
  saveWorkout as saveWorkoutToDB,
  deleteWorkout,
  saveWorkoutTemplate,
  getSetting,
} from "@/utils/database.js";
import ExerciseSelector from "@/components/ExerciseSelector.vue";
import ExerciseStats from "@/components/ExerciseStats.vue";
import NeoButton from "@/components/NeoButton.vue";
import NeoPanel from "@/components/NeoPanel.vue";
import NeoHeader from "@/components/NeoHeader.vue";
import DestructiveButton from "@/components/DestructiveButton.vue";
import StrengthSetEditor from "@/components/StrengthSetEditor.vue";
import CardioSetEditor from "@/components/CardioSetEditor.vue";
import { useToast } from "@/composables/useToast.js";

const router = useRouter();
const { t } = useI18n();
const { showSuccess, showError } = useToast();

const props = defineProps({
  id: String,
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
const allWorkouts = ref([]);
const weightUnit = ref("kg");
const distanceUnit = ref("km");
const isInitialLoad = ref(true);
let saveTimeout = null;

const rpeOptions = [
  "RPE 1",
  "RPE 2",
  "RPE 3",
  "RPE 4",
  "RPE 5",
  "RPE 6",
  "RPE 7",
  "RPE 8",
  "RPE 9",
  "RPE 10",
];

const isNew = computed(() => !props.id);

// Set dynamic page title
useHead({
  title: () => {
    const workoutName = workout.value.name || t("workout.unnamed");
    const prefix = isNew.value ? t("workout.new") : t("workout.edit");
    return `${prefix}${isNew.value ? "" : `: ${workoutName}`}`;
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
 * Load existing workout data
 */
async function loadWorkout() {
  if (props.id) {
    try {
      const data = await getWorkout(parseInt(props.id));
      if (data) {
        workout.value = {
          ...data,
          started: new Date(data.started),
          ended: data.ended ? new Date(data.ended) : null,
        };
      }
    } catch (error) {
      console.error(t("workout.loadError"), error);
    }
  }
}

/**
 * Load weight unit setting
 */
async function loadWeightUnit() {
  try {
    const unit = await getSetting("weightUnit", "kg");
    weightUnit.value = unit;
  } catch (error) {
    console.error("Error loading weight unit:", error);
  }
}

/**
 * Load distance unit setting
 */
async function loadDistanceUnit() {
  try {
    const unit = await getSetting("distanceUnit", "km");
    distanceUnit.value = unit;
  } catch (error) {
    console.error("Error loading distance unit:", error);
  }
}

/**
 * Load all workouts for max weight calculations
 */
async function loadAllWorkouts() {
  try {
    const data = await getWorkouts();
    allWorkouts.value = data;
  } catch (error) {
    console.error("Error loading all workouts:", error);
  }
}

/**
 * Save workout to database
 */
async function saveWorkout() {
  if (saving.value) return;

  try {
    saving.value = true;
    const workoutData = JSON.parse(
      JSON.stringify({
        ...workout.value,
        updated: new Date(),
      })
    );

    const id = await saveWorkoutToDB(workoutData);

    // Navigate if it's a new workout (ID will be set by route params)
    if (isNew.value) {
      workout.value.id = id;
      router.replace({ name: "workout-edit", params: { id: id.toString() } });
    }
  } catch (error) {
    console.error("Error saving workout:", error);
    showError(t("workout.saveError"));
  } finally {
    saving.value = false;
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
 * Add a new set to an exercise
 * @param {number} exerciseIndex - Index of the exercise
 */
function addSet(exerciseIndex) {
  workout.value.exercises[exerciseIndex].sets.push(createNewSet());
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
    rpe: "",
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
 * Show exercise statistics
 * @param {number} exerciseIndex - Index of the exercise
 */
function openExerciseStats(exerciseIndex) {
  selectedExerciseForStats.value = workout.value.exercises[exerciseIndex];
  showExerciseStats.value = true;
}

/**
 * Calculate the percentage of current weight vs max weight for an exercise
 * @param {string} exerciseName - Name of the exercise
 * @param {number} currentWeight - Current weight being lifted
 * @param {string} currentArm - Current arm setting ('left', 'right', 'both', or '')
 * @returns {string} Percentage string or '-' if no data
 */
function getMaxPercentage(exerciseName, currentWeight, currentArm) {
  if (!currentWeight || currentWeight <= 0) {
    return "-";
  }

  let maxWeight = 0;

  // Check all historical workouts
  allWorkouts.value.forEach((historicalWorkout) => {
    if (historicalWorkout.exercises) {
      historicalWorkout.exercises.forEach((exercise) => {
        if (exercise.name === exerciseName && exercise.sets) {
          exercise.sets.forEach((set) => {
            if (
              set.weight &&
              set.weight > maxWeight &&
              set.type === "regular"
            ) {
              // Only consider sets with the same arm setting or compatible arm settings
              if (isArmCompatible(currentArm, set.arm)) {
                maxWeight = set.weight;
              }
            }
          });
        }
      });
    }
  });

  // Also check current workout
  workout.value.exercises.forEach((exercise) => {
    if (exercise.name === exerciseName) {
      exercise.sets.forEach((set) => {
        if (set.weight && set.weight > maxWeight && set.type === "regular") {
          // Only consider sets with the same arm setting or compatible arm settings
          if (isArmCompatible(currentArm, set.arm)) {
            maxWeight = set.weight;
          }
        }
      });
    }
  });

  if (maxWeight === 0) {
    return "-";
  }

  const percentage = Math.round((currentWeight / maxWeight) * 100);
  return `${percentage}%`;
}

/**
 * Check if two arm settings are compatible for comparison
 * @param {string} currentArm - Current arm setting
 * @param {string} historicalArm - Historical arm setting to compare
 * @returns {boolean} Whether the arms are compatible for comparison
 */
function isArmCompatible(currentArm, historicalArm) {
  // If either arm is not specified, they're compatible
  if (!currentArm || !historicalArm) {
    return true;
  }

  // Exact match
  if (currentArm === historicalArm) {
    return true;
  }

  // 'both' is compatible with individual arms (since both arms working together might be stronger)
  if (
    historicalArm === "both" &&
    (currentArm === "left" || currentArm === "right")
  ) {
    return true;
  }

  // Individual arms are not compatible with 'both' (single arm max should not include both-arm sets)
  // Individual arms are not compatible with each other
  return false;
}

/**
 * Get the previous best reps for a given weight and exercise
 * @param {string} exerciseName - Name of the exercise
 * @param {number} weight - Weight to check reps for
 * @param {string} arm - Arm setting
 * @returns {number|null} Previous best reps or null if no data
 */
function getPreviousReps(exerciseName, weight, arm) {
  if (!weight || weight <= 0) {
    return null;
  }

  let bestReps = 0;

  // Check all historical workouts (excluding current workout)
  allWorkouts.value.forEach((historicalWorkout) => {
    // Skip the current workout being edited
    if (historicalWorkout.id === workout.value.id) {
      return;
    }

    if (historicalWorkout.exercises) {
      historicalWorkout.exercises.forEach((exercise) => {
        if (exercise.name === exerciseName && exercise.sets) {
          exercise.sets.forEach((set) => {
            if (set.weight === weight && set.reps && set.type === "regular") {
              if (isArmCompatible(arm, set.arm) && set.reps > bestReps) {
                bestReps = set.reps;
              }
            }
          });
        }
      });
    }
  });

  return bestReps > 0 ? bestReps : null;
}

/**
 * Check if current reps is a new record for the weight
 * @param {string} exerciseName - Name of the exercise
 * @param {number} weight - Weight being lifted
 * @param {number} reps - Reps being performed
 * @param {string} arm - Arm setting
 * @returns {boolean} Whether this is a new rep record
 */
function isRepRecord(exerciseName, weight, reps, arm) {
  if (!weight || !reps || weight <= 0 || reps <= 0) {
    return false;
  }

  const previousBest = getPreviousReps(exerciseName, weight, arm);
  const beatsHistoricalRecord = previousBest === null || reps > previousBest;

  if (!beatsHistoricalRecord) {
    return false;
  }

  // Check if this is the best reps for this weight in the current workout
  let currentWorkoutBest = 0;
  workout.value.exercises.forEach((exercise) => {
    if (exercise.name === exerciseName) {
      exercise.sets.forEach((set) => {
        if (set.weight === weight && set.reps && set.type === "regular") {
          if (isArmCompatible(arm, set.arm) && set.reps > currentWorkoutBest) {
            currentWorkoutBest = set.reps;
          }
        }
      });
    }
  });

  // Only show star if this set has the highest reps for this weight in current workout
  return reps === currentWorkoutBest;
}

/**
 * Check if current weight is a new record for the exercise
 * @param {string} exerciseName - Name of the exercise
 * @param {number} weight - Weight being lifted
 * @param {string} arm - Arm setting
 * @returns {boolean} Whether this is a new weight record
 */
function isWeightRecord(exerciseName, weight, arm) {
  if (!weight || weight <= 0) {
    return false;
  }

  let maxWeight = 0;

  // Check all historical workouts for max weight (excluding current workout)
  allWorkouts.value.forEach((historicalWorkout) => {
    // Skip the current workout being edited
    if (historicalWorkout.id === workout.value.id) {
      return;
    }

    if (historicalWorkout.exercises) {
      historicalWorkout.exercises.forEach((exercise) => {
        if (exercise.name === exerciseName && exercise.sets) {
          exercise.sets.forEach((set) => {
            if (
              set.weight &&
              set.weight > maxWeight &&
              set.type === "regular"
            ) {
              if (isArmCompatible(arm, set.arm)) {
                maxWeight = set.weight;
              }
            }
          });
        }
      });
    }
  });

  return weight > maxWeight;
}

/**
 * Duplicate the current workout
 */
async function duplicateWorkout() {
  showContextMenu.value = false;

  const duplicatedWorkout = {
    name: workout.value.name
      ? `${workout.value.name} (${t("workout.copyName")})`
      : "",
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
 * Delete workout (called from DestructiveButton after confirmation)
 */
async function deleteWorkoutConfirmed() {
  showContextMenu.value = false;

  try {
    await deleteWorkout(parseInt(props.id));
    router.push({ name: "log" });
  } catch (error) {
    console.error("Error deleting workout:", error);
    showError(t("workout.deleteError"));
  }
}

/**
 * Go back to workout list
 */
function goBack() {
  router.push({ name: "log" });
}

onMounted(async () => {
  await loadAllWorkouts();
  await loadWorkout();
  await loadWeightUnit();
  await loadDistanceUnit();

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
  { deep: true }
);

// Cleanup timeout on component unmount
onUnmounted(() => {
  if (saveTimeout) {
    clearTimeout(saveTimeout);
  }
});
</script>
