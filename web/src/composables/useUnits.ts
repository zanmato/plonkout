import { ref, onMounted } from "vue";
import { getSetting } from "@/utils/database";

/**
 * Load the user's weight and distance unit settings.
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
