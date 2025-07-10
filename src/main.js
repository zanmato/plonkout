import { createApp } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import { createI18n } from "vue-i18n";
import { createHead } from "@unhead/vue/client";
import PrimeVue from "primevue/config";
import ToastService from "primevue/toastservice";
import App from "@/App.vue";
import "@/assets/css/app.css";
import "vue3-virtual-scroller/dist/vue3-virtual-scroller.css";

// Import i18n translations
import en from "@/locales/en.json";
import sv from "@/locales/sv.json";

// Router configuration with dynamic imports for code splitting
const routes = [
  { path: "/", redirect: "/log" },
  {
    path: "/templates",
    component: () => import("@/views/Workouts.vue"),
    name: "templates",
  },
  {
    path: "/log",
    component: () => import("@/views/WorkoutLog.vue"),
    name: "log",
  },
  {
    path: "/workout/:id?",
    component: () => import("@/views/WorkoutEdit.vue"),
    name: "workout-edit",
    props: true,
  },
  {
    path: "/template/:id?",
    component: () => import("@/views/WorkoutEdit.vue"),
    name: "template-edit",
    props: true,
  },
  {
    path: "/statistics",
    component: () => import("@/views/Statistics.vue"),
    name: "statistics",
  },
  {
    path: "/settings",
    component: () => import("@/views/Settings.vue"),
    name: "settings",
  },
];

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
});

// i18n configuration
const messages = {
  en,
  sv,
};

const i18n = createI18n({
  locale: "en",
  fallbackLocale: "en",
  legacy: false,
  datetimeFormats: {
    sv: {
      long: {
        year: "numeric",
        month: "short",
        day: "numeric",
        hour: "numeric",
        minute: "numeric",
      },
      short: {
        year: "numeric",
        month: "short",
        day: "numeric",
      },
      month: {
        month: "long",
        year: "numeric",
      },
    },
    en: {
      long: {
        year: "numeric",
        month: "short",
        day: "numeric",
        hour: "numeric",
        minute: "numeric",
      },
      short: {
        year: "numeric",
        month: "short",
        day: "numeric",
      },
      month: {
        month: "long",
        year: "numeric",
      },
    },
  },
  messages,
});

const head = createHead();

const app = createApp(App);
app.use(router);
app.use(i18n);
app.use(head);
app.use(PrimeVue);
app.use(ToastService);
app.mount("#app");
