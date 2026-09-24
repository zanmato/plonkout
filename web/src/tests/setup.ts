import { afterAll, afterEach, beforeAll, vi, type Mock } from "vitest";
import { config } from "@vue/test-utils";
import { createHead } from "@unhead/vue/client";
import { createI18n } from "vue-i18n";
import { client } from "@/api/gen/client.gen";
import { clearSettingsCache } from "@/api/data";
import { API, server } from "./helpers/msw";
import { backend } from "./mocks/backend";

// Import actual locale files
import en from "@/locales/en.json";
import sv from "@/locales/sv.json";

// Every test talks to the in-memory fake backend over MSW. Tests seed it with
// backend.seed and add handlers of their own with server.use.
beforeAll(() => {
  client.setConfig({ baseUrl: API });
  server.listen({ onUnhandledRequest: "error" });
});
afterEach(() => {
  server.resetHandlers();
  backend.reset();
  clearSettingsCache();
});
afterAll(() => server.close());

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
const mockPush: Mock = vi.fn();
const mockReplace: Mock = vi.fn();

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
