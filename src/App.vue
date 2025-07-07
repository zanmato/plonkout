<template>
  <div class="app-container">
    <!-- Main content area -->
    <main class="flex-1 overflow-hidden bg-gray-100 dark:bg-zinc-800">
      <router-view />
    </main>

    <!-- Toast notifications -->
    <VoltToast />

    <!-- Bottom tab navigation -->
    <nav class="neo-tab-bar dark:bg-zinc-800 dark:text-white safe-area-bottom">
      <div class="flex justify-around gap-3 items-center p-3">
        <router-link
          v-for="tab in tabs"
          :key="tab.name"
          :to="tab.path"
          class="flex flex-col flex-1 items-center px-4 py-2 text-xs font-semibold rounded-md border-3 border-black"
          :class="
            isActiveTab(tab.name)
              ? 'bg-purple-500 text-white'
              : 'text-gray-700 hover:bg-gray-100 dark:bg-zinc-700 dark:text-white dark:hover:bg-gray-400'
          "
        >
          <span class="material-icons text-xl mb-1">{{ tab.icon }}</span>
          <span>{{ t(`tabs.${tab.name}`) }}</span>
        </router-link>
      </div>
    </nav>
  </div>
</template>

<script setup>
import { onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import { useHead } from "@unhead/vue";
import { getSetting } from "@/utils/database.js";
import VoltToast from "@/volt/Toast.vue";

const { t, locale } = useI18n();
const currentTheme = ref("system");

// Set up base head configuration
useHead({
  title: "Plonkout",
  meta: [
    {
      name: "description",
      content: "Workout logger",
    },
    {
      name: "theme-color",
      content: () => getThemeColor(),
    },
    {
      name: "viewport",
      content: "width=device-width, initial-scale=1.0, viewport-fit=cover",
    },
  ],
});

const tabs = [
  {
    name: "log",
    path: "/log",
    icon: "list_alt",
  },
  {
    name: "templates",
    path: "/templates",
    icon: "fitness_center",
  },
  {
    name: "statistics",
    path: "/statistics",
    icon: "bar_chart",
  },
  {
    name: "settings",
    path: "/settings",
    icon: "settings",
  },
];

const route = useRoute();

/**
 * Check if the current tab is active based on the current route
 * @param {string} tabName - The name of the tab to check
 * @returns {boolean} - Whether the tab is active
 */
const isActiveTab = (tabName) => {
  const routeName = route.name;
  if (tabName === "templates") {
    return routeName === "templates";
  }
  if (tabName === "log") {
    return routeName === "log" || routeName === "workout-edit";
  }
  return routeName === tabName;
};

/**
 * Get theme color for meta tag based on current theme
 * @returns {string} The theme color hex value
 */
function getThemeColor() {
  const isDark =
    currentTheme.value === "dark" ||
    (currentTheme.value === "system" &&
      window.matchMedia("(prefers-color-scheme: dark)").matches);

  // Return dark theme color or light theme color
  return isDark ? "#18181b" : "#88AAEE";
}

/**
 * Apply theme to document
 * @param {string} theme - The theme setting ('light', 'dark', 'system')
 */
function applyTheme(theme) {
  currentTheme.value = theme;

  const isDark =
    theme === "dark" ||
    (theme === "system" &&
      window.matchMedia("(prefers-color-scheme: dark)").matches);

  if (isDark) {
    document.documentElement.classList.add("dark");
  } else {
    document.documentElement.classList.remove("dark");
  }
}

/**
 * Load and apply app settings on startup
 */
async function loadAppSettings() {
  try {
    // Load locale setting
    const savedLocale = await getSetting("locale", "en");
    locale.value = savedLocale;

    // Load and apply theme setting
    const savedTheme = await getSetting("theme", "system");
    applyTheme(savedTheme);

    // Listen for system theme changes when using 'system' theme
    if (savedTheme === "system") {
      const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
      mediaQuery.addEventListener("change", () => {
        applyTheme("system");
      });
    }

    currentTheme.value = savedTheme;
  } catch (error) {
    console.error("Error loading app settings:", error);
  }
}

onMounted(() => {
  loadAppSettings();
});
</script>

<style scoped>
.app-container {
  height: 100vh;
  display: flex;
  flex-direction: column;
}
</style>
