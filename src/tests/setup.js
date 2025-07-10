import { vi } from "vitest";
import { config } from "@vue/test-utils";
import { createHead } from "@unhead/vue/client";
import { createI18n } from "vue-i18n";

// Import actual locale files
import en from "@/locales/en.json";
import sv from "@/locales/sv.json";

// Mock database module
vi.mock("@/utils/database.js", async () => {
  const mocks = await import("./mocks/database.js");
  return mocks;
});

const mockIntersectionObserver = vi.fn();
mockIntersectionObserver.mockReturnValue({
  observe: () => null,
  unobserve: () => null,
  disconnect: () => null,
});
window.IntersectionObserver = mockIntersectionObserver;

// Mock PrimeVue Toast
vi.mock("primevue/usetoast", () => ({
  useToast: () => ({
    add: vi.fn(),
    removeGroup: vi.fn(),
    removeAllGroups: vi.fn(),
  }),
}));

// Mock vue-router
const mockPush = vi.fn();
const mockReplace = vi.fn();

vi.mock("vue-router", () => ({
  useRouter: () => ({
    push: mockPush,
    replace: mockReplace,
    go: vi.fn(),
    back: vi.fn(),
    forward: vi.fn(),
  }),
  useRoute: () => ({
    params: {},
    query: {},
    path: "/",
    name: "test",
  }),
}));

// Mock global confirm and alert functions
global.confirm = vi.fn(() => true);
global.alert = vi.fn();

// Create head instance for tests
const head = createHead();

// Create actual i18n instance for tests
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
  messages: {
    en,
    sv,
  },
});

// Global test utilities
config.global.plugins = [head, i18n];
config.global.mocks = {};

// Export router mocks for test use
export { mockPush, mockReplace };
