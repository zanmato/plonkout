<template>
  <div
    @click="$emit('close')"
    class="fixed inset-0 z-50 flex items-end justify-center"
    style="background-color: rgba(0, 0, 0, 0.4)"
  >
    <NeoPanel
      @click.stop
      class="w-full max-w-md max-h-96 flex flex-col safe-area-bottom"
      data-testid="exercise-selector-content"
    >
      <!-- Header -->
      <div class="p-4 border-b-2 border-nb-border">
        <div class="flex items-center justify-between">
          <h3 class="text-xl font-semibold text-black dark:text-white mb-2">
            {{ t("exercise.selector.title") }}
          </h3>
          <NeoButton
            @click="showNewExerciseForm = true"
            variant="primary"
            size="sm"
          >
            {{ t("exercise.selector.new") }}
          </NeoButton>
        </div>

        <!-- Search -->
        <div class="floating-label-container mt-4">
          <input
            v-model="searchQuery"
            type="text"
            id="exercise-search"
            class="floating-input"
            :placeholder="t('exercise.selector.search')"
          />
          <label for="exercise-search" class="floating-label">
            {{ t("exercise.selector.search") }}
          </label>
        </div>
      </div>

      <!-- Exercise List -->
      <div class="flex-1 overflow-auto p-4">
        <div
          v-if="loading"
          class="text-center py-8 text-black dark:text-white opacity-70"
        >
          {{ t("exercise.selector.loading") }}
        </div>

        <div
          v-else-if="filteredExercises.length === 0"
          class="text-center py-8 text-black dark:text-white opacity-70"
        >
          {{ t("exercise.selector.noResults") }}
        </div>

        <div v-else class="space-y-4">
          <div
            v-for="group in groupedExercises"
            :key="group.muscleGroup"
            class="space-y-2"
          >
            <h4
              class="text-sm font-bold text-black dark:text-white opacity-70 uppercase tracking-wider px-1"
            >
              {{
                t(`exercise.muscleGroups.${group.muscleGroup.toLowerCase()}`) ||
                group.muscleGroup
              }}
            </h4>
            <div class="space-y-2">
              <button
                v-for="exercise in group.exercises"
                :key="exercise.id"
                @click="selectExercise(exercise)"
                class="w-full text-left p-3 bg-nb-overlay border-2 border-nb-border rounded-lg shadow-brutal-sm hover:shadow-none hover:translate-x-1 hover:translate-y-1 transition-all duration-200"
              >
                <div class="font-bold text-white dark:text-black">
                  {{ exercise.name }}
                </div>
                <div
                  class="text-sm text-white dark:text-black opacity-70 flex items-center justify-between"
                >
                  <span class="text-xs">
                    {{
                      exercise.singleArm
                        ? t("exercise.selector.singleArm")
                        : t("exercise.selector.bothArms")
                    }}
                  </span>
                  <span
                    v-if="exercise.type"
                    class="text-xs px-2 py-1 bg-nb-border rounded text-black dark:text-white"
                  >
                    {{ t(`exercise.types.${exercise.type}`) }}
                  </span>
                </div>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Cancel Button -->
      <div class="p-4 border-t-2 border-nb-border">
        <NeoButton @click="$emit('close')" variant="overlay" full-width>
          {{ t("common.cancel") }}
        </NeoButton>
      </div>
    </NeoPanel>

    <!-- New Exercise Form -->
    <div
      v-if="showNewExerciseForm"
      @click="showNewExerciseForm = false"
      class="fixed inset-0 z-60 flex items-center justify-center p-4"
      style="background-color: rgba(0, 0, 0, 0.4)"
    >
      <NeoPanel @click.stop class="w-full max-w-sm" padding="lg">
        <h3 class="text-xl font-semibold text-black dark:text-white mb-2 mb-6">
          {{ t("exercise.selector.newExercise") }}
        </h3>

        <div class="space-y-6">
          <div class="floating-label-container">
            <input
              v-model="newExercise.name"
              type="text"
              id="exercise-name"
              class="floating-input"
              :placeholder="t('exercise.selector.namePlaceholder')"
            />
            <label for="exercise-name" class="floating-label">
              {{ t("exercise.selector.name") }}
            </label>
          </div>

          <div>
            <label
              class="block text-sm font-medium text-black dark:text-white mb-2"
            >
              {{ t("exercise.selector.muscleGroup") }}
            </label>
            <VoltSelect
              v-model="newExercise.muscleGroup"
              :options="muscleGroupOptions"
              optionLabel="label"
              optionValue="value"
              :placeholder="t('exercise.selector.muscleGroupPlaceholder')"
              class="w-full"
            />
          </div>

          <div>
            <label
              class="block text-sm font-medium text-black dark:text-white mb-2"
            >
              {{ t("exercise.selector.exerciseType") }}
            </label>
            <VoltSelect
              v-model="newExercise.type"
              :options="exerciseTypeOptions"
              optionLabel="label"
              optionValue="value"
              class="w-full"
            />
          </div>

          <div>
            <label
              class="block text-sm font-medium text-black dark:text-white mb-2"
            >
              {{ t("exercise.selector.displayType") }}
            </label>
            <VoltSelect
              v-model="newExercise.displayType"
              :options="displayTypeOptions"
              optionLabel="label"
              optionValue="value"
              class="w-full"
            />
          </div>

          <div
            v-if="newExercise.type !== 'cardio'"
            class="flex items-center space-x-3"
          >
            <input
              v-model="newExercise.singleArm"
              type="checkbox"
              id="exercise-single-arm"
              class="w-5 h-5 border-2 border-nb-border rounded bg-nb-bg checked:bg-nb-accent checked:border-nb-accent focus:outline-none focus:ring-2 focus:ring-nb-accent focus:ring-offset-2"
            />
            <label
              for="exercise-single-arm"
              class="text-sm font-medium text-black dark:text-white"
            >
              {{ t("exercise.selector.singleArmLabel") }}
            </label>
          </div>
        </div>

        <div class="flex space-x-3 mt-8">
          <NeoButton
            @click="showNewExerciseForm = false"
            variant="overlay"
            class="flex-1"
          >
            {{ t("common.cancel") }}
          </NeoButton>
          <NeoButton
            @click="createExercise"
            :disabled="!newExercise.name.trim()"
            variant="primary"
            class="flex-1"
          >
            {{ t("exercise.selector.create") }}
          </NeoButton>
        </div>
      </NeoPanel>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { getExercises, saveExercise } from "@/utils/database.js";
import { useToast } from "@/composables/useToast.js";
import NeoButton from "@/components/NeoButton.vue";
import NeoPanel from "@/components/NeoPanel.vue";
import VoltSelect from "@/volt/Select.vue";

const emit = defineEmits(["close", "select"]);
const { t } = useI18n();
const { showError } = useToast();

const exercises = ref([]);
const loading = ref(true);
const searchQuery = ref("");
const showNewExerciseForm = ref(false);

const newExercise = ref({
  name: "",
  muscleGroup: "",
  singleArm: true,
  type: "strength",
  displayType: "reps",
});

// Predefined muscle group options
const muscleGroupOptions = computed(() => [
  { value: "Shoulders", label: t("exercise.muscleGroups.shoulders") },
  { value: "Legs", label: t("exercise.muscleGroups.legs") },
  { value: "Biceps", label: t("exercise.muscleGroups.biceps") },
  { value: "Triceps", label: t("exercise.muscleGroups.triceps") },
  { value: "Chest", label: t("exercise.muscleGroups.chest") },
  { value: "Abs", label: t("exercise.muscleGroups.abs") },
  { value: "Back", label: t("exercise.muscleGroups.back") },
  { value: "Forearm", label: t("exercise.muscleGroups.forearm") },
]);

// Exercise type options
const exerciseTypeOptions = computed(() => [
  { value: "strength", label: t("exercise.types.strength") },
  { value: "cardio", label: t("exercise.types.cardio") },
]);

// Display type options
const displayTypeOptions = computed(() => [
  { value: "reps", label: t("exercise.displayTypes.reps") },
  { value: "time", label: t("exercise.displayTypes.time") },
]);

/**
 * Filter exercises based on search query
 */
const filteredExercises = computed(() => {
  if (!searchQuery.value.trim()) {
    return exercises.value;
  }

  const query = searchQuery.value.toLowerCase();
  return exercises.value.filter(
    (exercise) =>
      exercise.name.toLowerCase().includes(query) ||
      exercise.muscleGroup.toLowerCase().includes(query)
  );
});

/**
 * Group filtered exercises by muscle group
 */
const groupedExercises = computed(() => {
  const groups = {};

  filteredExercises.value.forEach((exercise) => {
    const muscleGroup = exercise.muscleGroup;
    if (!groups[muscleGroup]) {
      groups[muscleGroup] = {
        muscleGroup,
        exercises: [],
      };
    }
    groups[muscleGroup].exercises.push(exercise);
  });

  // Sort groups by muscle group name and exercises within each group by name
  return Object.values(groups)
    .sort((a, b) => a.muscleGroup.localeCompare(b.muscleGroup))
    .map((group) => ({
      ...group,
      exercises: group.exercises.sort((a, b) => a.name.localeCompare(b.name)),
    }));
});

/**
 * Load exercises from database
 */
async function loadExercises() {
  try {
    loading.value = true;
    exercises.value = await getExercises();
  } catch (error) {
    console.error(t("exercise.selector.loadError"), error);
  } finally {
    loading.value = false;
  }
}

/**
 * Select an exercise and emit to parent
 * @param {Object} exercise - The selected exercise
 */
function selectExercise(exercise) {
  emit("select", exercise);
}

/**
 * Create a new exercise
 */
async function createExercise() {
  if (!newExercise.value.name.trim()) return;

  try {
    const exerciseData = {
      name: newExercise.value.name.trim(),
      muscleGroup:
        newExercise.value.muscleGroup || muscleGroupOptions.value[0].value,
      singleArm:
        newExercise.value.type === "cardio"
          ? false
          : newExercise.value.singleArm,
      type: newExercise.value.type,
      displayType: newExercise.value.displayType,
    };

    const id = await saveExercise(exerciseData);
    const savedExercise = { ...exerciseData, id };

    exercises.value.push(savedExercise);
    selectExercise(savedExercise);

    // Reset form
    newExercise.value = {
      name: "",
      muscleGroup: "",
      singleArm: true,
      type: "strength",
      displayType: "reps",
    };
    showNewExerciseForm.value = false;
  } catch (error) {
    console.error(t("exercise.selector.createError"), error);
    showError(t("exercise.selector.createError"));
  }
}

onMounted(() => {
  loadExercises();
});
</script>
