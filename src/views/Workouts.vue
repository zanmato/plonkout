<template>
  <div class="workouts h-full flex flex-col">
    <!-- Header -->
    <NeoHeader :title="t('tabs.templates')">
      <template #right>
        <NeoButton
          @click="addTemplate"
          variant="primary"
          size="sm"
          class="rounded-full w-10 h-10 !px-0 !py-0"
          icon-only
        >
          <template #icon>
            <span class="material-icons">add</span>
          </template>
          {{ t("workout.new") }}
        </NeoButton>
      </template>
    </NeoHeader>

    <!-- Content -->
    <div class="flex-1 overflow-auto hide-scrollbar">
      <div v-if="loading" class="flex items-center justify-center h-32">
        <div class="text-gray-500">{{ t("workout.loading") }}</div>
      </div>

      <div v-else-if="templates.length === 0" class="p-4">
        <NeoPanel class="text-center py-8">
          <div class="text-base font-bold text-black dark:text-white mb-2">
            {{ t("templates.noTemplates") }}
          </div>
          <div class="text-sm text-black dark:text-white opacity-70">
            {{ t("templates.noTemplatesDescription") }}
          </div>
        </NeoPanel>
      </div>

      <div v-else class="p-4 space-y-4">
        <NeoPanel
          v-for="template in templates"
          :key="template.id"
          class="cursor-pointer transition-all duration-200 hover:shadow-none hover:translate-x-1 hover:translate-y-1 active:shadow-none active:translate-x-1 active:translate-y-1"
          @click="startWorkout(template)"
        >
          <div class="flex justify-between items-start mb-4">
            <div class="flex-1">
              <div class="text-lg font-bold text-black dark:text-white mb-2">
                {{ template.name || t("workout.unnamed") }}
              </div>
              <div
                class="text-sm text-black dark:text-white opacity-80 leading-relaxed"
              >
                {{ getTemplateSummary(template) }}
              </div>
            </div>
            <div class="flex items-center space-x-2 ml-4">
              <NeoButton
                @click.stop="editTemplate(template.id)"
                variant="overlay"
                size="sm"
              >
                <template #icon>
                  <span class="material-icons">edit</span>
                </template>
              </NeoButton>
              <DestructiveButton
                @confirm="deleteTemplate(template.id)"
                :confirm-text="t('workout.delete')"
                size="sm"
                icon-only
              >
                <template #icon>
                  <span class="material-icons">delete</span>
                </template>
              </DestructiveButton>
            </div>
          </div>
        </NeoPanel>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useHead } from "@unhead/vue";
import {
  getWorkoutTemplates,
  deleteWorkoutTemplate,
  saveWorkout as saveWorkoutToDB,
} from "@/utils/database.js";
import NeoButton from "@/components/NeoButton.vue";
import NeoPanel from "@/components/NeoPanel.vue";
import NeoHeader from "@/components/NeoHeader.vue";
import DestructiveButton from "@/components/DestructiveButton.vue";
import { useToast } from "@/composables/useToast.js";

const router = useRouter();
const { t } = useI18n();
const { showSuccess, showError } = useToast();

// Set page title
useHead({
  title: () => t("tabs.templates"),
});

const templates = ref([]);
const loading = ref(true);

/**
 * Load workout templates from database
 */
async function loadTemplates() {
  try {
    loading.value = true;
    const data = await getWorkoutTemplates();
    templates.value = data.sort(
      (a, b) => new Date(b.created) - new Date(a.created)
    );
  } catch (error) {
    console.error("Error loading templates:", error);
    showError(t("templates.loadError"));
  } finally {
    loading.value = false;
  }
}

/**
 * Navigate to create new template
 */
function addTemplate() {
  router.push("/workout");
}

/**
 * Start a workout from a template
 * @param {Object} template - The template to start from
 */
async function startWorkout(template) {
  try {
    const newWorkout = {
      name: template.name,
      started: new Date(),
      ended: null,
      notes: template.notes || "",
      exercises: template.exercises.map((exercise) => ({
        ...exercise,
        sets: exercise.sets.map((set) => ({
          ...set,
          weight: null,
          reps: null,
          notes: "",
        })),
      })),
    };

    const id = await saveWorkoutToDB(newWorkout);
    router.push(`/workout/${id}`);
  } catch (error) {
    console.error("Error starting workout:", error);
    showError(t("templates.startError"));
  }
}

/**
 * Edit a template
 * @param {number} templateId - ID of template to edit
 */
function editTemplate(templateId) {
  router.push(`/template/${templateId}`);
}

/**
 * Delete a template
 * @param {number} templateId - ID of template to delete
 */
async function deleteTemplate(templateId) {
  try {
    await deleteWorkoutTemplate(templateId);
    await loadTemplates();
    showSuccess(t("templates.deleteSuccess"));
  } catch (error) {
    console.error("Error deleting template:", error);
    showError(t("templates.deleteError"));
  }
}

/**
 * Get template summary (exercises and sets)
 * @param {Object} template - The template object
 * @returns {string} Summary string
 */
function getTemplateSummary(template) {
  if (!template.exercises || template.exercises.length === 0) {
    return t("workout.summary.noExercises");
  }

  const totalSets = template.exercises.reduce((total, exercise) => {
    return (
      total +
      (exercise.sets
        ? exercise.sets.filter((set) => set.type !== "warmup").length
        : 0)
    );
  }, 0);

  if (template.exercises.length === 1) {
    return t("templates.summary.singleExercise", {
      totalSets,
      exerciseName: template.exercises[0].name,
    });
  } else {
    return t("templates.summary.multipleExercises", {
      totalSets,
      exerciseCount: template.exercises.length,
    });
  }
}

onMounted(() => {
  loadTemplates();
});
</script>

<style scoped>
/* Additional styles if needed */
</style>
