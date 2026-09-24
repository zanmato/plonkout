<template>
  <div class="app-container">
    <!-- Main content area -->
    <main class="flex-1 overflow-hidden bg-gray-100 dark:bg-zinc-800">
      <router-view />
    </main>

    <!-- Toast notifications -->
    <VoltToast />

    <!-- Bottom tab navigation, hidden on the sign in pages -->
    <nav
      v-if="!route.meta.public"
      class="border-t-2 border-black bg-white dark:bg-zinc-800 dark:text-white safe-area-bottom"
      style="box-shadow: 0 -4px 0px black"
    >
      <div class="flex justify-around gap-2 items-center p-3">
        <router-link
          v-for="tab in tabs"
          :key="tab.name"
          :to="tab.path"
          class="flex flex-col flex-1 min-w-0 items-center px-1 py-2 text-xs font-semibold rounded-md border-3 border-black"
          :class="
            isActiveTab(tab.name)
              ? 'bg-purple-500 text-white'
              : 'text-gray-700 hover:bg-gray-100 dark:bg-zinc-700 dark:text-white dark:hover:bg-gray-400'
          "
        >
          <span class="material-icons text-xl mb-1">{{ tab.icon }}</span>
          <span class="max-w-full truncate">{{ t(`tabs.${tab.name}`) }}</span>
        </router-link>
      </div>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import { useHead } from "@unhead/vue";
import { useRegisterSW } from "virtual:pwa-register/vue";
import { getLocalPref, type Theme } from "@/utils/localPrefs";
import VoltToast from "@/volt/Toast.vue";

const { t, locale } = useI18n();
const currentTheme = ref<Theme>("system");
const baseUrl = import.meta.env.BASE_URL.replace(/\/$/, "");

// PWA auto-reload functionality
const { needRefresh, updateServiceWorker } = useRegisterSW({
  onRegistered(r) {
    console.log("SW Registered: " + r);
  },
  onRegisterError(error) {
    console.log("SW registration error", error);
  },
});

// Set up base head configuration with title template
useHead({
  title: "Plonkout",
  titleTemplate: (title?: string) => (title ? `${title} - Plonkout` : "Plonkout"),
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
      name: "apple-mobile-web-app-title",
      content: "Plonkout",
    },
  ],
  link: [
    {
      rel: "icon",
      type: "image/png",
      href: () => `${baseUrl}/favicon-96x96.png`,
      sizes: "96x96",
    },
    {
      rel: "icon",
      type: "image/svg+xml",
      href: () => `${baseUrl}/favicon.svg`,
    },
    {
      rel: "shortcut icon",
      href: () => `${baseUrl}/favicon.ico`,
    },
    {
      rel: "apple-touch-icon",
      sizes: "180x180",
      href: () => `${baseUrl}/apple-touch-icon.png`,
    },
    {
      rel: "manifest",
      href: () => `${baseUrl}/site.webmanifest`,
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
    name: "plan",
    path: "/plan",
    icon: "event_note",
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
 */
const isActiveTab = (tabName: string): boolean => {
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
 */
function getThemeColor(): string {
  const isDark =
    currentTheme.value === "dark" ||
    (currentTheme.value === "system" &&
      window.matchMedia("(prefers-color-scheme: dark)").matches);

  // Return dark theme color or light theme color
  return isDark ? "#0a0a0a" : "#ffffff";
}

/**
 * Apply theme to document
 */
function applyTheme(theme: Theme) {
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
function loadAppSettings() {
  // Theme and language are stored per device, so they are known before sign in
  locale.value = getLocalPref("locale");

  const savedTheme = getLocalPref("theme");
  applyTheme(savedTheme);

  // Listen for system theme changes when using 'system' theme
  if (savedTheme === "system") {
    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    mediaQuery.addEventListener("change", () => {
      applyTheme("system");
    });
  }
}

// Auto-reload when PWA update is available
watch(needRefresh, (needRefresh) => {
  if (needRefresh) {
    console.log("New app version available, reloading...");
    updateServiceWorker(true);
  }
});

function handleViewportChange() {
  // Update a CSS custom property that you can use alongside or instead of dvh
  document.documentElement.style.setProperty(
    "--actual-viewport-height",
    `${window.visualViewport!.height}px`
  );

  // Force dvh recalculation by triggering a reflow
  document.body.style.transform = "translateZ(0)";
  requestAnimationFrame(() => {
    document.body.style.transform = "";
  });
}

onMounted(() => {
  loadAppSettings();

  if ("visualViewport" in window) {
    window.visualViewport!.addEventListener("resize", handleViewportChange);
    handleViewportChange();
  }
});
</script>

<style scoped>
.app-container {
  height: 100dvh;
  min-height: var(--actual-viewport-height, 100dvh);
  display: flex;
  flex-direction: column;
}
</style>
