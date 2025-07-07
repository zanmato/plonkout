<template>
  <div class="workout-edit h-full flex flex-col">
    <!-- Header -->
    <NeoHeader>
      <template #left>
        <NeoButton @click="goBack" variant="primary" size="sm">
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
          class="flex items-center space-x-2 text-sm font-semibold text-nb-text bg-nb-bg border-2 border-nb-border rounded-md px-3 py-1 shadow-brutal-sm"
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
      class="fixed inset-0 bg-black bg-opacity-25 z-50 flex items-end justify-center"
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
          <NeoButton @click="confirmDelete" variant="danger" full-width>
            <template #icon>
              <span class="material-icons">delete</span>
            </template>
            {{ t("workout.delete") }}
          </NeoButton>
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
                class="floating-input"
              />
              <label for="workout-started" class="floating-label">
                {{ t("workout.started") }}
              </label>
            </div>

            <div class="floating-label-container">
              <input
                v-model="endedDatetime"
                type="datetime-local"
                id="workout-ended"
                class="floating-input"
              />
              <label for="workout-ended" class="floating-label">
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
                  @click="showExerciseStats(exerciseIndex)"
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
              <div
                v-for="(set, setIndex) in exercise.sets"
                :key="setIndex"
                class="bg-nb-overlay border-2 border-nb-border rounded-lg p-4 shadow-brutal-sm"
              >
                <div class="flex items-center justify-between mb-3">
                  <!-- Set Number -->
                  <button
                    @click="toggleSetType(exerciseIndex, setIndex)"
                    :class="[
                      'w-8 h-8 border-2 border-nb-border rounded-full flex items-center justify-center text-sm font-bold shadow-brutal-sm',
                      set.type === 'warmup'
                        ? 'bg-orange-500 text-white'
                        : 'bg-purple-400 text-black',
                    ]"
                  >
                    {{ getSetNumber(exercise.sets, setIndex, exercise, set) }}
                  </button>

                  <!-- Weight, Reps, RPE, and Arm -->
                  <div class="flex gap-4">
                    <div class="flex flex-col items-center text-center">
                      <div class="relative flex items-center">
                        <input
                          v-model.number="set.weight"
                          type="number"
                          step="0.5"
                          class="text-base font-bold text-nb-text bg-transparent border-none text-center w-16 focus:outline-none"
                          placeholder="0"
                        />
                      </div>
                      <div class="text-xs font-medium text-nb-text opacity-70">
                        {{ weightUnit }}

                        <span
                          v-if="
                            set.weight &&
                            isWeightRecord(exercise.name, set.weight, set.arm)
                          "
                          class="text-yellow-500 text-sm ml-1 m-icon"
                          :title="t('exercise.newWeightRecord')"
                        >
                          star
                        </span>
                      </div>
                    </div>
                    <div class="flex flex-col items-center text-center">
                      <div class="relative flex items-center">
                        <input
                          v-if="exerciseDisplay === 'reps'"
                          v-model.number="set.reps"
                          type="number"
                          class="text-base font-bold text-nb-text bg-transparent border-none text-center w-16 focus:outline-none"
                          placeholder="0"
                        />
                        <input
                          v-else
                          v-model="set.time"
                          type="text"
                          class="text-base font-bold text-nb-text bg-transparent border-none text-center w-16 focus:outline-none"
                          placeholder="0:00"
                        />
                      </div>
                      <div class="text-xs font-medium">
                        <template v-if="exerciseDisplay === 'reps'">
                          <template
                            v-if="
                              set.weight &&
                              getPreviousReps(exercise.name, set.weight, set.arm)
                            "
                          >
                            {{ t("exercise.reps") }} ({{ t("exercise.previousBest") }}:
                            {{
                              getPreviousReps(exercise.name, set.weight, set.arm)
                            }})
                          </template>
                          <template v-else> {{ t("exercise.reps") }} </template>
                          <span
                            v-if="
                              set.reps &&
                              set.weight &&
                              isRepRecord(
                                exercise.name,
                                set.weight,
                                set.reps,
                                set.arm
                              )
                            "
                            class="text-yellow-500 text-sm ml-1 m-icon"
                            :title="t('exercise.newRepRecord')"
                          >
                            star
                          </span>
                        </template>
                        <template v-else>
                          {{ t("exercise.time") }}
                        </template>
                      </div>
                    </div>
                    <div class="flex flex-col items-center text-center">
                      <select
                        v-model="set.rpe"
                        class="text-base font-bold text-nb-text bg-transparent border-none text-align-last-center w-16 focus:outline-none appearance-none"
                      >
                        <option value="">-</option>
                        <option
                          v-for="rpe in rpeOptions"
                          :key="rpe"
                          :value="rpe"
                        >
                          {{ rpe.replace("RPE ", "") }}
                        </option>
                      </select>
                      <div class="text-xs font-medium text-nb-text opacity-70">
                        RPE
                      </div>
                    </div>
                    <div
                      v-if="exercise.singleArm"
                      class="flex flex-col items-center text-center"
                    >
                      <select
                        v-model="set.arm"
                        class="text-base font-bold text-nb-text bg-transparent text-align-last-center border-none text-center w-16 focus:outline-none appearance-none"
                      >
                        <option value="">-</option>
                        <option value="left">
                          {{ t("exercise.arms.left") }}
                        </option>
                        <option value="right">
                          {{ t("exercise.arms.right") }}
                        </option>
                        <option value="both">
                          {{ t("exercise.arms.both") }}
                        </option>
                      </select>
                      <div
                        class="text-xs font-medium text-nb-text opacity-70 lowercase"
                      >
                        {{ t("exercise.arm") }}
                      </div>
                    </div>
                  </div>
                </div>

                <!-- Notes and % of max Row -->
                <div class="flex gap-3 items-center">
                  <!-- % of max -->
                  <div class="flex flex-col items-center text-center">
                    <div class="text-base font-bold text-nb-text">
                      {{ getMaxPercentage(exercise.name, set.weight, set.arm) }}
                    </div>
                    <div class="text-xs font-medium text-nb-text opacity-70">
                      {{ t("exercise.percentOfMax") }}
                    </div>
                  </div>

                  <!-- Notes -->
                  <div class="flex flex-col flex-1">
                    <input
                      v-model="set.notes"
                      type="text"
                      class="bg-white border-2 border-nb-border rounded-md px-2 py-1 text-sm font-medium text-nb-text focus:outline-none"
                      :placeholder="t('exercise.notes')"
                    />
                  </div>
                </div>
              </div>
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
import NeoButton from "@/components/NeoButton.vue";
import NeoPanel from "@/components/NeoPanel.vue";
import NeoHeader from "@/components/NeoHeader.vue";
import DestructiveButton from "@/components/DestructiveButton.vue";
import { useToast } from "@/composables/useToast.js";

const router = useRouter();
const { t } = useI18n();
const { showSuccess, showError, showInfo } = useToast();

// Set dynamic page title
useHead({
  title: () => {
    const workoutName = workout.value.name || t('workout.unnamed');
    const prefix = isNew.value ? t('workout.new') : t('workout.edit');
    return `${prefix}${isNew.value ? '' : `: ${workoutName}`} - Plonkout`;
  }
});

const props = defineProps({
  id: String,
});

const workout = ref({
  name: "",
  started: new Date(),
  ended: null,
  notes: "",
  exercises: [],
});

const saving = ref(false);
const showContextMenu = ref(false);
const showExerciseSelector = ref(false);
const allWorkouts = ref([]);
const weightUnit = ref("kg");
const exerciseDisplay = ref("reps");
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
 * Load exercise display setting
 */
async function loadExerciseDisplay() {
  try {
    const display = await getSetting("exerciseDisplay", "reps");
    exerciseDisplay.value = display;
  } catch (error) {
    console.error("Error loading exercise display:", error);
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

    if (isNew.value) {
      router.replace(`/workout/${id}`);
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
    // For warmup sets, count all warmup sets regardless of arm
    const warmupSets = sets
      .slice(0, setIndex + 1)
      .filter((s) => s.type === "warmup");
    let setNumber = `W${warmupSets.length}`;

    // Add arm designation for single-arm exercises
    if (exercise.singleArm && currentSet.arm) {
      const armSuffix =
        currentSet.arm === "left" ? "L" : currentSet.arm === "right" ? "R" : "";
      if (armSuffix) {
        setNumber += armSuffix;
      }
    }

    return setNumber;
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
 * Show exercise statistics (placeholder)
 * @param {number} _exerciseIndex - Index of the exercise
 */
function showExerciseStats(_exerciseIndex) {
  // TODO: Implement exercise statistics
  showInfo(t("exercise.statsComingSoon"));
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

  // Check all historical workouts
  allWorkouts.value.forEach((historicalWorkout) => {
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
  return previousBest === null || reps > previousBest;
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

  // Check all historical workouts for max weight
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
    router.push(`/workout/${id}`);
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
 * Confirm and delete workout
 */
async function confirmDelete() {
  showContextMenu.value = false;

  if (confirm(t("workout.deleteConfirm"))) {
    try {
      await deleteWorkout(parseInt(props.id));
      router.push("/log");
    } catch (error) {
      console.error("Error deleting workout:", error);
      showError(t("workout.deleteError"));
    }
  }
}

/**
 * Go back to workout list
 */
function goBack() {
  router.push("/log");
}

onMounted(async () => {
  await loadAllWorkouts();
  await loadWorkout();
  await loadWeightUnit();
  await loadExerciseDisplay();

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
