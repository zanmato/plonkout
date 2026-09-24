<template>
  <div class="settings h-full flex flex-col">
    <!-- Header -->
    <NeoHeader :title="t('settings.title')" />

    <!-- Content -->
    <div class="flex-1 overflow-y-auto p-4 space-y-4">
      <AccountSettings />

      <NeoPanel>
        <!-- Language Setting -->
        <div class="mb-6">
          <h3 class="text-xl font-semibold text-black dark:text-white mb-2">
            {{ t("settings.language") }}
          </h3>
          <VoltSelect
            v-model="currentLocale"
            :options="languageOptions"
            optionLabel="label"
            optionValue="value"
            class="w-full"
            @change="changeLanguage"
          />
        </div>

        <!-- Theme Setting -->
        <div class="mb-6">
          <h3 class="text-xl font-semibold text-black dark:text-white mb-2">
            {{ t("settings.theme") }}
          </h3>
          <SelectButton
            v-model="theme"
            :options="themeOptions"
            optionLabel="label"
            optionValue="value"
            class="w-full"
            @change="saveTheme"
          />
        </div>

        <!-- Weight Unit Setting -->
        <div class="mb-6">
          <h3 class="text-xl font-semibold text-black dark:text-white mb-2">
            {{ t("settings.weightUnit") }}
          </h3>
          <SelectButton
            v-model="weightUnit"
            :options="weightUnitOptions"
            optionLabel="label"
            optionValue="value"
            class="w-full"
            @change="saveWeightUnit"
          />
        </div>

        <!-- Distance Unit Setting -->
        <div class="mb-6">
          <h3 class="text-xl font-semibold text-black dark:text-white mb-2">
            {{ t("settings.distanceUnit") }}
          </h3>
          <SelectButton
            v-model="distanceUnit"
            :options="distanceUnitOptions"
            optionLabel="label"
            optionValue="value"
            class="w-full"
            @change="saveDistanceUnit"
          />
        </div>

        <!-- Compare Scope Setting -->
        <div class="mb-6">
          <h3 class="text-xl font-semibold text-black dark:text-white mb-2">
            {{ t("settings.compareScope.title") }}
          </h3>
          <p class="text-sm text-black dark:text-white opacity-70 mb-4">
            {{ t("settings.compareScope.description") }}
          </p>
          <SelectButton
            v-model="compareScope"
            :options="compareScopeOptions"
            optionLabel="label"
            optionValue="value"
            class="w-full"
            @change="saveCompareScope"
          />
        </div>

        <!-- Dominant Arm Setting -->
        <div class="mb-6">
          <h3 class="text-xl font-semibold text-black dark:text-white mb-2">
            {{ t("settings.dominantArm.title") }}
          </h3>
          <p class="text-sm text-black dark:text-white opacity-70 mb-4">
            {{ t("settings.dominantArm.description") }}
          </p>
          <SelectButton
            v-model="dominantArm"
            :options="dominantArmOptions"
            optionLabel="label"
            optionValue="value"
            :allowEmpty="false"
            class="w-full"
            data-testid="dominant-arm"
            @change="saveDominantArm"
          />
        </div>

        <!-- Data Management -->
        <div class="mb-6">
          <h3 class="text-xl font-semibold text-black dark:text-white mb-2">
            {{ t("settings.dataManagement.title") }}
          </h3>
          <div class="space-y-3">
            <NeoButton
              variant="secondary"
              full-width
              class="text-left"
              @click="exportData"
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

            <NeoButton
              variant="secondary"
              full-width
              class="text-left"
              :disabled="importing"
              @click="importInput?.click()"
            >
              <template #icon>
                <span class="material-icons">upload</span>
              </template>
              <div class="flex flex-col">
                <div class="font-medium">
                  {{ t("settings.dataManagement.import") }}
                </div>
                <div class="text-sm text-gray-600 dark:text-gray-400">
                  {{ t("settings.dataManagement.importDescription") }}
                </div>
              </div>
            </NeoButton>
            <input
              ref="importInput"
              type="file"
              accept="application/json,.json"
              class="hidden"
              data-testid="import-file-input"
              @change="importData"
            />
          </div>
        </div>
      </NeoPanel>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, useTemplateRef } from "vue";
import { useI18n } from "vue-i18n";
import { useHead } from "@unhead/vue";
import {
  getSetting,
  saveSetting,
  exportAll,
  importLegacyExport,
} from "@/api/data";
import { getLocalPref, setLocalPref, type Theme } from "@/utils/localPrefs";
import { useToast } from "@/composables/useToast";
import type { DominantArm } from "@/utils/plan";
import NeoHeader from "@/components/NeoHeader.vue";
import NeoPanel from "@/components/NeoPanel.vue";
import AccountSettings from "@/components/AccountSettings.vue";
import NeoButton from "@/components/NeoButton.vue";
import VoltSelect from "@/volt/Select.vue";
import SelectButton from "@/volt/SelectButton.vue";
import {
  COMPARE_SCOPES,
  type CompareScope,
} from "@/composables/useExerciseHistory";

const { locale, t } = useI18n();
const { showSuccess, showError } = useToast();

// Set page title
useHead({
  title: () => t("settings.title"),
});

const currentLocale = ref("en");
const weightUnit = ref("kg");
const compareScope = ref<CompareScope>("intensity");
const distanceUnit = ref("km");
const theme = ref<Theme>("system");
const dominantArm = ref<DominantArm>("right");
const importing = ref(false);
const importInput = useTemplateRef<HTMLInputElement>("importInput");
const storageUsed = ref("");

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

const compareScopeOptions = computed(() =>
  COMPARE_SCOPES.map((value) => ({
    value,
    label: t(`settings.compareScope.${value}`),
  })),
);

const dominantArmOptions = computed(() => [
  { value: "right", label: t("settings.dominantArm.right") },
  { value: "left", label: t("settings.dominantArm.left") },
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
function changeLanguage() {
  locale.value = currentLocale.value;
  setLocalPref("locale", currentLocale.value);
}

/**
 * Save weight unit preference
 */
async function saveWeightUnit() {
  await saveSetting("weightUnit", weightUnit.value);
}

/**
 * Save the compare scope preference
 */
async function saveCompareScope() {
  await saveSetting("compareScope", compareScope.value);
}

/**
 * Save the dominant arm, which single arm plan weights are listed for
 */
async function saveDominantArm() {
  await saveSetting("dominantArm", dominantArm.value);
}

/**
 * Save distance unit preference
 */
async function saveDistanceUnit() {
  await saveSetting("distanceUnit", distanceUnit.value);
}

/**
 * Save theme preference and apply it
 */
function saveTheme() {
  setLocalPref("theme", theme.value);
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
 * Export everything in the account as a JSON backup file
 */
async function exportData() {
  try {
    const data = await exportAll();

    const blob = new Blob([JSON.stringify(data, null, 2)], {
      type: "application/json",
    });

    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `plonkout-backup-${
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
 * Import a JSON file exported by the old local only app
 */
async function importData(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;

  importing.value = true;
  try {
    const result = await importLegacyExport(JSON.parse(await file.text()));
    showSuccess(t("settings.dataManagement.importSuccess", { ...result }));
  } catch (error) {
    console.error(t("settings.dataManagement.importError"), error);
    showError(t("settings.dataManagement.importError"));
  } finally {
    importing.value = false;
    // Let the same file be picked again
    input.value = "";
  }
}

/**
 * Load settings, theme and language are stored on this device
 */
async function loadSettings() {
  currentLocale.value = getLocalPref("locale");
  theme.value = getLocalPref("theme");

  try {
    const savedWeightUnit = await getSetting("weightUnit", "kg");
    const savedDistanceUnit = await getSetting("distanceUnit", "km");
    compareScope.value = await getSetting<CompareScope>(
      "compareScope",
      "intensity",
    );
    weightUnit.value = savedWeightUnit;
    distanceUnit.value = savedDistanceUnit;
    dominantArm.value = await getSetting<DominantArm>("dominantArm", "right");
  } catch (error) {
    console.error(t("settings.loadError"), error);
  }
}
</script>
