import { ref, onMounted } from "vue";
import { getSetting } from "@/utils/database.js";

/**
 * Load the user's weight and distance unit settings.
 * @returns {{ weightUnit: import("vue").Ref<string>, distanceUnit: import("vue").Ref<string>, loadUnits: () => Promise<void> }}
 */
export function useUnits() {
  const weightUnit = ref("kg");
  const distanceUnit = ref("km");

  async function loadUnits() {
    try {
      const [weight, distance] = await Promise.all([
        getSetting("weightUnit", "kg"),
        getSetting("distanceUnit", "km"),
      ]);
      weightUnit.value = weight;
      distanceUnit.value = distance;
    } catch (error) {
      console.error("Error loading unit settings:", error);
    }
  }

  onMounted(loadUnits);

  return { weightUnit, distanceUnit, loadUnits };
}
