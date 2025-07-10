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

// Import views
import Workouts from "@/views/Workouts.vue";
import WorkoutLog from "@/views/WorkoutLog.vue";
import Statistics from "@/views/Statistics.vue";
import Settings from "@/views/Settings.vue";
import WorkoutEdit from "@/views/WorkoutEdit.vue";

// Router configuration
const routes = [
  { path: "/", redirect: "/log" },
  { path: "/templates", component: Workouts, name: "templates" },
  { path: "/log", component: WorkoutLog, name: "log" },
  {
    path: "/workout/:id?",
    component: WorkoutEdit,
    name: "workout-edit",
    props: true,
  },
  {
    path: "/template/:id?",
    component: WorkoutEdit,
    name: "template-edit",
    props: true,
  },
  { path: "/statistics", component: Statistics, name: "statistics" },
  { path: "/settings", component: Settings, name: "settings" },
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
