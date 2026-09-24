import { createApp } from "vue";
import {
  createRouter,
  createWebHashHistory,
  type RouteRecordRaw,
} from "vue-router";
import { createI18n } from "vue-i18n";
import { createHead } from "@unhead/vue/client";
import PrimeVue from "primevue/config";
import ToastService from "primevue/toastservice";
import App from "@/App.vue";
import { onUnauthorized } from "@/api";
import { useAuth } from "@/composables/useAuth";
import "@/assets/css/app.css";
import "vue3-virtual-scroller/dist/vue3-virtual-scroller.css";

// Import i18n translations
import en from "@/locales/en.json";
import sv from "@/locales/sv.json";

declare module "vue-router" {
  interface RouteMeta {
    /** Reachable without signing in. */
    public?: boolean;
  }
}

// Router configuration with dynamic imports for code splitting
const routes: RouteRecordRaw[] = [
  { path: "/", redirect: "/log" },
  {
    path: "/welcome",
    component: () => import("@/views/auth/Welcome.vue"),
    name: "welcome",
    meta: { public: true },
  },
  {
    path: "/signup",
    component: () => import("@/views/auth/Signup.vue"),
    name: "signup",
    meta: { public: true },
  },
  {
    path: "/recover",
    component: () => import("@/views/auth/Recover.vue"),
    name: "recover",
    meta: { public: true },
  },
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
    path: "/plan",
    component: () => import("@/views/Plan.vue"),
    name: "plan",
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
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes,
});

// Everything but the sign in pages needs a user. The user is loaded once and
// shared, so this is one request per app start, not per navigation.
router.beforeEach(async (to) => {
  const auth = useAuth();
  let signedIn = false;
  try {
    signedIn = (await auth.load()) !== null;
  } catch (error) {
    // The server could not be reached. The sign in page says so when used.
    console.error("Could not load the signed in user:", error);
  }
  if (to.meta.public) {
    return signedIn && to.name === "welcome" ? { name: "log" } : true;
  }
  return signedIn ? true : { name: "welcome", query: { returnTo: to.fullPath } };
});

// A session that expires mid use sends the person back to sign in, and back
// to where they were afterwards.
onUnauthorized(() => {
  useAuth().clear();
  const current = router.currentRoute.value;
  if (!current.meta.public) {
    router.replace({ name: "welcome", query: { returnTo: current.fullPath } });
  }
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
