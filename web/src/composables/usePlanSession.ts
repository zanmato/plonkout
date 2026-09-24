import { ref } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { startSession } from "@/api/plans";
import { useToast } from "@/composables/useToast";
import type { Id } from "@/types/domain";

/**
 * Starting a planned session from anywhere in the app: the server logs a
 * prefilled workout for it, which then opens in the editor.
 */
export function usePlanSession() {
  const router = useRouter();
  const { t } = useI18n();
  const { showError } = useToast();
  /** The session being started, so its button can show it. */
  const starting = ref<Id | null>(null);

  async function start(sessionId: Id): Promise<void> {
    if (starting.value) return;
    starting.value = sessionId;
    try {
      const workoutId = await startSession(sessionId);
      await router.push({ name: "workout-edit", params: { id: workoutId } });
    } catch (error) {
      console.error("Error starting planned session:", error);
      showError(t("plan.startError"));
    } finally {
      starting.value = null;
    }
  }

  return { start, starting };
}
