<template>
  <div class="settings h-full flex flex-col">
    <!-- Header -->
    <NeoHeader :title="t('settings.title')" />

    <!-- Content -->
    <div class="flex-1 p-4">
      <NeoPanel>
        <!-- Language Setting -->
        <div class="mb-6">
          <h3 class="text-xl font-semibold text-black dark:text-white mb-2">
            {{ t("settings.language") }}
          </h3>
          <VoltSelect
            v-model="currentLocale"
            @change="changeLanguage"
            :options="languageOptions"
            optionLabel="label"
            optionValue="value"
            class="w-full"
          />
        </div>

        <!-- Theme Setting -->
        <div class="mb-6">
          <h3 class="text-xl font-semibold text-black dark:text-white mb-2">
            {{ t("settings.theme") }}
          </h3>
          <SelectButton
            v-model="theme"
            @change="saveTheme"
            :options="themeOptions"
            optionLabel="label"
            optionValue="value"
            class="w-full"
          />
        </div>

        <!-- Weight Unit Setting -->
        <div class="mb-6">
          <h3 class="text-xl font-semibold text-black dark:text-white mb-2">
            {{ t("settings.weightUnit") }}
          </h3>
          <SelectButton
            v-model="weightUnit"
            @change="saveWeightUnit"
            :options="weightUnitOptions"
            optionLabel="label"
            optionValue="value"
            class="w-full"
          />
        </div>

        <!-- Distance Unit Setting -->
        <div class="mb-6">
          <h3 class="text-xl font-semibold text-black dark:text-white mb-2">
            {{ t("settings.distanceUnit") }}
          </h3>
          <SelectButton
            v-model="distanceUnit"
            @change="saveDistanceUnit"
            :options="distanceUnitOptions"
            optionLabel="label"
            optionValue="value"
            class="w-full"
          />
        </div>

        <!-- Block Periodization Settings -->
        <div class="mb-6">
          <h3 class="text-xl font-semibold text-black dark:text-white mb-2">
            {{ t("settings.blockPeriodization.title") }}
          </h3>
          <p class="text-sm text-black dark:text-white opacity-70 mb-4">
            {{ t("settings.blockPeriodization.description") }}
          </p>
          <div class="space-y-4">
            <!-- Weeks per Block -->
            <div>
              <label
                class="block text-sm font-medium text-black dark:text-white mb-1"
              >
                {{ t("settings.blockPeriodization.weeksPerBlock") }}
              </label>
              <input
                v-model.number="blockWeeksPerBlock"
                type="number"
                min="1"
                max="12"
                @change="saveBlockSettings"
                class="w-full px-3 py-2 bg-white dark:bg-zinc-800 border-2 border-nb-border rounded-md text-black dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
              />
            </div>

            <!-- Workouts per Week -->
            <div>
              <label
                class="block text-sm font-medium text-black dark:text-white mb-1"
              >
                {{ t("settings.blockPeriodization.workoutsPerWeek") }}
              </label>
              <input
                v-model.number="blockWorkoutsPerWeek"
                type="number"
                min="1"
                max="7"
                @change="saveBlockSettings"
                class="w-full px-3 py-2 bg-white dark:bg-zinc-800 border-2 border-nb-border rounded-md text-black dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
              />
            </div>

            <!-- Start Percentage -->
            <div>
              <label
                class="block text-sm font-medium text-black dark:text-white mb-1"
              >
                {{ t("settings.blockPeriodization.startPercentage") }}
              </label>
              <div class="flex items-center">
                <input
                  v-model.number="blockStartPercentage"
                  type="number"
                  min="50"
                  max="90"
                  @change="saveBlockSettings"
                  class="flex-1 px-3 py-2 bg-white dark:bg-zinc-800 border-2 border-nb-border rounded-md text-black dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                />
                <span class="ml-2 text-black dark:text-white">%</span>
              </div>
            </div>

            <!-- Weekly Progression -->
            <div>
              <label
                class="block text-sm font-medium text-black dark:text-white mb-1"
              >
                {{ t("settings.blockPeriodization.progressionPerWeek") }}
              </label>
              <div class="flex items-center">
                <input
                  v-model.number="blockProgressionPerWeek"
                  type="number"
                  min="1"
                  max="10"
                  @change="saveBlockSettings"
                  class="flex-1 px-3 py-2 bg-white dark:bg-zinc-800 border-2 border-nb-border rounded-md text-black dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                />
                <span class="ml-2 text-black dark:text-white">%</span>
              </div>
            </div>

            <!-- Reset All Blocks -->
            <DestructiveButton
              @confirm="handleResetAllBlocks"
              :confirm-text="t('settings.blockPeriodization.resetAll')"
              full-width
            >
              <template #icon>
                <span class="material-icons">restart_alt</span>
              </template>
              {{ t("settings.blockPeriodization.resetAll") }}
            </DestructiveButton>
          </div>
        </div>

        <!-- Data Management -->
        <div class="mb-6">
          <h3 class="text-xl font-semibold text-black dark:text-white mb-2">
            {{ t("settings.dataManagement.title") }}
          </h3>
          <div class="space-y-3">
            <NeoButton
              @click="exportData"
              variant="secondary"
              full-width
              class="text-left"
            >
              <template #icon>
                <span class="material-icons">download</span>
              </template>
              <div class="flex flex-col">
                <div class="font-medium">
                  {{ t("settings.dataManagement.export") }}
                </div>
                <div class="text-sm text-gray-600 dark:text-gray-400">
                  {{ t("settings.dataManagement.exportDescription") }}
                </div>
              </div>
            </NeoButton>

            <DestructiveButton
              @confirm="clearAllData"
              :confirm-text="t('settings.dataManagement.clear')"
              full-width
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
</template>

<script setup>
import { ref, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useHead } from "@unhead/vue";
import { getSetting, saveSetting, getWorkouts } from "@/utils/database.js";
import {
  getGlobalSettings,
  saveGlobalSettings,
  resetAllBlocks,
} from "@/utils/blockPeriodization.js";
import { useToast } from "@/composables/useToast.js";
import NeoHeader from "@/components/NeoHeader.vue";
import NeoPanel from "@/components/NeoPanel.vue";
import NeoButton from "@/components/NeoButton.vue";
import DestructiveButton from "@/components/DestructiveButton.vue";
import VoltSelect from "@/volt/Select.vue";
import SelectButton from "@/volt/SelectButton.vue";

const { locale, t } = useI18n();
const { showSuccess, showError } = useToast();

// Set page title
useHead({
  title: () => t("settings.title"),
});

const currentLocale = ref("en");
const weightUnit = ref("kg");
const distanceUnit = ref("km");
const theme = ref("system");
const storageUsed = ref("");

// Block periodization settings
const blockWeeksPerBlock = ref(5);
const blockWorkoutsPerWeek = ref(1);
const blockStartPercentage = ref(75);
const blockProgressionPerWeek = ref(5);

// Options for Select and SelectButton components
const languageOptions = ref([
  { value: "en", label: t("settings.languages.en") },
  { value: "sv", label: t("settings.languages.sv") },
  { value: "de", label: t("settings.languages.de") },
  { value: "es", label: t("settings.languages.es") },
]);

const themeOptions = ref([
  { value: "light", label: t("settings.themes.light") },
  { value: "dark", label: t("settings.themes.dark") },
  { value: "system", label: t("settings.themes.system") },
]);

const weightUnitOptions = ref([
  { value: "kg", label: t("settings.units.kg") },
  { value: "lbs", label: t("settings.units.lbs") },
]);

const distanceUnitOptions = ref([
  { value: "km", label: t("settings.units.km") },
  { value: "miles", label: t("settings.units.miles") },
]);

onMounted(() => {
  storageUsed.value = t("settings.appInfo.calculating");
  loadSettings();
});

/**
 * Change application language
 */
async function changeLanguage() {
  locale.value = currentLocale.value;
  await saveSetting("locale", currentLocale.value);
}

/**
 * Save weight unit preference
 */
async function saveWeightUnit() {
  await saveSetting("weightUnit", weightUnit.value);
}

/**
 * Save distance unit preference
 */
async function saveDistanceUnit() {
  await saveSetting("distanceUnit", distanceUnit.value);
}

/**
 * Save block periodization settings
 */
async function saveBlockSettings() {
  await saveGlobalSettings({
    weeksPerBlock: blockWeeksPerBlock.value,
    workoutsPerWeek: blockWorkoutsPerWeek.value,
    startPercentage: blockStartPercentage.value,
    progressionPerWeek: blockProgressionPerWeek.value,
  });
}

/**
 * Reset all block data
 */
async function handleResetAllBlocks() {
  await resetAllBlocks();
  showSuccess(t("settings.blockPeriodization.resetSuccess"));
}

/**
 * Save theme preference and apply it
 */
async function saveTheme() {
  await saveSetting("theme", theme.value);
  applyTheme();
}

/**
 * Apply theme to document
 */
function applyTheme() {
  const isDark =
    theme.value === "dark" ||
    (theme.value === "system" &&
      window.matchMedia("(prefers-color-scheme: dark)").matches);

  if (isDark) {
    document.documentElement.classList.add("dark");
  } else {
    document.documentElement.classList.remove("dark");
  }
}

/**
 * Export workout data as JSON file
 */
async function exportData() {
  try {
    const workouts = await getWorkouts();
    const data = {
      version: "1.0.0",
      exportDate: new Date().toISOString(),
      workouts: workouts,
    };

    const blob = new Blob([JSON.stringify(data, null, 2)], {
      type: "application/json",
    });

    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `plonkout-export-${
      new Date().toISOString().split("T")[0]
    }.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  } catch (error) {
    console.error(t("settings.dataManagement.exportError"), error);
    showError(t("settings.dataManagement.exportError"));
  }
}

/**
 * Clear all application data
 */
async function clearAllData() {
  try {
    // Clear IndexedDB
    if ("indexedDB" in window) {
      const databases = await indexedDB.databases();
      await Promise.all(
        databases.map((db) => {
          return new Promise((resolve, reject) => {
            const deleteReq = indexedDB.deleteDatabase(db.name);
            deleteReq.onsuccess = () => resolve();
            deleteReq.onerror = () => reject(deleteReq.error);
          });
        }),
      );
    }

    // Clear localStorage
    localStorage.clear();

    // Clear sessionStorage
    sessionStorage.clear();

    showSuccess(t("settings.dataManagement.clearSuccess"));
    window.location.reload();
  } catch (error) {
    console.error(t("settings.dataManagement.clearError"), error);
    showError(t("settings.dataManagement.clearError"));
  }
}

/**
 * Load settings from database
 */
async function loadSettings() {
  try {
    const savedLocale = await getSetting("locale", "en");
    const savedWeightUnit = await getSetting("weightUnit", "kg");
    const savedDistanceUnit = await getSetting("distanceUnit", "km");
    const savedTheme = await getSetting("theme", "system");
    currentLocale.value = savedLocale;
    locale.value = savedLocale;
    weightUnit.value = savedWeightUnit;
    distanceUnit.value = savedDistanceUnit;
    theme.value = savedTheme;

    // Load block periodization settings
    const blockSettings = await getGlobalSettings();
    blockWeeksPerBlock.value = blockSettings.weeksPerBlock;
    blockWorkoutsPerWeek.value = blockSettings.workoutsPerWeek;
    blockStartPercentage.value = blockSettings.startPercentage;
    blockProgressionPerWeek.value = blockSettings.progressionPerWeek;

    applyTheme();
  } catch (error) {
    console.error(t("settings.loadError"), error);
  }
}
</script>
