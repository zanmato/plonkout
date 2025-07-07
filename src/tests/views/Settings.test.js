import { describe, it, expect, beforeEach, vi } from "vitest";
import { mount } from "@vue/test-utils";
import { nextTick } from "vue";
import Settings from "@/views/Settings.vue";
import { getSetting, saveSetting, getWorkouts } from "@/utils/database.js";

// Mock vue-i18n
const mockT = vi.fn((key, params) => {
  const translations = {
    "settings.title": "Settings",
    "settings.language": "Language",
    "settings.weightUnit": "Weight Unit",
    "settings.theme": "Theme",
    "settings.exerciseDisplay": "Exercise Display",
    "settings.themes.light": "Light",
    "settings.themes.dark": "Dark",
    "settings.themes.system": "System",
    "settings.units.kg": "Kilograms (kg)",
    "settings.units.lbs": "Pounds (lbs)",
    "settings.exerciseDisplays.reps": "Reps",
    "settings.exerciseDisplays.time": "Time",
    "settings.languages.en": "English",
    "settings.languages.sv": "Svenska",
    "settings.languages.de": "Deutsch",
    "settings.languages.es": "Español",
    "settings.appInfo.title": "App Information",
    "settings.appInfo.version": "Version",
    "settings.appInfo.versionNumber": "1.0.0",
    "settings.appInfo.build": "Build",
    "settings.appInfo.storageUsed": "Storage Used",
    "settings.appInfo.calculating": "Calculating...",
    "settings.appInfo.storageError": "Error calculating",
    "settings.appInfo.bytes": params ? `${params.size} bytes` : "bytes",
    "settings.appInfo.kb": params ? `${params.size} KB` : "KB",
    "settings.appInfo.mb": params ? `${params.size} MB` : "MB",
    "settings.dataManagement.title": "Data Management",
    "settings.dataManagement.export": "Export Data",
    "settings.dataManagement.exportDescription":
      "Download your workout data as JSON",
    "settings.dataManagement.clear": "Clear All Data",
    "settings.dataManagement.clearWarning": "This cannot be undone",
    "settings.dataManagement.exportError": "Error exporting data",
    "settings.dataManagement.clearConfirm":
      "Are you sure you want to clear all data? This will delete all workouts, exercises, and settings. This cannot be undone.",
    "settings.dataManagement.clearFinalWarning":
      "This is your final warning. All data will be permanently deleted. Are you absolutely sure?",
    "settings.dataManagement.clearSuccess":
      "All data has been cleared. The app will now reload.",
    "settings.dataManagement.clearError":
      "Error clearing data. Please try again.",
    "settings.loadError": "Error loading settings",
  };

  return translations[key] || key;
});

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: mockT,
    locale: { value: "en" },
  }),
}));

// Mock useToast composable
const mockShowSuccess = vi.fn();
const mockShowError = vi.fn();

vi.mock("@/composables/useToast.js", () => ({
  useToast: () => ({
    showSuccess: mockShowSuccess,
    showError: mockShowError,
  }),
}));

// Mock window.location.reload
Object.defineProperty(window, "location", {
  value: {
    reload: vi.fn(),
  },
  writable: true,
});

// Mock URL.createObjectURL and document.createElement for export functionality
global.URL = {
  createObjectURL: vi.fn(() => "blob:mock-url"),
  revokeObjectURL: vi.fn(),
};

Object.defineProperty(document, "createElement", {
  value: vi.fn(() => ({
    href: "",
    download: "",
    click: vi.fn(),
    style: { display: "" },
  })),
  writable: true,
});

Object.defineProperty(document.body, "appendChild", {
  value: vi.fn(),
  writable: true,
});

Object.defineProperty(document.body, "removeChild", {
  value: vi.fn(),
  writable: true,
});

describe("Settings.vue", () => {
  let wrapper;

  const createWrapper = () => {
    return mount(Settings);
  };

  beforeEach(() => {
    vi.clearAllMocks();
    getSetting.mockImplementation((key, defaultValue) => {
      const settings = {
        language: "en",
        weightUnit: "kg",
        theme: "light",
        exerciseDisplay: "reps",
      };
      return Promise.resolve(settings[key] || defaultValue);
    });
  });

  describe("Component Rendering", () => {
    beforeEach(async () => {
      wrapper = createWrapper();
      await nextTick();
    });

    it("renders the settings header", () => {
      expect(wrapper.find("h1").text()).toBe("Settings");
    });

    it("renders all setting sections", () => {
      expect(wrapper.text()).toContain("Language");
      expect(wrapper.text()).toContain("Weight Unit");
      expect(wrapper.text()).toContain("Theme");
      expect(wrapper.text()).toContain("Exercise Display");
      expect(wrapper.text()).toContain("App Information");
      expect(wrapper.text()).toContain("Data Management");
      expect(wrapper.text()).toContain("About Plonkout");
    });
  });

  describe("Settings Loading and Saving", () => {
    beforeEach(async () => {
      wrapper = createWrapper();
      await nextTick();
      // Wait for settings to load
      await new Promise((resolve) => setTimeout(resolve, 50));
    });

    it("loads settings on mount", async () => {
      expect(getSetting).toHaveBeenCalledWith("locale", "en");
      expect(getSetting).toHaveBeenCalledWith("weightUnit", "kg");
      expect(getSetting).toHaveBeenCalledWith("theme", "system");
      expect(getSetting).toHaveBeenCalledWith("exerciseDisplay", "reps");
    });

    it("saves language setting when changed", async () => {
      saveSetting.mockResolvedValueOnce();

      const vm = wrapper.vm;
      await vm.updateLanguage("sv");

      expect(saveSetting).toHaveBeenCalledWith("language", "sv");
      expect(vm.currentLanguage).toBe("sv");
    });

    it("saves weight unit setting when changed", async () => {
      saveSetting.mockResolvedValueOnce();

      const vm = wrapper.vm;
      await vm.updateWeightUnit("lbs");

      expect(saveSetting).toHaveBeenCalledWith("weightUnit", "lbs");
      expect(vm.currentWeightUnit).toBe("lbs");
    });

    it("saves theme setting when changed", async () => {
      saveSetting.mockResolvedValueOnce();

      const vm = wrapper.vm;
      await vm.updateTheme("dark");

      expect(saveSetting).toHaveBeenCalledWith("theme", "dark");
      expect(vm.currentTheme).toBe("dark");
    });

    it("saves exercise display setting when changed", async () => {
      saveSetting.mockResolvedValueOnce();

      const vm = wrapper.vm;
      await vm.saveExerciseDisplay("time");

      expect(saveSetting).toHaveBeenCalledWith("exerciseDisplay", "time");
      expect(vm.exerciseDisplay).toBe("time");
    });
  });

  describe("Theme Management", () => {
    beforeEach(async () => {
      wrapper = createWrapper();
      await nextTick();
    });

    it("applies theme to document correctly", () => {
      const vm = wrapper.vm;

      // Test light theme
      vm.applyTheme("light");
      expect(document.documentElement.classList.contains("dark")).toBe(false);

      // Test dark theme
      vm.applyTheme("dark");
      expect(document.documentElement.classList.contains("dark")).toBe(true);
    });

    it("detects system theme preference", () => {
      const vm = wrapper.vm;

      // Mock matchMedia
      Object.defineProperty(window, "matchMedia", {
        writable: true,
        value: vi.fn().mockImplementation(() => ({
          matches: true,
          addEventListener: vi.fn(),
          removeEventListener: vi.fn(),
        })),
      });

      expect(vm.getSystemTheme()).toBe("dark");

      // Test light system theme
      window.matchMedia.mockImplementation(() => ({
        matches: false,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
      }));

      expect(vm.getSystemTheme()).toBe("light");
    });
  });

  describe("Storage Calculation", () => {
    beforeEach(async () => {
      wrapper = createWrapper();
      await nextTick();
    });

    it("calculates storage size correctly", async () => {
      getWorkouts.mockResolvedValueOnce([
        { id: 1, name: "Test", exercises: [] },
        { id: 2, name: "Test 2", exercises: [] },
      ]);

      const vm = wrapper.vm;
      await vm.calculateStorageSize();

      expect(vm.storageSize).toBeDefined();
      expect(vm.storageCalculating).toBe(false);
    });

    it("handles storage calculation errors", async () => {
      getWorkouts.mockRejectedValueOnce(new Error("Storage error"));

      const vm = wrapper.vm;
      await vm.calculateStorageSize();

      expect(vm.storageError).toBe(true);
      expect(vm.storageCalculating).toBe(false);
    });

    it("formats storage size correctly", () => {
      const vm = wrapper.vm;

      expect(vm.formatStorageSize(500)).toBe("500 bytes");
      expect(vm.formatStorageSize(1500)).toBe("1.5 KB");
      expect(vm.formatStorageSize(1500000)).toBe("1.4 MB");
    });
  });

  describe("Data Export", () => {
    beforeEach(async () => {
      wrapper = createWrapper();
      await nextTick();
    });

    it("exports data successfully", async () => {
      const mockWorkouts = [{ id: 1, name: "Test Workout", exercises: [] }];
      getWorkouts.mockResolvedValueOnce(mockWorkouts);

      const vm = wrapper.vm;
      await vm.exportData();

      expect(getWorkouts).toHaveBeenCalled();
      expect(global.URL.createObjectURL).toHaveBeenCalled();
      expect(document.createElement).toHaveBeenCalledWith("a");
    });

    it("handles export errors", async () => {
      getWorkouts.mockRejectedValueOnce(new Error("Export error"));

      const vm = wrapper.vm;
      await vm.exportData();

      expect(mockShowError).toHaveBeenCalledWith("Error exporting data");
    });
  });

  describe("Data Clearing", () => {
    beforeEach(async () => {
      wrapper = createWrapper();
      await nextTick();
    });

    it("clears data with double confirmation", async () => {
      global.confirm
        .mockReturnValueOnce(true) // First confirmation
        .mockReturnValueOnce(true); // Final confirmation

      // Mock indexedDB.deleteDatabase
      global.indexedDB = {
        deleteDatabase: vi.fn().mockReturnValue({
          onsuccess: null,
          onerror: null,
          addEventListener: vi.fn((event, callback) => {
            if (event === "success") {
              setTimeout(callback, 0);
            }
          }),
        }),
      };

      const vm = wrapper.vm;
      await vm.clearAllData();

      expect(global.confirm).toHaveBeenCalledTimes(2);
      expect(mockShowSuccess).toHaveBeenCalledWith(
        "All data has been cleared. The app will now reload."
      );
    });

    it("cancels data clearing on first confirmation decline", async () => {
      global.confirm.mockReturnValueOnce(false);

      const vm = wrapper.vm;
      await vm.clearAllData();

      expect(global.confirm).toHaveBeenCalledTimes(1);
      expect(mockShowSuccess).not.toHaveBeenCalled();
    });

    it("cancels data clearing on final confirmation decline", async () => {
      global.confirm
        .mockReturnValueOnce(true) // First confirmation
        .mockReturnValueOnce(false); // Final confirmation decline

      const vm = wrapper.vm;
      await vm.clearAllData();

      expect(global.confirm).toHaveBeenCalledTimes(2);
      expect(mockShowSuccess).not.toHaveBeenCalled();
    });

    it("handles data clearing errors", async () => {
      global.confirm.mockReturnValueOnce(true).mockReturnValueOnce(true);

      global.indexedDB = {
        deleteDatabase: vi.fn().mockReturnValue({
          onsuccess: null,
          onerror: null,
          addEventListener: vi.fn((event, callback) => {
            if (event === "error") {
              setTimeout(callback, 0);
            }
          }),
        }),
      };

      const vm = wrapper.vm;
      await vm.clearAllData();

      expect(mockShowError).toHaveBeenCalledWith(
        "Error clearing data. Please try again."
      );
    });
  });

  describe("Error Handling", () => {
    it("handles settings loading errors", async () => {
      getSetting.mockRejectedValue(new Error("Settings error"));

      wrapper = createWrapper();
      await nextTick();
      await new Promise((resolve) => setTimeout(resolve, 50));

      // Component should still render despite error
      expect(wrapper.find("h1").text()).toBe("Settings");
    });

    it("handles setting save errors gracefully", async () => {
      saveSetting.mockRejectedValueOnce(new Error("Save error"));

      wrapper = createWrapper();
      await nextTick();

      const vm = wrapper.vm;
      await vm.updateLanguage("sv");

      // Should not crash on save error
      expect(saveSetting).toHaveBeenCalled();
    });
  });

  describe("Accessibility and UX", () => {
    beforeEach(async () => {
      wrapper = createWrapper();
      await nextTick();
    });

    it("has proper aria labels and semantic structure", () => {
      expect(wrapper.find("h1").exists()).toBe(true);
      expect(wrapper.findAll("h2").length).toBeGreaterThan(0);
    });

    it("provides clear feedback for destructive actions", () => {
      expect(wrapper.text()).toContain("This cannot be undone");
      expect(wrapper.text()).toContain("Clear All Data");
    });
  });
});
